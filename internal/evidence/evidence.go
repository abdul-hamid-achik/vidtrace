package evidence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdul-hamid-achik/veclite"
	vlsession "github.com/abdul-hamid-achik/veclite/session"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/embed"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

const (
	SchemaVersion = "1"
	// Collection is the single evidence collection. It holds one record per
	// timeline entry, indexed by evidence_id, with BM25 over the content and a
	// named "text" vector space for semantic/hybrid search (when an embedder
	// is provided). Pre-v0.17.0 databases used three collections
	// (evidence_entries_keyword, evidence_entries_text, evidence_meta);
	// vidtrace migrate-evidence converts them to this single collection.
	Collection      = "evidence_entries"
	TextVectorSpace = "text"

	// Legacy collection names used before the v0.17.0 single-collection
	// migration. Kept here so migrate-evidence can read and convert them.
	legacyKeywordCollection = "evidence_entries_keyword"
	legacyTextCollection    = "evidence_entries_text"
	legacyMetaCollection    = "evidence_meta"
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

	db, err := vlsession.New(vlsession.Config{Path: dbPath}).ReadWrite()
	if err != nil {
		return IndexReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	coll, err := ensureEvidenceCollection(db, opts.Embedder)
	if err != nil {
		return IndexReport{}, err
	}

	counts, err := indexLoadedInto(coll, loaded, opts.Embedder)
	if err != nil {
		return IndexReport{}, err
	}

	var semanticCount int
	var profile *embed.Profile
	if opts.Embedder != nil {
		semanticCount = counts.Semantic
		profile = profileOrNil(coll, opts.Embedder)
	}

	if err := db.Sync(); err != nil {
		return IndexReport{}, fmt.Errorf("sync evidence db: %w", err)
	}

	report := IndexReport{
		OK:              true,
		BundleDir:       loaded.Dir,
		DBPath:          dbPath,
		Collection:      Collection,
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

	db, err := vlsession.New(vlsession.Config{Path: resolvedDBPath}).ReadWrite()
	if err != nil {
		return MultiIndexReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	coll, err := ensureEvidenceCollection(db, embedder)
	if err != nil {
		return MultiIndexReport{}, err
	}

	// Phase 2: load and index each bundle one at a time to bound memory.
	report := MultiIndexReport{
		OK:         true,
		DBPath:     resolvedDBPath,
		Collection: Collection,
		Mode:       ModeKeyword,
		Bundles:    make([]BundleIndexResult, 0, len(resolvedDirs)),
	}
	for _, dir := range resolvedDirs {
		loaded, err := bundle.Load(dir)
		if err != nil {
			return MultiIndexReport{}, err
		}
		counts, err := indexLoadedInto(coll, loaded, embedder)
		if err != nil {
			return MultiIndexReport{}, err
		}
		bundleResult := BundleIndexResult{
			BundleDir:       loaded.Dir,
			IndexedEntries:  counts.Indexed,
			InsertedEntries: counts.Inserted,
			UpdatedEntries:  counts.Updated,
			SemanticEntries: counts.Semantic,
		}
		report.Bundles = append(report.Bundles, bundleResult)
		report.IndexedEntries += counts.Indexed
		report.InsertedEntries += counts.Inserted
		report.UpdatedEntries += counts.Updated
		report.SemanticEntries += counts.Semantic
	}
	if embedder != nil {
		report.Embedding = profileOrNil(coll, embedder)
	}
	if err := db.Sync(); err != nil {
		return MultiIndexReport{}, fmt.Errorf("sync evidence db: %w", err)
	}

	report.Summary = fmt.Sprintf("Indexed %d evidence entries from %d bundle(s) into %s.", report.IndexedEntries, len(report.Bundles), Collection)
	if report.SemanticEntries > 0 {
		report.Summary += fmt.Sprintf(" Indexed %d semantic entries into the %s vector space.", report.SemanticEntries, TextVectorSpace)
	}
	return report, nil
}

// indexCounts holds the per-bundle upsert tally.
type indexCounts struct {
	Indexed  int
	Inserted int
	Updated  int
	Semantic int
}

// indexLoadedInto upserts each timeline entry as one record into the single
// evidence collection, keyed by evidence_id. When an embedder is provided it
// also embeds the content and stores the vector in the named "text" space on
// the same record, so keyword, semantic, and hybrid search all run against one
// collection. Upserting by key is idempotent across re-indexes and preserves
// CreatedAt/bookkeeping on replace.
func indexLoadedInto(coll *veclite.Collection, loaded bundle.Bundle, embedder embed.Embedder) (indexCounts, error) {
	var counts indexCounts
	items := recordsForBundle(loaded)
	if len(items) == 0 {
		return counts, nil
	}

	var vectors [][]float32
	if embedder != nil {
		texts := make([]string, len(items))
		for i, item := range items {
			texts[i] = item.content
		}
		var err error
		vectors, err = embedder.Embed(context.Background(), texts)
		if err != nil {
			return indexCounts{}, fmt.Errorf("embed bundle %s: %w", loaded.Dir, err)
		}
		if len(vectors) != len(items) {
			return indexCounts{}, fmt.Errorf("embedder returned %d vectors for %d records", len(vectors), len(items))
		}
		if dim := len(vectors[0]); dim == 0 {
			return indexCounts{}, fmt.Errorf("embedder returned empty vectors")
		} else {
			for i, vec := range vectors {
				if len(vec) != dim {
					return indexCounts{}, fmt.Errorf("embedder returned inconsistent vector length at index %d: got %d, want %d", i, len(vec), dim)
				}
			}
		}
	}

	for i, item := range items {
		in := veclite.RecordInput{
			Content: item.content,
			Payload: item.payload,
		}
		if embedder != nil {
			in.Vectors = map[string][]float32{TextVectorSpace: vectors[i]}
		}
		_, inserted, err := coll.UpsertRecordByKey("evidence_id", item.id, in)
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
	counts.Semantic = len(vectors)
	return counts, nil
}

func indexSummary(keywordEntries, semanticEntries int) string {
	summary := fmt.Sprintf("Indexed %d evidence entries into %s.", keywordEntries, Collection)
	if semanticEntries > 0 {
		summary += fmt.Sprintf(" Indexed %d semantic entries into the %s vector space.", semanticEntries, TextVectorSpace)
	}
	return summary
}

// ensureEvidenceCollection opens (or creates) the single evidence collection
// with a BM25 text index over the content and the payload fields used for
// filtering. When an embedder is provided it also declares the named "text"
// vector space and attaches an EmbeddingProfile, so vector inserts are
// validated and search can detect a mismatched embedder. The profile is read
// back and compared against the requested one to reject mixing embedders.
func ensureEvidenceCollection(db *veclite.DB, embedder embed.Embedder) (*veclite.Collection, error) {
	if db.HasCollection(Collection) {
		coll, err := db.GetCollection(Collection)
		if err != nil {
			return nil, err
		}
		if embedder != nil {
			want := embedder.Profile()
			if existing, ok := coll.EmbeddingProfile(); ok {
				if existing.Provider != want.Provider || existing.Model != want.Model {
					return nil, fmt.Errorf("embedding profile mismatch: index uses %s/%s, requested %s/%s",
						existing.Provider, existing.Model, want.Provider, want.Model)
				}
				if existing.Dimension > 0 && want.Dimensions > 0 && existing.Dimension != want.Dimensions {
					return nil, fmt.Errorf("embedding profile mismatch: index has dim %d, requested %d",
						existing.Dimension, want.Dimensions)
				}
			} else {
				// Pre-v0.17.0 DB: the collection exists but has no first-class
				// profile and no text space. Declare the space and attach the
				// profile so semantic search works. This also migrates legacy
				// keyword-only DBs on first semantic index.
				if err := setupTextSpace(coll, want); err != nil {
					return nil, err
				}
			}
		}
		return coll, nil
	}

	coll, err := db.CreateCollection(Collection,
		veclite.WithTextIndex("evidence_id", "bundle", "source_video", "frame", "ocr_path", "source"),
	)
	if err != nil {
		return nil, err
	}
	if embedder != nil {
		if err := setupTextSpace(coll, embedder.Profile()); err != nil {
			return nil, err
		}
	}
	return coll, nil
}

// setupTextSpace declares the named "text" vector space and attaches the
// embedding profile to it (and to the collection default space for visibility
// via EmbeddingProfile()). Dimension auto-detects from the first vector when 0.
func setupTextSpace(coll *veclite.Collection, profile embed.Profile) error {
	if !coll.HasVectorSpace(TextVectorSpace) {
		vp := veclite.EmbeddingProfile{
			Provider: profile.Provider,
			Model:    profile.Model,
		}
		if profile.Dimensions > 0 {
			vp.Dimension = profile.Dimensions
		}
		if err := coll.AddVectorSpace(veclite.VectorSpaceConfig{
			Name:     TextVectorSpace,
			Modality: "text",
			Provider: profile.Provider,
			Model:    profile.Model,
			Profile:  &vp,
		}); err != nil {
			return err
		}
		// Also set the collection-level default-space profile so
		// coll.EmbeddingProfile() reports the embedder for compatibility checks.
		if err := coll.SetEmbeddingProfile(vp); err != nil {
			return err
		}
	}
	return nil
}

// profileOrNil returns the embedding profile from the collection when the text
// space is present, so the index report echoes it. Returns nil for keyword-only.
func profileOrNil(coll *veclite.Collection, embedder embed.Embedder) *embed.Profile {
	if !coll.HasVectorSpace(TextVectorSpace) {
		return nil
	}
	p := embedder.Profile()
	return &p
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

	db, err := vlsession.New(vlsession.Config{Path: dbPath}).ReadOnly()
	if err != nil {
		return SearchReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if mode == ModeKeyword {
		coll, err := db.GetCollection(Collection)
		if err != nil {
			return SearchReport{}, fmt.Errorf("evidence collection not found: %s", Collection)
		}
		results, err := coll.TextSearch(query, searchOptionsFor(coll, limit, activeFilters)...)
		if err != nil {
			return SearchReport{}, fmt.Errorf("search evidence: %w", err)
		}
		return buildSearchReport(dbPath, query, mode, Collection, filterEcho, limit, results), nil
	}

	// Semantic and hybrid modes embed the query and search the named "text"
	// vector space on the single evidence collection.
	if opts.Embedder == nil {
		return SearchReport{}, fmt.Errorf("%s search requires an embedding provider (configure --embed)", mode)
	}
	if !db.HasCollection(Collection) {
		return SearchReport{}, fmt.Errorf("no evidence index found in %s; run vidtrace index first", dbPath)
	}
	coll, err := db.GetCollection(Collection)
	if err != nil {
		return SearchReport{}, fmt.Errorf("evidence collection not found: %s", Collection)
	}
	if !coll.HasVectorSpace(TextVectorSpace) {
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

	searchOpts := searchOptionsFor(coll, limit, activeFilters)
	var results []veclite.Result
	switch mode {
	case ModeSemantic:
		results, err = coll.SearchSpace(TextVectorSpace, queryVec, searchOpts...)
	case ModeHybrid:
		results, err = coll.HybridSearchSpace(TextVectorSpace, queryVec, query, searchOpts...)
	}
	if err != nil {
		return SearchReport{}, fmt.Errorf("%s search: %w", mode, err)
	}
	return buildSearchReport(dbPath, query, mode, Collection, filterEcho, limit, results), nil
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
	if !db.HasCollection(Collection) {
		return nil, nil
	}
	coll, err := db.GetCollection(Collection)
	if err != nil {
		return nil, nil
	}
	if p, ok := coll.EmbeddingProfile(); ok {
		return &embed.Profile{Provider: p.Provider, Model: p.Model, Dimensions: p.Dimension}, nil
	}
	return nil, nil
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

// MigrateReport summarizes a migrate-evidence run. It records how many records
// were copied from each legacy collection, whether the old collections were
// dropped, and the resulting single-collection state.
type MigrateReport struct {
	OK              bool   `json:"ok"`
	DBPath          string `json:"db_path"`
	Collection      string `json:"collection"`
	KeywordRecords  int    `json:"keyword_records"`
	SemanticRecords int    `json:"semantic_records"`
	MigratedRecords int    `json:"migrated_records"`
	DroppedLegacy   bool   `json:"dropped_legacy"`
	AlreadyMigrated bool   `json:"already_migrated,omitempty"`
	Summary         string `json:"summary"`
}

// Migrate converts a pre-v0.17.0 evidence database (which used three
// collections: evidence_entries_keyword, evidence_entries_text, and
// evidence_meta) into the single-collection layout used by v0.17.0+. It is
// idempotent: a database that already has the single collection and no legacy
// collections reports AlreadyMigrated and is left untouched. Legacy
// collections are read-only here, so a read-only database is rejected.
func Migrate(dbPath string) (MigrateReport, error) {
	resolved, err := resolveRequiredPath(dbPath, "db")
	if err != nil {
		return MigrateReport{}, err
	}
	if _, err := os.Stat(resolved); err != nil {
		return MigrateReport{}, fmt.Errorf("evidence db not found: %s", resolved)
	}

	db, err := vlsession.New(vlsession.Config{Path: resolved}).ReadWrite()
	if err != nil {
		return MigrateReport{}, fmt.Errorf("open evidence db: %w", err)
	}
	defer func() { _ = db.Close() }()

	report := MigrateReport{DBPath: resolved, Collection: Collection}

	hasLegacy := db.HasCollection(legacyKeywordCollection) || db.HasCollection(legacyTextCollection)
	hasNew := db.HasCollection(Collection)
	if !hasLegacy && (!hasNew || !db.HasCollection(legacyMetaCollection)) {
		// Nothing to do: either already migrated, or not an evidence DB.
		if hasNew && !hasLegacy {
			report.OK = true
			report.AlreadyMigrated = true
			report.Summary = fmt.Sprintf("Database %s already uses the single %s collection; nothing to migrate.", resolved, Collection)
			return report, nil
		}
		return MigrateReport{}, fmt.Errorf("no legacy evidence collections found in %s; not a pre-v0.17.0 evidence database", resolved)
	}

	// Ensure the destination collection exists with BM25 over the filter
	// fields. Vectors are attached per-record via InsertRecord, so we only add
	// the named text space if the legacy text collection had vectors.
	var coll *veclite.Collection
	if hasNew {
		coll, err = db.GetCollection(Collection)
		if err != nil {
			return MigrateReport{}, fmt.Errorf("get %s: %w", Collection, err)
		}
	} else {
		coll, err = db.CreateCollection(Collection,
			veclite.WithTextIndex("evidence_id", "bundle", "source_video", "frame", "ocr_path", "source"),
		)
		if err != nil {
			return MigrateReport{}, fmt.Errorf("create %s: %w", Collection, err)
		}
	}

	// Index keyword records by evidence_id, carrying over the matching vector
	// from the legacy text collection when one exists.
	keywordByID := map[string]veclite.Record{}
	if db.HasCollection(legacyKeywordCollection) {
		kcoll, err := db.GetCollection(legacyKeywordCollection)
		if err != nil {
			return MigrateReport{}, fmt.Errorf("get %s: %w", legacyKeywordCollection, err)
		}
		records := kcoll.All()
		for _, r := range records {
			id, _ := r.Payload["evidence_id"].(string)
			if id == "" {
				continue
			}
			keywordByID[id] = *r
			report.KeywordRecords++
		}
	}

	vectorByID := map[string][]float32{}
	if db.HasCollection(legacyTextCollection) {
		tcoll, err := db.GetCollection(legacyTextCollection)
		if err != nil {
			return MigrateReport{}, fmt.Errorf("get %s: %w", legacyTextCollection, err)
		}
		records := tcoll.All()
		for _, r := range records {
			id, _ := r.Payload["evidence_id"].(string)
			if id == "" {
				continue
			}
			if len(r.Vector) > 0 {
				vectorByID[id] = r.Vector
				report.SemanticRecords++
			}
		}
		// Declare the named text space now that we know we have vectors.
		if len(vectorByID) > 0 && !coll.HasVectorSpace(TextVectorSpace) {
			var dim int
			for _, v := range vectorByID {
				dim = len(v)
				break
			}
			if err := coll.AddVectorSpace(veclite.VectorSpaceConfig{
				Name:     TextVectorSpace,
				Modality: "text",
			}); err != nil {
				return MigrateReport{}, fmt.Errorf("declare text space: %w", err)
			}
			_ = dim // dimension inferred by InsertRecord
		}
	}

	for id, kr := range keywordByID {
		in := veclite.RecordInput{
			Content: kr.Content,
			Payload: kr.Payload,
		}
		if vec, ok := vectorByID[id]; ok {
			in.Vectors = map[string][]float32{TextVectorSpace: vec}
		}
		_, _, err := coll.UpsertRecordByKey("evidence_id", id, in)
		if err != nil {
			return MigrateReport{}, fmt.Errorf("migrate evidence %s: %w", id, err)
		}
		report.MigratedRecords++
	}

	// Drop the legacy collections now that the data is consolidated.
	for _, name := range []string{legacyKeywordCollection, legacyTextCollection, legacyMetaCollection} {
		if db.HasCollection(name) {
			if err := db.DropCollection(name); err != nil {
				return MigrateReport{}, fmt.Errorf("drop legacy %s: %w", name, err)
			}
			report.DroppedLegacy = true
		}
	}

	if err := db.Sync(); err != nil {
		return MigrateReport{}, fmt.Errorf("sync evidence db: %w", err)
	}

	report.OK = true
	report.Summary = fmt.Sprintf("Migrated %d records (%d semantic) into the single %s collection and dropped legacy collections.", report.MigratedRecords, report.SemanticRecords, Collection)
	return report, nil
}
