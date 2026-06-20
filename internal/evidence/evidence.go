package evidence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdul-hamid-achik/veclite"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

const (
	SchemaVersion     = "1"
	KeywordCollection = "evidence_entries_keyword"
)

type IndexOptions struct {
	BundleDir string
	DBPath    string
}

type IndexReport struct {
	OK              bool   `json:"ok"`
	BundleDir       string `json:"bundle_dir"`
	DBPath          string `json:"db_path"`
	Collection      string `json:"collection"`
	Mode            string `json:"mode"`
	IndexedEntries  int    `json:"indexed_entries"`
	InsertedEntries int    `json:"inserted_entries"`
	UpdatedEntries  int    `json:"updated_entries"`
	Summary         string `json:"summary"`
}

type SearchOptions struct {
	DBPath string
	Query  string
	Limit  int
}

type SearchReport struct {
	OK         bool           `json:"ok"`
	Query      string         `json:"query"`
	DBPath     string         `json:"db_path"`
	Collection string         `json:"collection"`
	Mode       string         `json:"mode"`
	Results    []SearchResult `json:"results"`
}

type SearchResult struct {
	Score         float32 `json:"score"`
	EvidenceID    string  `json:"evidence_id"`
	Bundle        string  `json:"bundle"`
	SourceVideo   string  `json:"source_video"`
	TimeSeconds   float64 `json:"time_seconds"`
	Frame         string  `json:"frame"`
	OCRPath       string  `json:"ocr_path"`
	OCR           string  `json:"ocr"`
	Transcript    string  `json:"transcript"`
	HasOCR        bool    `json:"has_ocr"`
	HasTranscript bool    `json:"has_transcript"`
}

type record struct {
	id      string
	content string
	payload map[string]any
}

