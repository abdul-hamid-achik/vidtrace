package investigate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
	"github.com/abdul-hamid-achik/vidtrace/internal/fcheap"
)

type Options struct {
	BundleDir   string
	Query       string
	DBPath      string
	CodebaseDir string
	Limit       int
	// Connect enables fcheap connect to run vecgrep over the codebase and
	// return real file:line code matches. Requires CodebaseDir.
	Connect bool
	// StashID restores a stashed bundle from fcheap before investigation.
	// When set, BundleDir is ignored (the restored stash is used instead).
	StashID string
	// ConnectMode controls the vecgrep search mode: semantic, keyword, or
	// hybrid. Empty uses vecgrep's default (hybrid).
	ConnectMode string
	// ConnectLimit caps the number of code matches returned.
	ConnectLimit int
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
	StashID          string                  `json:"stash_id,omitempty"`
	CodeMatches      []fcheap.CodeMatch      `json:"code_matches,omitempty"`
	ConnectError     string                  `json:"connect_error,omitempty"`
}

func Run(opts Options) (Report, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return Report{}, fmt.Errorf("query is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 5
	}

	var stashID string
	bundleDir := strings.TrimSpace(opts.BundleDir)

	// If a stash ID is provided, restore the bundle from fcheap first.
	if strings.TrimSpace(opts.StashID) != "" {
		stashID = strings.TrimSpace(opts.StashID)
		if !fcheap.Available() {
			return Report{}, fmt.Errorf("fcheap is not installed; cannot restore stash %s", stashID)
		}
		restoreCtx, restoreCancel := context.WithTimeout(context.Background(), 60*time.Second)
		restored, err := fcheap.Restore(restoreCtx, stashID, "")
		restoreCancel()
		if err != nil {
			return Report{}, fmt.Errorf("restore stash %s: %w", stashID, err)
		}
		bundleDir = restored
	}

	if bundleDir == "" {
		return Report{}, fmt.Errorf("bundle path is required")
	}

	absBundleDir, err := filepath.Abs(bundleDir)
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
		BundleDir: absBundleDir,
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
	// If --connect is set with a codebase, run fcheap connect to get real
	// code matches. This is a best-effort enhancement: failures are recorded
	// in ConnectError rather than aborting the report.
	var codeMatches []fcheap.CodeMatch
	var connectError string
	if opts.Connect && codebaseDir != "" {
		codeMatches, connectError = runConnect(absBundleDir, stashID, codebaseDir, query, opts)
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
		Summary:          summary(searchReport.Results, suggested, codebaseDir, codeMatches),
		StashID:          stashID,
		CodeMatches:      codeMatches,
		ConnectError:     connectError,
	}

	return report, nil
}

