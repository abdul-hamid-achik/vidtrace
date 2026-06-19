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
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

type Options struct {
	BundleDir  string
	TicketPath string
}

type Result struct {
	OK           bool          `json:"ok"`
	Status       string        `json:"status"`
	Confidence   string        `json:"confidence"`
	Score        float64       `json:"score"`
	BundleDir    string        `json:"bundle_dir"`
	TicketPath   string        `json:"ticket_path"`
	MatchedTerms []string      `json:"matched_terms"`
	MissingTerms []string      `json:"missing_terms"`
	TermHits     []TermHit     `json:"term_hits"`
	Evidence     []EvidenceRef `json:"evidence"`
	Summary      string        `json:"summary"`
	Gaps         []string      `json:"gaps"`
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

	terms := keywords(string(ticketData))
	evidenceText := newTextIndex(doc.SearchableText())
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
	termHits := findTermHits(doc, matched, 12)
	evidence := findEvidence(doc, matched, 5)
	confidence := classifyConfidence(status, score, len(termHits))
	gapList := gaps(status, terms, evidence)

	result := Result{
		OK:           true,
		Status:       status,
		Confidence:   confidence,
		Score:        math.Round(score*1000) / 1000,
		BundleDir:    doc.Dir,
		TicketPath:   ticketPath,
		MatchedTerms: nonNilStrings(matched),
		MissingTerms: nonNilStrings(missing),
		TermHits:     nonNilTermHits(termHits),
		Evidence:     nonNilEvidence(evidence),
		Summary:      summary(status, confidence, matched, missing, evidence),
		Gaps:         nonNilStrings(gapList),
	}
	return result, nil
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

func classify(totalTerms, matchedTerms int, score float64) string {
	switch {
	case totalTerms == 0:
		return "inconclusive"
	case matchedTerms == 0:
		return "mismatch"
	case matchedTerms >= 3 || score >= 0.35:
		return "match"
	default:
		return "inconclusive"
	}
}

func classifyConfidence(status string, score float64, termHits int) string {
	switch {
	case status == "match" && score >= 0.6 && termHits >= 2:
		return "high"
	case status == "match":
		return "medium"
	case termHits > 0:
		return "low"
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
	case "match":
		return fmt.Sprintf("The ticket appears to match the video evidence with %s confidence. %d term(s) matched across OCR/transcript evidence.", confidence, len(matched))
	case "mismatch":
		return "The ticket does not appear to match the extracted video evidence; no meaningful ticket terms were found."
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
	if status != "match" {
		gaps = append(gaps, "This is a heuristic text comparison; inspect referenced frames before closing the ticket.")
	}
	return gaps
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
