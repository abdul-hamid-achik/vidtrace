package investigate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
)

type Options struct {
	BundleDir   string
	Query       string
	DBPath      string
	CodebaseDir string
	Limit       int
}

type Report struct {
	OK               bool                    `json:"ok"`
	Query            string                  `json:"query"`
	BundleDir        string                  `json:"bundle_dir"`
	DBPath           string                  `json:"db_path,omitempty"`
	TemporaryDB      bool                    `json:"temporary_db,omitempty"`
	CodebaseDir      string                  `json:"codebase_dir,omitempty"`
	Mode             string                  `json:"mode"`
	Evidence         []evidence.SearchResult `json:"evidence"`
	SuggestedQueries []string                `json:"suggested_queries"`
	VecgrepCommands  []string                `json:"vecgrep_commands,omitempty"`
	Summary          string                  `json:"summary"`
}

func Run(opts Options) (Report, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return Report{}, fmt.Errorf("query is required")
	}
	if strings.TrimSpace(opts.BundleDir) == "" {
		return Report{}, fmt.Errorf("bundle path is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 5
	}

	bundleDir, err := filepath.Abs(opts.BundleDir)
	if err != nil {
		return Report{}, fmt.Errorf("resolve bundle: %w", err)
	}

	dbPath := strings.TrimSpace(opts.DBPath)
	temporaryDB := false
	var cleanup func()
	if dbPath == "" {
		tmpDir, err := os.MkdirTemp("", "vidtrace-investigate-*")
		if err != nil {
			return Report{}, fmt.Errorf("create temporary evidence db: %w", err)
		}
		cleanup = func() { _ = os.RemoveAll(tmpDir) }
		defer cleanup()
		dbPath = filepath.Join(tmpDir, "evidence.veclite")
		temporaryDB = true
	} else {
		dbPath, err = filepath.Abs(dbPath)
		if err != nil {
			return Report{}, fmt.Errorf("resolve evidence db: %w", err)
		}
	}

	indexReport, err := evidence.IndexBundle(evidence.IndexOptions{
		BundleDir: bundleDir,
		DBPath:    dbPath,
	})
	if err != nil {
		return Report{}, err
	}

	searchReport, err := evidence.Search(evidence.SearchOptions{
		DBPath: dbPath,
		Query:  query,
		Limit:  limit,
	})
	if err != nil {
		return Report{}, err
	}

	codebaseDir := strings.TrimSpace(opts.CodebaseDir)
	if codebaseDir != "" {
		codebaseDir, err = filepath.Abs(codebaseDir)
		if err != nil {
			return Report{}, fmt.Errorf("resolve codebase: %w", err)
		}
	}

	suggested := SuggestedCodeQueries(query, searchReport.Results)
	reportDBPath := dbPath
	if temporaryDB {
		reportDBPath = ""
	}
	report := Report{
		OK:               true,
		Query:            query,
		BundleDir:        indexReport.BundleDir,
		DBPath:           reportDBPath,
		TemporaryDB:      temporaryDB,
		CodebaseDir:      codebaseDir,
		Mode:             searchReport.Mode,
		Evidence:         searchReport.Results,
		SuggestedQueries: suggested,
		VecgrepCommands:  VecgrepCommands(codebaseDir, suggested),
		Summary:          summary(searchReport.Results, suggested, codebaseDir),
	}
	return report, nil
}

func Markdown(report Report) string {
	var b strings.Builder
	writef(&b, "# Investigation Handoff\n\n")
	writef(&b, "%s\n\n", report.Summary)

	writef(&b, "## Video Evidence\n\n")
	if len(report.Evidence) == 0 {
		writef(&b, "- No matching timestamped evidence found for `%s`.\n", report.Query)
	} else {
		for _, item := range report.Evidence {
			text := firstNonEmpty(item.Transcript, item.OCR, item.Frame)
			writef(&b, "- %.2fs `%s` `%s`: %s\n", item.TimeSeconds, item.Frame, item.OCRPath, truncate(text, 180))
		}
	}

	writef(&b, "\n## Suggested Code Searches\n\n")
	if len(report.SuggestedQueries) == 0 {
		writef(&b, "- No suggested code searches.\n")
	} else {
		for _, query := range report.SuggestedQueries {
			writef(&b, "- `%s`\n", query)
		}
	}

	if len(report.VecgrepCommands) > 0 {
		writef(&b, "\n## Vecgrep Commands\n\n")
		writef(&b, "```bash\n")
		for _, command := range report.VecgrepCommands {
			writef(&b, "%s\n", command)
		}
		writef(&b, "```\n")
	} else {
		writef(&b, "\n## Vecgrep Commands\n\n")
		writef(&b, "- Pass `--codebase /path/to/repo` to include ready-to-run vecgrep commands.\n")
	}

	writef(&b, "\n## Notes\n\n")
	writef(&b, "- Start with the cited frame paths before changing code.\n")
	writef(&b, "- Use vecgrep for source-code search; vidtrace does not index source code.\n")
	return b.String()
}

