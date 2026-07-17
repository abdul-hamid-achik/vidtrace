package analysis

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

type Options struct {
	BundleDir  string
	TicketPath string
	// Mode is "keyword" (default) for classic term matching, or "hybrid" to
	// also rank ticket text against timeline evidence via a temporary VecLite
	// index. Hybrid still reports term hits; semantic ranking only affects the
	// ordered Evidence list and can raise confidence when strong hits exist.
	Mode string
}

type Result struct {
	OK              bool          `json:"ok"`
	Status          string        `json:"status"`
	Coverage        string        `json:"coverage"` // unknown|no_observation|inconclusive|supported|contradicted (SPEC §8.4)
	Confidence      string        `json:"confidence"`
	Score           float64       `json:"score"`
	BundleDir       string        `json:"bundle_dir"`
	TicketPath      string        `json:"ticket_path"`
	MatchedTerms    []string      `json:"matched_terms"`
	MissingTerms    []string      `json:"missing_terms"`
	TermHits        []TermHit     `json:"term_hits"`
	Evidence        []EvidenceRef `json:"evidence"`
	Summary         string        `json:"summary"`
	Gaps            []string      `json:"gaps"`
	Degraded        bool          `json:"degraded,omitempty"`
	Hint            string        `json:"hint,omitempty"`
	SuggestedAction string        `json:"suggested_action,omitempty"`
}

type EvidenceRef struct {
	TimeSeconds float64 `json:"time_seconds"`
	Frame       string  `json:"frame"`
	OCRPath     string  `json:"ocr_path"`
	Text        string  `json:"text"`
}

type TermHit struct {
	Term        string  `json:"term"`
	Source      string  `json:"source"`
	TimeSeconds float64 `json:"time_seconds"`
	Frame       string  `json:"frame"`
	OCRPath     string  `json:"ocr_path,omitempty"`
	Text        string  `json:"text"`
}

func Compare(opts Options) (Result, error) {
	if strings.TrimSpace(opts.BundleDir) == "" {
		return Result{}, fmt.Errorf("bundle path is required")
	}
	if strings.TrimSpace(opts.TicketPath) == "" {
		return Result{}, fmt.Errorf("ticket path is required")
	}

	doc, err := bundle.Load(opts.BundleDir)
	if err != nil {
		return Result{}, err
	}

	ticketPath, err := filepath.Abs(opts.TicketPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve ticket: %w", err)
	}
	ticketData, err := os.ReadFile(ticketPath)
	if err != nil {
		return Result{}, fmt.Errorf("read ticket: %w", err)
	}

	ticketText := string(ticketData)
	terms := keywords(ticketText)
	searchable := doc.SearchableText()
	evidenceText := newTextIndex(searchable)
	var matched []string
	var missing []string
	for _, term := range terms {
		if evidenceText.Contains(term) {
			matched = append(matched, term)
		} else {
			missing = append(missing, term)
		}
	}

	score := 0.0
	if len(terms) > 0 {
		score = float64(len(matched)) / float64(len(terms))
	}
	status := classify(len(terms), len(matched), score)
	if status == "supported" && detectContradiction(ticketText, searchable) {
		status = "contradicted"
	}
	termHits := findTermHits(doc, matched, 12)
	evidence := findEvidence(doc, matched, 5)
	if strings.EqualFold(strings.TrimSpace(opts.Mode), "hybrid") {
		if hybridEvidence, ok := hybridEvidenceFromTicket(doc.Dir, ticketText, 5); ok && len(hybridEvidence) > 0 {
			evidence = hybridEvidence
			// A strong hybrid hit with at least one term match raises thin
			// inconclusive results toward supported without inventing terms.
			if status == "inconclusive" && len(matched) > 0 {
				status = "supported"
				score = math.Max(score, 0.4)
			}
		}
	}
	confidence := classifyConfidence(status, score, len(termHits))
	gapList := gaps(status, terms, evidence)

	result := Result{
		OK:              true,
		Status:          status,
		Coverage:        status,
		Confidence:      confidence,
		Score:           math.Round(score*1000) / 1000,
		BundleDir:       doc.Dir,
		TicketPath:      ticketPath,
		MatchedTerms:    nonNilStrings(matched),
		MissingTerms:    nonNilStrings(missing),
		TermHits:        nonNilTermHits(termHits),
		Evidence:        nonNilEvidence(evidence),
		Summary:         summary(status, confidence, matched, missing, evidence),
		Degraded:        status == "unknown" || status == "no_observation",
		Hint:            hintForStatus(status),
		SuggestedAction: actionForStatus(status),
		Gaps:            nonNilStrings(gapList),
	}
	return result, nil
}

