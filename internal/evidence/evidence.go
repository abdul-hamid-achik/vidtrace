package evidence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdul-hamid-achik/veclite"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/embed"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

const (
	SchemaVersion     = "1"
	KeywordCollection = "evidence_entries_keyword"
	// TextCollection holds vector + content records for semantic and hybrid
	// search; MetaCollection records the embedding profile used to build it.
	TextCollection = "evidence_entries_text"
	MetaCollection = "evidence_meta"

	embeddingProfileKey = "embedding_profile"
)

// Search modes.
const (
	ModeKeyword  = "keyword"
	ModeSemantic = "semantic"
	ModeHybrid   = "hybrid"
)

type IndexOptions struct {
	BundleDir string
	DBPath    string
	// Embedder, when set, also builds the semantic index (vector + content) and
	// records its embedding profile. When nil, only the BM25 keyword index is
	// built and the keyword contract is unchanged.
	Embedder embed.Embedder
}

type IndexReport struct {
	OK              bool           `json:"ok"`
	BundleDir       string         `json:"bundle_dir"`
	DBPath          string         `json:"db_path"`
	Collection      string         `json:"collection"`
	Mode            string         `json:"mode"`
	IndexedEntries  int            `json:"indexed_entries"`
	InsertedEntries int            `json:"inserted_entries"`
	UpdatedEntries  int            `json:"updated_entries"`
	SemanticEntries int            `json:"semantic_entries,omitempty"`
	Embedding       *embed.Profile `json:"embedding,omitempty"`
	Summary         string         `json:"summary"`
}

// BundleIndexResult is the per-bundle tally inside a MultiIndexReport.
type BundleIndexResult struct {
	BundleDir       string `json:"bundle_dir"`
	IndexedEntries  int    `json:"indexed_entries"`
	InsertedEntries int    `json:"inserted_entries"`
	UpdatedEntries  int    `json:"updated_entries"`
	SemanticEntries int    `json:"semantic_entries,omitempty"`
}

// MultiIndexReport aggregates indexing several bundles into one database. It is
// the additive shape returned only when more than one bundle is indexed at once;
// single-bundle indexing keeps the IndexReport contract unchanged.
type MultiIndexReport struct {
	OK              bool                `json:"ok"`
	DBPath          string              `json:"db_path"`
	Collection      string              `json:"collection"`
	Mode            string              `json:"mode"`
	IndexedEntries  int                 `json:"indexed_entries"`
	InsertedEntries int                 `json:"inserted_entries"`
	UpdatedEntries  int                 `json:"updated_entries"`
	SemanticEntries int                 `json:"semantic_entries,omitempty"`
	Embedding       *embed.Profile      `json:"embedding,omitempty"`
	Bundles         []BundleIndexResult `json:"bundles"`
	Summary         string              `json:"summary"`
}

type SearchOptions struct {
	DBPath string
	Query  string
	Limit  int
	// Mode selects keyword (default), semantic, or hybrid search. Semantic and
	// hybrid require an Embedder and a semantic index built with the same profile.
	Mode     string
	Embedder embed.Embedder
	// Filters narrow results by payload metadata. Empty string fields and nil
	// time bounds are ignored, so the zero value keeps the unfiltered behavior.
	Bundle      string
	SourceVideo string
	Source      string
	MinTime     *float64
	MaxTime     *float64
}

// SearchFilters echoes the applied metadata filters in the search report. It is
// only populated when at least one filter is active, keeping the BM25 JSON
// contract additive.
type SearchFilters struct {
	Bundle      string   `json:"bundle,omitempty"`
	SourceVideo string   `json:"source_video,omitempty"`
	Source      string   `json:"source,omitempty"`
	MinTime     *float64 `json:"min_time,omitempty"`
	MaxTime     *float64 `json:"max_time,omitempty"`
}