// runConnect calls fcheap connect to run vecgrep over the codebase using the
// stashed or local bundle's text. It returns code matches and an error string
// (empty on success). If no stash ID is provided, the bundle is saved to fcheap
// first as a temporary stash, then connected. If fcheap is not available, returns
// a descriptive error without panicking.
//
// Note: when a temporary stash is created (no explicit --stash), it persists in
// the fcheap vault after the connect call. Callers who want a clean vault should
// drop it via `fcheap drop <id> --force` or use --stash to provide a pre-stashed
// bundle.
func runConnect(absBundleDir, stashID, codebaseDir, query string, opts Options) ([]fcheap.CodeMatch, string) {
	if !fcheap.Available() {
		return nil, "fcheap is not installed or not on PATH"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// If no stash ID is provided, save the bundle to fcheap first so we can
	// use fcheap connect. The stash is temporary and only used for the
	// connect call.
	effectiveStashID := stashID
	if effectiveStashID == "" {
		saveResult, err := fcheap.Save(ctx, absBundleDir, "vidtrace-investigate", "vidtrace", nil)
		if err != nil {
			return nil, fmt.Sprintf("stash bundle for connect: %s", err)
		}
		effectiveStashID = saveResult.ID
	}

	connectLimit := opts.ConnectLimit
	if connectLimit <= 0 {
		connectLimit = 10
	}

	result, err := fcheap.Connect(ctx, fcheap.ConnectOptions{
		StashID:     effectiveStashID,
		CodebaseDir: codebaseDir,
		Query:       query,
		Mode:        opts.ConnectMode,
		Limit:       connectLimit,
		Index:       true,
	})
	if err != nil {
		return nil, err.Error()
	}

	return result.Matches, ""
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

	if report.StashID != "" {
		writef(&b, "\n## Stash\n\n")
		writef(&b, "- Restored from fcheap stash: `%s`\n", report.StashID)
	}

	if len(report.CodeMatches) > 0 {
		writef(&b, "\n## Code Matches\n\n")
		for _, match := range report.CodeMatches {
			writef(&b, "- `%s` (score %.4f): %s\n", match.File, match.Score, truncate(match.Text, 120))
		}
	}

	if report.ConnectError != "" {
		writef(&b, "\n## Connect Error\n\n")
		writef(&b, "- %s\n", report.ConnectError)
	}

	writef(&b, "\n## Notes\n\n")
	writef(&b, "- Start with the cited frame paths before changing code.\n")
	writef(&b, "- Use vecgrep for source-code search; vidtrace does not index source code.\n")
	if len(report.CodeMatches) > 0 {
		writef(&b, "- Code matches were found via fcheap connect (vecgrep over the codebase).\n")
	}
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

func summary(results []evidence.SearchResult, suggestions []string, codebaseDir string, codeMatches []fcheap.CodeMatch) string {
	codebase := "no codebase provided"
	if codebaseDir != "" {
		codebase = "vecgrep command suggestions included"
	}
	suffix := ""
	if len(codeMatches) > 0 {
		suffix = fmt.Sprintf("; %d code match(es) found via fcheap connect", len(codeMatches))
	}
	return fmt.Sprintf("Found %d video evidence hit(s) and %d suggested code search(es); %s.%s", len(results), len(suggestions), codebase, suffix)
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
		if isNoiseWord(word) {
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

// isNoiseWord reports whether a lowercased OCR/transcript word is dense-UI noise
// that should not become a code-search suggestion: stop words, browser/OS chrome,
// calendar words (month and day names), four-digit years, and URLs or domains.
// Code-like tokens (letters and digits, for example ticket IDs) bypass this path.
func isNoiseWord(word string) bool {
	if _, ok := stopWords[word]; ok {
		return true
	}
	if _, ok := chromeWords[word]; ok {
		return true
	}
	if _, ok := dateWords[word]; ok {
		return true
	}
	if isYearLike(word) {
		return true
	}
	return isURLLike(word)
}

func isYearLike(word string) bool {
	if len(word) != 4 {
		return false
	}
	for _, r := range word {
		if r < '0' || r > '9' {
			return false
		}
	}
	return strings.HasPrefix(word, "19") || strings.HasPrefix(word, "20")
}

func isURLLike(word string) bool {
	if strings.Contains(word, "://") {
		return true
	}
	return domainPattern.MatchString(word)
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

// domainPattern matches host/domain tokens such as "example.com" or
// "app.example.com/path" so browser address-bar noise is dropped from
// suggestions while route-like path words are still allowed through other tokens.
var domainPattern = regexp.MustCompile(`^([a-z0-9-]+\.)+(com|org|net|io|dev|app|co|gov|edu|info|xyz)(/.*)?$`)

var stopWords = map[string]struct{}{
	"about": {}, "after": {}, "also": {}, "and": {}, "are": {}, "but": {}, "can": {}, "cannot": {},
	"click": {}, "clicked": {}, "does": {}, "for": {}, "from": {}, "has": {}, "have": {}, "here": {},
	"into": {}, "not": {}, "now": {}, "one": {}, "the": {}, "then": {}, "this": {}, "that": {},
	"when": {}, "with": {}, "without": {}, "work": {}, "works": {}, "you": {},
}

// chromeWords are browser and OS chrome tokens that frequently appear in dense UI
// captures but rarely describe the application bug being investigated.
var chromeWords = map[string]struct{}{
	"http": {}, "https": {}, "www": {}, "localhost": {}, "chrome": {}, "firefox": {},
	"safari": {}, "mozilla": {}, "webkit": {}, "bookmarks": {}, "bookmark": {}, "reload": {},
	"refresh": {}, "newtab": {}, "untitled": {}, "devtools": {}, "incognito": {}, "favicon": {},
}

// dateWords are month and day names (full and common abbreviations) that show up
// in clocks, calendars, and timestamps rather than in bug-relevant UI text.
var dateWords = map[string]struct{}{
	"january": {}, "february": {}, "march": {}, "april": {}, "may": {}, "june": {},
	"july": {}, "august": {}, "september": {}, "october": {}, "november": {}, "december": {},
	"jan": {}, "feb": {}, "mar": {}, "apr": {}, "jun": {}, "jul": {}, "aug": {},
	"sep": {}, "sept": {}, "oct": {}, "nov": {}, "dec": {},
	"monday": {}, "tuesday": {}, "wednesday": {}, "thursday": {}, "friday": {}, "saturday": {}, "sunday": {},
	"mon": {}, "tue": {}, "tues": {}, "wed": {}, "thu": {}, "thur": {}, "thurs": {}, "fri": {}, "sat": {}, "sun": {},
}
