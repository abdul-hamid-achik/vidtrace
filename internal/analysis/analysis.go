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
	Score        float64       `json:"score"`
	BundleDir    string        `json:"bundle_dir"`
	TicketPath   string        `json:"ticket_path"`
	MatchedTerms []string      `json:"matched_terms"`
	MissingTerms []string      `json:"missing_terms"`
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
	evidenceText := strings.ToLower(doc.SearchableText())
	var matched []string
	var missing []string
	for _, term := range terms {
		if strings.Contains(evidenceText, term) {
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
	evidence := findEvidence(doc, matched, 5)
	gapList := gaps(status, terms, evidence)

	result := Result{
		OK:           true,
		Status:       status,
		Score:        math.Round(score*1000) / 1000,
		BundleDir:    doc.Dir,
		TicketPath:   ticketPath,
		MatchedTerms: nonNilStrings(matched),
		MissingTerms: nonNilStrings(missing),
		Evidence:     nonNilEvidence(evidence),
		Summary:      summary(status, matched, missing, evidence),
		Gaps:         nonNilStrings(gapList),
	}
	return result, nil
}

func Markdown(result Result) string {
	var b strings.Builder
	writef(&b, "## Summary\n\n%s\n\n", result.Summary)
	writef(&b, "## Ticket Match\n\nStatus: %s\n\nScore: %.3f\n\n", result.Status, result.Score)

	writef(&b, "Matched terms: %s\n\n", listOrNone(result.MatchedTerms))
	writef(&b, "Missing terms: %s\n\n", listOrNone(result.MissingTerms))

	writef(&b, "## Evidence\n\n")
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
		term := strings.Trim(match, "._-:/")
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
	sort.Strings(terms)
	return terms
}

var wordPattern = regexp.MustCompile(`[a-z0-9][a-z0-9._:/-]*`)

var stopWords = map[string]struct{}{
	"about": {}, "after": {}, "also": {}, "and": {}, "are": {}, "but": {}, "can": {}, "cannot": {},
	"does": {}, "for": {}, "from": {}, "has": {}, "have": {}, "into": {}, "not": {}, "now": {},
	"the": {}, "then": {}, "this": {}, "that": {}, "when": {}, "with": {}, "without": {}, "you": {},
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

func findEvidence(doc bundle.Bundle, terms []string, limit int) []EvidenceRef {
	if len(terms) == 0 || limit <= 0 {
		return nil
	}

	var refs []EvidenceRef
	for _, entry := range doc.Timeline.Entries {
		text := entryEvidenceText(entry)
		lower := strings.ToLower(text)
		for _, term := range terms {
			if strings.Contains(lower, term) {
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

func entryEvidenceText(entry timeline.Entry) string {
	var parts []string
	parts = append(parts, entry.OCR.Text)
	for _, segment := range entry.Transcript {
		parts = append(parts, segment.Text)
	}
	return strings.Join(parts, " ")
}

func summary(status string, matched, missing []string, evidence []EvidenceRef) string {
	switch status {
	case "match":
		return fmt.Sprintf("The ticket appears to match the video evidence. %d term(s) matched across OCR/transcript evidence.", len(matched))
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