type SearchReport struct {
	OK         bool           `json:"ok"`
	Query      string         `json:"query"`
	DBPath     string         `json:"db_path"`
	Collection string         `json:"collection"`
	Mode       string         `json:"mode"`
	Filters    *SearchFilters `json:"filters,omitempty"`
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

	counts, err := indexLoadedInto(coll, loaded)
	if err != nil {
		return IndexReport{}, err
	}

	var semanticCount int
	var profile *embed.Profile
	if opts.Embedder != nil {
		semanticCount, profile, err = indexSemanticBundle(context.Background(), db, opts.Embedder, loaded)
		if err != nil {
			return IndexReport{}, err
		}
	}

	if err := db.Sync(); err != nil {
		return IndexReport{}, fmt.Errorf("sync evidence db: %w", err)
	}

	report := IndexReport{
		OK:              true,
		BundleDir:       loaded.Dir,
		DBPath:          dbPath,
		Collection:      KeywordCollection,
		Mode:            ModeKeyword,
		IndexedEntries:  counts.Indexed,
		InsertedEntries: counts.Inserted,
		UpdatedEntries:  counts.Updated,
		SemanticEntries: semanticCount,
		Embedding:       profile,
		Summary:         indexSummary(counts.Indexed, semanticCount),
	}
	return report, nil
}