// hybridEvidenceFromTicket indexes the bundle into a temp DB and searches with
// the full ticket text (keyword mode). Failures return ok=false so compare
// still succeeds with classic term evidence.
func hybridEvidenceFromTicket(bundleDir, ticketText string, limit int) ([]EvidenceRef, bool) {
	tmpDir, err := os.MkdirTemp("", "vidtrace-compare-hybrid-*")
	if err != nil {
		return nil, false
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	dbPath := filepath.Join(tmpDir, "evidence.veclite")

	if _, err := evidence.IndexBundle(evidence.IndexOptions{
		BundleDir: bundleDir,
		DBPath:    dbPath,
	}); err != nil {
		return nil, false
	}
	report, err := evidence.Search(evidence.SearchOptions{
		DBPath: dbPath,
		Query:  ticketText,
		Limit:  limit,
		Mode:   evidence.ModeKeyword,
	})
	if err != nil || len(report.Results) == 0 {
		return nil, false
	}
	refs := make([]EvidenceRef, 0, len(report.Results))
	for _, r := range report.Results {
		text := strings.TrimSpace(r.Transcript)
		if text == "" {
			text = strings.TrimSpace(r.OCR)
		}
		refs = append(refs, EvidenceRef{
			TimeSeconds: r.TimeSeconds,
			Frame:       r.Frame,
			OCRPath:     r.OCRPath,
			Text:        truncateSingleLine(text, 180),
		})
	}
	return refs, true
}

func Markdown(result Result) string {
	var b strings.Builder
	writef(&b, "## Summary\n\n%s\n\n", result.Summary)
	writef(&b, "## Ticket Match\n\nStatus: %s\n\nConfidence: %s\n\nScore: %.3f\n\n", result.Status, result.Confidence, result.Score)

	writef(&b, "Matched terms: %s\n\n", listOrNone(result.MatchedTerms))
	writef(&b, "Missing terms: %s\n\n", listOrNone(result.MissingTerms))

	writef(&b, "## Term Hits\n\n")
	if len(result.TermHits) == 0 {
		writef(&b, "- No term-level hits were found.\n")
	} else {
		for _, hit := range result.TermHits {
			writef(&b, "- `%s` in %s at %.2fs `%s`: %s\n", hit.Term, hit.Source, hit.TimeSeconds, hit.Frame, hit.Text)
		}
	}

	writef(&b, "\n## Evidence\n\n")
	if len(result.Evidence) == 0 {
		writef(&b, "- No direct OCR or transcript evidence matched the ticket terms.\n")
	} else {
		for _, item := range result.Evidence {
			writef(&b, "- %.2fs `%s` `%s`: %s\n", item.TimeSeconds, item.Frame, item.OCRPath, item.Text)
		}
	}

	writef(&b, "\n## Reproduction Notes\n\n")
	if len(result.Evidence) == 0 {
		writef(&b, "- Inspect `timeline.json`, `ocr/ocr_all_frames.txt`, and selected frames manually.\n")
	} else {
		writef(&b, "- Start with the evidence timestamps above, then open the referenced frames when visual confirmation is needed.\n")
	}

	writef(&b, "\n## Gaps\n\n")
	for _, gap := range result.Gaps {
		writef(&b, "- %s\n", gap)
	}
	return b.String()
}

func keywords(text string) []string {
	matches := wordPattern.FindAllString(strings.ToLower(text), -1)
	seen := make(map[string]struct{})
	var terms []string
	for _, match := range matches {
		for _, term := range keywordCandidates(match) {
			if len(term) < 3 {
				continue
			}
			if _, ok := stopWords[term]; ok {
				continue
			}
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms
}

var wordPattern = regexp.MustCompile(`[a-z0-9][a-z0-9._:/-]*`)
var splitPattern = regexp.MustCompile(`[._:/-]+`)
var nonAlphaNumericPattern = regexp.MustCompile(`[^a-z0-9]+`)

var stopWords = map[string]struct{}{
	"about": {}, "after": {}, "also": {}, "and": {}, "are": {}, "but": {}, "can": {}, "cannot": {},
	"does": {}, "for": {}, "from": {}, "has": {}, "have": {}, "into": {}, "not": {}, "now": {},
	"one": {}, "the": {}, "then": {}, "this": {}, "that": {}, "when": {}, "with": {}, "without": {}, "you": {},
	"una": {}, "con": {}, "del": {}, "los": {}, "las": {}, "para": {}, "por": {}, "que": {},
}

// classify returns a coverage-aware evidence status (SPEC §8.4):
//   unknown        — no evidence was collected (no terms to search for).
//   no_observation — evidence was collected but the terms don't appear.
//   inconclusive   — some terms matched but below the support threshold.
//   supported      — enough terms matched to support the claim.
//   contradicted   — evidence contradicts the claim (set by a future heuristic).

// hintForStatus returns a human-readable hint for the coverage status.
func hintForStatus(status string) string {
	switch status {
	case "unknown":
		return "no usable evidence was collected — the OCR/transcript pipeline may not have run or produced no text. Run vidtrace extract on the video bundle to generate evidence."
	case "no_observation":
		return "evidence was collected but none of the ticket terms appear in the OCR or transcript. The bug may not be visible in this recording, or the terms may need broadening."
	case "inconclusive":
		return "some terms matched but not enough to support the claim. Review the partial matches + consider whether the ticket description is accurate."
	case "contradicted":
		return "the evidence appears to contradict the claim. Review the matched evidence carefully — it may show the opposite of what the ticket describes."
	default:
		return ""
	}
}

// actionForStatus returns a suggested action for the coverage status.
func actionForStatus(status string) string {
	switch status {
	case "unknown":
		return "vidtrace extract <video.mp4> --json"
	case "no_observation":
		return "vidtrace investigate <bundle> --query \"<broader ticket terms>\" --json  (or search with --mode hybrid when a semantic index exists)"
	case "inconclusive":
		return "review the partial term_hits + evidence fields, then decide whether to investigate further or close the ticket"
	case "contradicted":
		return "review the matched evidence carefully — the recording may show the opposite of the ticket claim"
	default:
		return ""
	}
}

func classify(totalTerms, matchedTerms int, score float64) string {
	switch {
	case totalTerms == 0:
		return "unknown"
	case matchedTerms == 0:
		return "no_observation"
	case matchedTerms >= 3 || score >= 0.35:
		return "supported"
	default:
		return "inconclusive"
	}
}

func classifyConfidence(status string, score float64, termHits int) string {
	switch {
	case status == "supported" && score >= 0.6 && termHits >= 2:
		return "high"
	case status == "supported":
		return "medium"
	case status == "contradicted":
		return "medium"
	case status == "inconclusive" && termHits > 0:
		return "low"
	case status == "no_observation":
		return "low"
	case status == "unknown":
		return "none"
	default:
		return "low"
	}
}

func keywordCandidates(match string) []string {
	match = strings.Trim(match, "._-:/")
	if match == "" {
		return nil
	}

	var candidates []string
	parts := splitPattern.Split(match, -1)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			candidates = append(candidates, part)
		}
	}

	if len(parts) > 1 {
		compact := strings.Join(parts, "")
		if len(compact) >= 5 {
			candidates = append(candidates, compact)
		}
	}

	if len(candidates) == 0 {
		candidates = append(candidates, match)
	}
	return candidates
}

func findEvidence(doc bundle.Bundle, terms []string, limit int) []EvidenceRef {
	if len(terms) == 0 || limit <= 0 {
		return nil
	}

	var refs []EvidenceRef
	for _, entry := range doc.Timeline.Entries {
		text := entryEvidenceText(entry)
		index := newTextIndex(text)
		for _, term := range terms {
			if index.Contains(term) {
				refs = append(refs, EvidenceRef{
					TimeSeconds: entry.TimeSeconds,
					Frame:       entry.Frame,
					OCRPath:     entry.OCR.Path,
					Text:        truncateSingleLine(text, 180),
				})
				break
			}
		}
		if len(refs) >= limit {
			return refs
		}
	}
	return refs
}

func findTermHits(doc bundle.Bundle, terms []string, limit int) []TermHit {
	if len(terms) == 0 || limit <= 0 {
		return nil
	}

	var hits []TermHit
	seen := make(map[string]struct{})
	for _, entry := range doc.Timeline.Entries {
		addHits := func(source, text string) bool {
			index := newTextIndex(text)
			for _, term := range terms {
				key := fmt.Sprintf("%s:%s:%.3f:%s", term, source, entry.TimeSeconds, entry.Frame)
				if _, ok := seen[key]; ok || !index.Contains(term) {
					continue
				}
				seen[key] = struct{}{}
				hits = append(hits, TermHit{
					Term:        term,
					Source:      source,
					TimeSeconds: entry.TimeSeconds,
					Frame:       entry.Frame,
					OCRPath:     entry.OCR.Path,
					Text:        truncateSingleLine(text, 140),
				})
				if len(hits) >= limit {
					return true
				}
			}
			return false
		}

		if addHits("ocr", entry.OCR.Text) {
			return hits
		}
		for _, segment := range entry.Transcript {
			if addHits("transcript", segment.Text) {
				return hits
			}
		}
	}
	return hits
}

func entryEvidenceText(entry timeline.Entry) string {
	var parts []string
	parts = append(parts, entry.OCR.Text)
	for _, segment := range entry.Transcript {
		parts = append(parts, segment.Text)
	}
	return strings.Join(parts, " ")
}

func summary(status, confidence string, matched, missing []string, evidence []EvidenceRef) string {
	switch status {
	case "supported":
		return fmt.Sprintf("The ticket appears to match the video evidence with %s confidence. %d term(s) matched across OCR/transcript evidence.", confidence, len(matched))
	case "contradicted":
		return "The ticket claims a failure, but the extracted evidence mostly shows success states without matching failure language. Review frames carefully."
	case "no_observation":
		return "The ticket does not appear to match the extracted video evidence; no meaningful ticket terms were found."
	case "unknown":
		return "No usable ticket terms or evidence were available for comparison."
	default:
		if len(evidence) == 0 {
			return "The ticket/video relationship is inconclusive; no direct timeline evidence matched the ticket terms."
		}
		return "The ticket/video relationship is inconclusive; some terms matched but the evidence is too thin for a confident match."
	}
}

func gaps(status string, terms []string, evidence []EvidenceRef) []string {
	var gaps []string
	if len(terms) == 0 {
		gaps = append(gaps, "Ticket text did not contain enough searchable terms.")
	}
	if len(evidence) == 0 {
		gaps = append(gaps, "No direct timeline evidence matched the ticket terms.")
	}
	if status == "contradicted" {
		gaps = append(gaps, "Failure language in the ticket is not reflected in OCR/transcript evidence.")
	}
	if status != "supported" {
		gaps = append(gaps, "This is a heuristic text comparison; inspect referenced frames before closing the ticket.")
	}
	return gaps
}

var failureMarkers = []string{
	"fail", "failed", "failure", "broken", "cannot", "can't", "does not", "doesn't",
	"error", "bug", "crash", "stuck", "unable", "not working", "doesn't work", "does not work",
}

var successMarkers = []string{
	"success", "successfully", "saved", "completed", "done", "works", "working", "submitted",
}

// detectContradiction reports when a failure-oriented ticket is paired with
// success-oriented evidence that lacks those failure markers. Shared UI nouns
// may still match, so this only fires after classify would otherwise say supported.
func detectContradiction(ticket, evidence string) bool {
	ticketLower := strings.ToLower(ticket)
	evidenceLower := strings.ToLower(evidence)
	if !containsAny(ticketLower, failureMarkers) {
		return false
	}
	if !containsAny(evidenceLower, successMarkers) {
		return false
	}
	return !containsAny(evidenceLower, failureMarkers)
}

func containsAny(text string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func listOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func truncateSingleLine(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	if limit <= 3 {
		return text[:limit]
	}
	return text[:limit-3] + "..."
}

type textIndex struct {
	normalized string
	compact    string
}

func newTextIndex(text string) textIndex {
	normalized := normalizeSearchText(text)
	return textIndex{
		normalized: normalized,
		compact:    strings.ReplaceAll(normalized, " ", ""),
	}
}

func normalizeSearchText(text string) string {
	text = strings.ToLower(text)
	text = nonAlphaNumericPattern.ReplaceAllString(text, " ")
	return strings.Join(strings.Fields(text), " ")
}

func (idx textIndex) Contains(term string) bool {
	if term == "" {
		return false
	}
	term = normalizeSearchText(term)
	if term == "" {
		return false
	}
	if strings.Contains(" "+idx.normalized+" ", " "+term+" ") {
		return true
	}
	if len(term) >= 5 && strings.Contains(idx.compact, strings.ReplaceAll(term, " ", "")) {
		return true
	}
	return false
}

func writef(b *strings.Builder, format string, args ...any) {
	_, _ = fmt.Fprintf(b, format, args...)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilEvidence(values []EvidenceRef) []EvidenceRef {
	if values == nil {
		return []EvidenceRef{}
	}
	return values
}

func nonNilTermHits(values []TermHit) []TermHit {
	if values == nil {
		return []TermHit{}
	}
	return values
}