func IndexBundle(opts IndexOptions) (IndexReport, error) {
	bundleDir, err := resolveRequiredPath(opts.BundleDir, "bundle")
	if err != nil {
		return IndexReport{}, err
	}
	dbPath, err := resolveRequiredPath(opts.DBPath, "db")
	if err != nil {
		return IndexReport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return IndexReport{}, fmt.Errorf("create evidence db directory: %w", err)
	}

	validation := bundle.Validate(bundleDir)
	if !validation.OK {
		return IndexReport{}, fmt.Errorf("bundle validation failed: %s", validation.Summary)
	}

	loaded, err := bundle.Load(bundleDir)
	if err != nil {
		return IndexReport{}, err
	}

	db, err := veclite.Open(dbPath)
	if err != nil {
		return IndexReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	coll, err := keywordCollection(db)
	if err != nil {
		return IndexReport{}, err
	}

	report := IndexReport{
		OK:         true,
		BundleDir:  loaded.Dir,
		DBPath:     dbPath,
		Collection: KeywordCollection,
		Mode:       "keyword",
	}
	for _, item := range recordsForBundle(loaded) {
		_, inserted, err := coll.UpsertTextDocumentByKey("evidence_id", item.id, item.content, item.payload)
		if err != nil {
			return IndexReport{}, fmt.Errorf("index evidence %s: %w", item.id, err)
		}
		report.IndexedEntries++
		if inserted {
			report.InsertedEntries++
		} else {
			report.UpdatedEntries++
		}
	}
	if err := db.Sync(); err != nil {
		return IndexReport{}, fmt.Errorf("sync evidence db: %w", err)
	}

	report.Summary = fmt.Sprintf("Indexed %d evidence entries into %s.", report.IndexedEntries, KeywordCollection)
	return report, nil
}

func Search(opts SearchOptions) (SearchReport, error) {
	dbPath, err := resolveRequiredPath(opts.DBPath, "db")
	if err != nil {
		return SearchReport{}, err
	}
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return SearchReport{}, fmt.Errorf("query is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	if _, err := os.Stat(dbPath); err != nil {
		return SearchReport{}, fmt.Errorf("evidence db not found: %s", dbPath)
	}

	db, err := veclite.Open(dbPath, veclite.WithReadOnly(true))
	if err != nil {
		return SearchReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	coll, err := db.GetCollection(KeywordCollection)
	if err != nil {
		return SearchReport{}, fmt.Errorf("evidence collection not found: %s", KeywordCollection)
	}

	results, err := coll.TextSearch(query, veclite.TopK(limit))
	if err != nil {
		return SearchReport{}, fmt.Errorf("search evidence: %w", err)
	}

	report := SearchReport{
		OK:         true,
		Query:      query,
		DBPath:     dbPath,
		Collection: KeywordCollection,
		Mode:       "keyword",
		Results:    make([]SearchResult, 0, len(results)),
	}
	for _, result := range results {
		report.Results = append(report.Results, searchResultFromPayload(result))
	}
	return report, nil
}

func keywordCollection(db *veclite.DB) (*veclite.Collection, error) {
	if db.HasCollection(KeywordCollection) {
		return db.GetCollection(KeywordCollection)
	}
	return db.CreateCollection(KeywordCollection,
		veclite.WithTextIndex("evidence_id", "bundle", "source_video", "frame", "ocr_path", "source"),
	)
}

func recordsForBundle(b bundle.Bundle) []record {
	records := make([]record, 0, len(b.Timeline.Entries))
	for _, entry := range b.Timeline.Entries {
		ocrText := strings.TrimSpace(entry.OCR.Text)
		transcriptText := transcriptForEntry(entry)
		id := evidenceID(b.Dir, entry)
		payload := map[string]any{
			"schema_version": SchemaVersion,
			"evidence_id":    id,
			"bundle":         b.Dir,
			"source_video":   b.Metadata.SourceVideo,
			"time_seconds":   entry.TimeSeconds,
			"source":         "timeline",
			"frame":          entry.Frame,
			"ocr_path":       entry.OCR.Path,
			"ocr":            ocrText,
			"transcript":     transcriptText,
			"has_ocr":        ocrText != "",
			"has_transcript": transcriptText != "",
		}
		records = append(records, record{
			id:      id,
			content: evidenceContent(entry, ocrText, transcriptText),
			payload: payload,
		})
	}
	return records
}

func evidenceContent(entry timeline.Entry, ocrText, transcriptText string) string {
	lines := []string{
		fmt.Sprintf("time: %.3f", entry.TimeSeconds),
		"frame: " + entry.Frame,
	}
	if strings.TrimSpace(entry.OCR.Path) != "" {
		lines = append(lines, "ocr_path: "+entry.OCR.Path)
	}
	if ocrText != "" {
		lines = append(lines, "ocr: "+ocrText)
	}
	if transcriptText != "" {
		lines = append(lines, "transcript: "+transcriptText)
	}
	return strings.Join(lines, "\n")
}

func transcriptForEntry(entry timeline.Entry) string {
	var parts []string
	seen := map[string]struct{}{}
	for _, segment := range entry.Transcript {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

func evidenceID(bundleDir string, entry timeline.Entry) string {
	return filepath.ToSlash(fmt.Sprintf("%s#%.3f#%s", bundleDir, entry.TimeSeconds, entry.Frame))
}

func searchResultFromPayload(result veclite.Result) SearchResult {
	payload := map[string]any{}
	if result.Record != nil && result.Record.Payload != nil {
		payload = result.Record.Payload
	}
	return SearchResult{
		Score:         result.Score,
		EvidenceID:    stringPayload(payload, "evidence_id"),
		Bundle:        stringPayload(payload, "bundle"),
		SourceVideo:   stringPayload(payload, "source_video"),
		TimeSeconds:   floatPayload(payload, "time_seconds"),
		Frame:         stringPayload(payload, "frame"),
		OCRPath:       stringPayload(payload, "ocr_path"),
		OCR:           stringPayload(payload, "ocr"),
		Transcript:    stringPayload(payload, "transcript"),
		HasOCR:        boolPayload(payload, "has_ocr"),
		HasTranscript: boolPayload(payload, "has_transcript"),
	}
}

func resolveRequiredPath(path, name string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s path is required", name)
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", name, err)
	}
	return resolved, nil
}

func stringPayload(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func boolPayload(payload map[string]any, key string) bool {
	value, _ := payload[key].(bool)
	return value
}

func floatPayload(payload map[string]any, key string) float64 {
	switch value := payload[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}