// IndexBundles indexes several bundles into one evidence database. Every bundle
// path is resolved, de-duplicated (by real path, resolving symlinks), and
// validated before any database write, so an invalid path is rejected without
// creating or modifying the database. Indexing is idempotent by evidence_id, so
// if a later bundle fails to load or index, the bundles already processed remain
// indexed and re-running the command is safe.
func IndexBundles(bundleDirs []string, dbPath string, embedder embed.Embedder) (MultiIndexReport, error) {
	if len(bundleDirs) == 0 {
		return MultiIndexReport{}, fmt.Errorf("at least one bundle is required")
	}
	resolvedDBPath, err := resolveRequiredPath(dbPath, "db")
	if err != nil {
		return MultiIndexReport{}, err
	}

	// Phase 1: resolve, de-duplicate, and validate every bundle before writing.
	resolvedDirs := make([]string, 0, len(bundleDirs))
	seen := map[string]struct{}{}
	for _, dir := range bundleDirs {
		resolvedDir, err := resolveRequiredPath(dir, "bundle")
		if err != nil {
			return MultiIndexReport{}, err
		}
		// De-duplicate by real path so different spellings or symlinks pointing at
		// the same bundle are indexed once. Fall back to the absolute path when the
		// target cannot be resolved (it will then fail validation below).
		dedupKey := resolvedDir
		if real, err := filepath.EvalSymlinks(resolvedDir); err == nil {
			dedupKey = real
		}
		if _, dup := seen[dedupKey]; dup {
			continue
		}
		seen[dedupKey] = struct{}{}
		validation := bundle.Validate(resolvedDir)
		if !validation.OK {
			return MultiIndexReport{}, fmt.Errorf("bundle validation failed for %s: %s", resolvedDir, validation.Summary)
		}
		resolvedDirs = append(resolvedDirs, resolvedDir)
	}

	if err := os.MkdirAll(filepath.Dir(resolvedDBPath), 0o755); err != nil {
		return MultiIndexReport{}, fmt.Errorf("create evidence db directory: %w", err)
	}

	db, err := veclite.Open(resolvedDBPath)
	if err != nil {
		return MultiIndexReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	coll, err := keywordCollection(db)
	if err != nil {
		return MultiIndexReport{}, err
	}

	// Phase 2: load and index each bundle one at a time to bound memory.
	report := MultiIndexReport{
		OK:         true,
		DBPath:     resolvedDBPath,
		Collection: KeywordCollection,
		Mode:       ModeKeyword,
		Bundles:    make([]BundleIndexResult, 0, len(resolvedDirs)),
	}
	for _, dir := range resolvedDirs {
		loaded, err := bundle.Load(dir)
		if err != nil {
			return MultiIndexReport{}, err
		}
		counts, err := indexLoadedInto(coll, loaded)
		if err != nil {
			return MultiIndexReport{}, err
		}
		bundleResult := BundleIndexResult{
			BundleDir:       loaded.Dir,
			IndexedEntries:  counts.Indexed,
			InsertedEntries: counts.Inserted,
			UpdatedEntries:  counts.Updated,
		}
		if embedder != nil {
			semanticCount, profile, err := indexSemanticBundle(context.Background(), db, embedder, loaded)
			if err != nil {
				return MultiIndexReport{}, err
			}
			bundleResult.SemanticEntries = semanticCount
			report.SemanticEntries += semanticCount
			report.Embedding = profile
		}
		report.Bundles = append(report.Bundles, bundleResult)
		report.IndexedEntries += counts.Indexed
		report.InsertedEntries += counts.Inserted
		report.UpdatedEntries += counts.Updated
	}
	if err := db.Sync(); err != nil {
		return MultiIndexReport{}, fmt.Errorf("sync evidence db: %w", err)
	}

	report.Summary = fmt.Sprintf("Indexed %d evidence entries from %d bundle(s) into %s.", report.IndexedEntries, len(report.Bundles), KeywordCollection)
	if report.SemanticEntries > 0 {
		report.Summary += fmt.Sprintf(" Indexed %d semantic entries into %s.", report.SemanticEntries, TextCollection)
	}
	return report, nil
}

// indexCounts holds the per-bundle upsert tally.
type indexCounts struct {
	Indexed  int
	Inserted int
	Updated  int
}

func indexLoadedInto(coll *veclite.Collection, loaded bundle.Bundle) (indexCounts, error) {
	var counts indexCounts
	for _, item := range recordsForBundle(loaded) {
		_, inserted, err := coll.UpsertTextDocumentByKey("evidence_id", item.id, item.content, item.payload)
		if err != nil {
			return indexCounts{}, fmt.Errorf("index evidence %s: %w", item.id, err)
		}
		counts.Indexed++
		if inserted {
			counts.Inserted++
		} else {
			counts.Updated++
		}
	}
	return counts, nil
}

func indexSummary(keywordEntries, semanticEntries int) string {
	summary := fmt.Sprintf("Indexed %d evidence entries into %s.", keywordEntries, KeywordCollection)
	if semanticEntries > 0 {
		summary += fmt.Sprintf(" Indexed %d semantic entries into %s.", semanticEntries, TextCollection)
	}
	return summary
}

// indexSemanticBundle embeds each timeline entry's content and stores a vector +
// content document in the semantic collection, idempotently by evidence_id. It
// records the embedding profile so search can detect a mismatched embedder, and
// rejects mixing embedders by failing when the stored profile differs.
func indexSemanticBundle(ctx context.Context, db *veclite.DB, embedder embed.Embedder, loaded bundle.Bundle) (int, *embed.Profile, error) {
	records := recordsForBundle(loaded)
	if len(records) == 0 {
		return 0, nil, nil
	}

	texts := make([]string, len(records))
	for i, item := range records {
		texts[i] = item.content
	}
	vectors, err := embedder.Embed(ctx, texts)
	if err != nil {
		return 0, nil, fmt.Errorf("embed bundle %s: %w", loaded.Dir, err)
	}
	if len(vectors) != len(records) {
		return 0, nil, fmt.Errorf("embedder returned %d vectors for %d records", len(vectors), len(records))
	}
	if len(vectors[0]) == 0 {
		return 0, nil, fmt.Errorf("embedder returned empty vectors")
	}

	profile := embedder.Profile()
	profile.Dimensions = len(vectors[0])
	if err := ensureEmbeddingProfile(db, profile); err != nil {
		return 0, nil, err
	}

	coll, err := textCollection(db)
	if err != nil {
		return 0, nil, err
	}
	for i, item := range records {
		// Replace any existing vector for this evidence_id so re-indexing is
		// idempotent (veclite has no upsert that carries both vector and content).
		if _, err := coll.DeleteWhere(veclite.Equal("evidence_id", item.id)); err != nil {
			return 0, nil, fmt.Errorf("replace semantic evidence %s: %w", item.id, err)
		}
		if _, err := coll.InsertDocument(vectors[i], item.content, item.payload); err != nil {
			return 0, nil, fmt.Errorf("index semantic evidence %s: %w", item.id, err)
		}
	}
	return len(records), &profile, nil
}

func textCollection(db *veclite.DB) (*veclite.Collection, error) {
	if db.HasCollection(TextCollection) {
		return db.GetCollection(TextCollection)
	}
	// Dimension 0 lets veclite auto-detect from the first inserted vector; a
	// text index over the same content enables hybrid (vector + BM25) search.
	return db.CreateCollection(TextCollection,
		veclite.WithTextIndex("evidence_id", "bundle", "source_video", "frame", "ocr_path", "source"),
	)
}

func metaCollection(db *veclite.DB) (*veclite.Collection, error) {
	if db.HasCollection(MetaCollection) {
		return db.GetCollection(MetaCollection)
	}
	return db.CreateCollection(MetaCollection, veclite.WithTextIndex("key"))
}

// ensureEmbeddingProfile writes the embedding profile on first use and rejects a
// later embedder whose provider, model, or dimensions differ from it.
func ensureEmbeddingProfile(db *veclite.DB, profile embed.Profile) error {
	coll, err := metaCollection(db)
	if err != nil {
		return err
	}
	existing, err := readEmbeddingProfile(coll)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.Provider != profile.Provider || existing.Model != profile.Model || existing.Dimensions != profile.Dimensions {
			return fmt.Errorf("embedding profile mismatch: index uses %s/%s (dim %d), requested %s/%s (dim %d)",
				existing.Provider, existing.Model, existing.Dimensions, profile.Provider, profile.Model, profile.Dimensions)
		}
		return nil
	}
	payload := map[string]any{
		"key":        embeddingProfileKey,
		"provider":   profile.Provider,
		"model":      profile.Model,
		"dimensions": profile.Dimensions,
	}
	if _, _, err := coll.UpsertTextDocumentByKey("key", embeddingProfileKey, "", payload); err != nil {
		return fmt.Errorf("write embedding profile: %w", err)
	}
	return nil
}

