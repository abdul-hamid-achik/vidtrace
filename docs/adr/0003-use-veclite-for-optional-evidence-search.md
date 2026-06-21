# ADR-0003: Use VecLite for Optional Evidence Search

## Status

Accepted, superseded in part by ADR-0003 Addendum (single-collection migration, v0.17.0+)

## Context

vidtrace turns bug videos into artifact bundles: frames, OCR text, transcripts, metadata, and `timeline.json`. The extraction pipeline should stay reliable without requiring a vector database or an embedding provider.

Agents still need better ways to find the relevant timestamp before inspecting code. The planned evidence-search workflow should search timeline entries by OCR, transcript text, frame paths, and later semantic embeddings. vecgrep already owns codebase search, so vidtrace should not duplicate source-code indexing.

## Decision Drivers

- Keep `vidtrace extract` independent from VecLite and embedding providers.
- Make search local-first and usable from Go without a server.
- Support BM25 keyword search before semantic embeddings are configured.
- Keep codebase search in vecgrep.
- Leave room for future multimodal evidence search when VecLite supports named vector spaces.

## Considered Options

1. Keep only the current heuristic `compare` command.
2. Add a separate hosted search service.
3. Use VecLite as an optional local evidence-search index.

## Decision Outcome

Chosen option: **Use VecLite as an optional local evidence-search index**.

vidtrace adds evidence search through:

- `vidtrace index <bundle> --db <path>` to index an existing bundle.
- `vidtrace search <db> <query> --json` to return timestamped evidence.
- `vidtrace investigate <bundle> --query <text> [--codebase <path>]` as a handoff workflow that suggests vecgrep queries.

Extraction remains independent. Indexing reads an existing bundle and writes a separate VecLite database.

## Implementation Direction

Phase 1 uses BM25 only:

- collection: `evidence_entries` (single collection since v0.17.0; pre-v0.17.0 used `evidence_entries_keyword`)
- one record per `timeline.json` entry
- content: timestamp, OCR text, transcript text, and frame path
- payload: `schema_version`, `bundle`, `source_video`, `time_seconds`, `source`, `frame`, `ocr_path`, `has_ocr`, and `has_transcript`
- vector: none; use VecLite text-only records
- search: `TextSearch` only

Phase 2 adds semantic text and hybrid search behind explicit config (implemented):

- collection: `evidence_entries` (single collection since v0.17.0; pre-v0.17.0 used `evidence_entries_text`)
- vector: embedding of the same content indexed for BM25, stored in a named `text` vector space on the same record so VecLite `HybridSearchSpace` can combine vector and keyword scores for that space
- search modes: keyword (default, no provider needed), semantic (`SearchSpace`), hybrid (`HybridSearchSpace`)
- provider: an `Embedder` interface (`internal/embed`) with an Ollama provider that shells out over HTTP, matching how vidtrace orchestrates ffmpeg, ffprobe, tesseract, and whisper
- profile guard: the embedding profile (provider, model, dimensions) is attached to the collection and the named `text` vector space; indexing or searching with a different provider, model, or dimension is rejected

### Addendum: single-collection migration (v0.17.0+)

VecLite v0.16.0 introduced named vector spaces, and v0.17.0 added `UpsertRecordByKey` and `HybridSearchSpace` so a consumer can store one record per timeline entry with BM25 over the content and a named `text` vector space on the same record. vidtrace v0.17.0+ collapsed the pre-v0.17.0 three-collection layout (`evidence_entries_keyword` + `evidence_entries_text` + `evidence_meta`) into a single `evidence_entries` collection. `vidtrace migrate-evidence <db>` converts pre-v0.17.0 databases in place; running it on a modern database is a no-op (`already_migrated: true`). The single-collection layout removes content duplication, unifies the three search modes against one collection, and leaves room for future multimodal evidence (for example an `image` named space for frame embeddings).

Future VecLite named vector spaces can merge text and frame embeddings into one logical evidence collection.

## Consequences

**Good:**

- Existing extraction workflows stay stable.
- Agents can search evidence without watching the video.
- Users get useful keyword search before configuring embeddings.
- vecgrep remains the companion for source-code search.

**Bad:**

- BM25-only indexing depends on VecLite text-only document records.
- Semantic search duplicates records into a second collection until VecLite named vector spaces exist. *(Resolved in v0.17.0+ by the single-collection migration; see the Implementation Direction addendum.)*
- vidtrace must manage embedding-profile compatibility for semantic indexes.