func SuggestedCodeQueries(query string, results []evidence.SearchResult) []string {
	seen := map[string]struct{}{}
	var suggestions []string
	add := func(value string) {
		value = normalizePhrase(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, value)
	}

	add(query)
	for _, result := range results {
		for _, code := range codeLikeTokens(result.OCR + " " + result.Transcript) {
			add(code)
		}
		add(keywordPhrase(result.OCR, 6))
		add(keywordPhrase(result.Transcript, 8))
		if len(suggestions) >= 8 {
			break
		}
	}
	if len(suggestions) > 8 {
		suggestions = suggestions[:8]
	}
	return suggestions
}

func VecgrepCommands(codebaseDir string, queries []string) []string {
	if strings.TrimSpace(codebaseDir) == "" || len(queries) == 0 {
		return nil
	}
	commands := make([]string, 0, min(len(queries), 5))
	for _, query := range queries {
		commands = append(commands, fmt.Sprintf("cd %s && vecgrep search %s --format=json", shellQuote(codebaseDir), shellQuote(query)))
		if len(commands) >= 5 {
			break
		}
	}
	return commands
}

func summary(results []evidence.SearchResult, suggestions []string, codebaseDir string) string {
	codebase := "no codebase provided"
	if codebaseDir != "" {
		codebase = "vecgrep command suggestions included"
	}
	return fmt.Sprintf("Found %d video evidence hit(s) and %d suggested code search(es); %s.", len(results), len(suggestions), codebase)
}

func keywordPhrase(text string, limit int) string {
	words := meaningfulWords(text)
	if len(words) == 0 {
		return ""
	}
	if len(words) > limit {
		words = words[:limit]
	}
	return strings.Join(words, " ")
}

func meaningfulWords(text string) []string {
	matches := wordPattern.FindAllString(text, -1)
	seen := map[string]struct{}{}
	var words []string
	for _, match := range matches {
		word := strings.Trim(strings.ToLower(match), "_-.")
		if len(word) < 3 {
			continue
		}
		if _, ok := stopWords[word]; ok {
			continue
		}
		if _, ok := seen[word]; ok {
			continue
		}
		seen[word] = struct{}{}
		words = append(words, word)
	}
	return words
}

func codeLikeTokens(text string) []string {
	matches := wordPattern.FindAllString(text, -1)
	seen := map[string]struct{}{}
	var values []string
	for _, match := range matches {
		match = strings.Trim(match, "_-.")
		if len(match) < 4 || !hasLetter(match) || !hasDigit(match) {
			continue
		}
		key := strings.ToLower(match)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		values = append(values, match)
	}
	sort.Strings(values)
	return values
}

func normalizePhrase(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, " .,:;")
	return value
}

func hasLetter(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func hasDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncate(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:limit]
	}
	return value[:limit-1] + "..."
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func writef(b *strings.Builder, format string, args ...any) {
	_, _ = fmt.Fprintf(b, format, args...)
}

var wordPattern = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9._/-]*`)

var stopWords = map[string]struct{}{
	"about": {}, "after": {}, "also": {}, "and": {}, "are": {}, "but": {}, "can": {}, "cannot": {},
	"click": {}, "clicked": {}, "does": {}, "for": {}, "from": {}, "has": {}, "have": {}, "here": {},
	"into": {}, "not": {}, "now": {}, "one": {}, "the": {}, "then": {}, "this": {}, "that": {},
	"when": {}, "with": {}, "without": {}, "work": {}, "works": {}, "you": {},
}