func readEmbeddingProfile(coll *veclite.Collection) (*embed.Profile, error) {
	records, err := coll.Find(veclite.Equal("key", embeddingProfileKey))
	if err != nil {
		return nil, fmt.Errorf("read embedding profile: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	payload := records[0].Payload
	return &embed.Profile{
		Provider:   stringPayload(payload, "provider"),
		Model:      stringPayload(payload, "model"),
		Dimensions: int(floatPayload(payload, "dimensions")),
	}, nil
}

func normalizeMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", ModeKeyword:
		return ModeKeyword, nil
	case ModeSemantic:
		return ModeSemantic, nil
	case ModeHybrid:
		return ModeHybrid, nil
	default:
		return "", fmt.Errorf("unknown search mode %q (want keyword, semantic, or hybrid)", mode)
	}
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
	mode, err := normalizeMode(opts.Mode)
	if err != nil {
		return SearchReport{}, err
	}
	// Validate and build filters before touching the database so invalid filter
	// arguments fail fast regardless of database state.
	activeFilters, filterEcho, err := buildFilters(opts)
	if err != nil {
		return SearchReport{}, err
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

	if mode == ModeKeyword {
		coll, err := db.GetCollection(KeywordCollection)
		if err != nil {
			return SearchReport{}, fmt.Errorf("evidence collection not found: %s", KeywordCollection)
		}
		results, err := coll.TextSearch(query, searchOptionsFor(coll, limit, activeFilters)...)
		if err != nil {
			return SearchReport{}, fmt.Errorf("search evidence: %w", err)
		}
		return buildSearchReport(dbPath, query, mode, KeywordCollection, filterEcho, limit, results), nil
	}

	// Semantic and hybrid modes embed the query and search the vector collection.
	if opts.Embedder == nil {
		return SearchReport{}, fmt.Errorf("%s search requires an embedding provider (configure --embed)", mode)
	}
	if !db.HasCollection(TextCollection) {
		return SearchReport{}, fmt.Errorf("no semantic index found in %s; run vidtrace index with --embed first", dbPath)
	}
	indexedProfile, err := loadEmbeddingProfile(db)
	if err != nil {
		return SearchReport{}, err
	}
	want := opts.Embedder.Profile()
	if indexedProfile != nil && (indexedProfile.Provider != want.Provider || indexedProfile.Model != want.Model) {
		return SearchReport{}, fmt.Errorf("embedding profile mismatch: index uses %s/%s, search uses %s/%s",
			indexedProfile.Provider, indexedProfile.Model, want.Provider, want.Model)
	}

	vectors, err := opts.Embedder.Embed(context.Background(), []string{query})
	if err != nil {
		return SearchReport{}, fmt.Errorf("embed query: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return SearchReport{}, fmt.Errorf("embedder returned no query vector")
	}
	queryVec := vectors[0]
	if indexedProfile != nil && indexedProfile.Dimensions != 0 && indexedProfile.Dimensions != len(queryVec) {
		return SearchReport{}, fmt.Errorf("embedding dimension mismatch: index has %d, query has %d", indexedProfile.Dimensions, len(queryVec))
	}

	coll, err := db.GetCollection(TextCollection)
	if err != nil {
		return SearchReport{}, fmt.Errorf("semantic collection not found: %s", TextCollection)
	}
	searchOpts := searchOptionsFor(coll, limit, activeFilters)
	var results []veclite.Result
	switch mode {
	case ModeSemantic:
		results, err = coll.Search(queryVec, searchOpts...)
	case ModeHybrid:
		results, err = coll.HybridSearch(queryVec, query, searchOpts...)
	}
	if err != nil {
		return SearchReport{}, fmt.Errorf("%s search: %w", mode, err)
	}
	return buildSearchReport(dbPath, query, mode, TextCollection, filterEcho, limit, results), nil
}

// searchOptionsFor builds the TopK + filter options. VecLite truncates matches
// to TopK before applying payload filters, so when filters are active it
// over-fetches the full candidate set (capped by collection size); the caller
// trims to the user limit.
func searchOptionsFor(coll *veclite.Collection, limit int, activeFilters []veclite.Filter) []veclite.SearchOption {
	fetchK := limit
	var opts []veclite.SearchOption
	if len(activeFilters) > 0 {
		if count := coll.Count(); count > fetchK {
			fetchK = count
		}
		opts = append(opts, veclite.WithFilter(veclite.And(activeFilters...)))
	}
	opts = append(opts, veclite.TopK(fetchK))
	return opts
}

func buildSearchReport(dbPath, query, mode, collection string, filterEcho *SearchFilters, limit int, results []veclite.Result) SearchReport {
	report := SearchReport{
		OK:         true,
		Query:      query,
		DBPath:     dbPath,
		Collection: collection,
		Mode:       mode,
		Filters:    filterEcho,
		Results:    make([]SearchResult, 0, min(limit, len(results))),
	}
	for _, result := range results {
		if len(report.Results) >= limit {
			break
		}
		report.Results = append(report.Results, searchResultFromPayload(result))
	}
	return report
}

func loadEmbeddingProfile(db *veclite.DB) (*embed.Profile, error) {
	if !db.HasCollection(MetaCollection) {
		return nil, nil
	}
	coll, err := db.GetCollection(MetaCollection)
	if err != nil {
		return nil, nil
	}
	return readEmbeddingProfile(coll)
}

func keywordCollection(db *veclite.DB) (*veclite.Collection, error) {
	if db.HasCollection(KeywordCollection) {
		return db.GetCollection(KeywordCollection)
	}
	return db.CreateCollection(KeywordCollection,
		veclite.WithTextIndex("evidence_id", "bundle", "source_video", "frame", "ocr_path", "source"),
	)
}

// buildFilters converts SearchOptions metadata constraints into VecLite payload
// filters. It returns the filters to AND together for the search and a
// SearchFilters echo (nil when no filter is active) for the report.
func buildFilters(opts SearchOptions) ([]veclite.Filter, *SearchFilters, error) {
	var filters []veclite.Filter
	echo := &SearchFilters{}
	active := false

	if bundle := strings.TrimSpace(opts.Bundle); bundle != "" {
		resolved, err := filepath.Abs(bundle)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve bundle filter: %w", err)
		}
		filters = append(filters, veclite.Equal("bundle", resolved))
		echo.Bundle = resolved
		active = true
	}
	if sourceVideo := strings.TrimSpace(opts.SourceVideo); sourceVideo != "" {
		filters = append(filters, veclite.Equal("source_video", sourceVideo))
		echo.SourceVideo = sourceVideo
		active = true
	}
	if source := strings.TrimSpace(opts.Source); source != "" {
		filters = append(filters, veclite.Equal("source", source))
		echo.Source = source
		active = true
	}
	if opts.MinTime != nil && opts.MaxTime != nil && *opts.MinTime > *opts.MaxTime {
		return nil, nil, fmt.Errorf("min-time %.3f is greater than max-time %.3f", *opts.MinTime, *opts.MaxTime)
	}
	if opts.MinTime != nil {
		filters = append(filters, veclite.GreaterThanOrEqual("time_seconds", *opts.MinTime))
		minBound := *opts.MinTime
		echo.MinTime = &minBound
		active = true
	}
	if opts.MaxTime != nil {
		filters = append(filters, veclite.LessThanOrEqual("time_seconds", *opts.MaxTime))
		maxBound := *opts.MaxTime
		echo.MaxTime = &maxBound
		active = true
	}

	if !active {
		return nil, nil, nil
	}
	return filters, echo, nil
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
