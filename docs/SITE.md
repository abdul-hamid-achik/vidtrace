# Documentation Site

The documentation site is a VitePress app rooted at `docs/` and configured for Vercel.

Use this for public, browsable documentation. Keep local planning artifacts, generated media, and temporary bundles out of the published docs.

## Develop

From the repository root:

```bash
bun install --frozen-lockfile
bun run docs:dev
```

The dev server serves the Markdown files under `docs/`.

## Build

```bash
task site
```

`task site` runs:

```bash
bun install --frozen-lockfile
bun run docs:build
```

The static output is written to:

```bash
docs/.vitepress/dist
```

## Vercel

`vercel.json` uses:

| Setting | Value |
|---|---|
| Bun version | `1.x` |
| Install command | `bun install --frozen-lockfile` |
| Build command | `bun run docs:build` |
| Output directory | `docs/.vitepress/dist` |

Deploy the repository to Vercel with the default Git integration. Vercel installs from `bun.lock`, runs the build command, and serves the generated VitePress output.

## Versions

The docs site uses the latest stable VitePress line available when this was checked:

- Bun package manager: `1.x`
- VitePress: `1.6.4`
- Vite: `7.3.5`
- Vue plugin for Vite: `@vitejs/plugin-vue@6.0.7`
- esbuild: `0.28.1`

VitePress also publishes a `next` alpha line. Keep production docs on the latest stable line unless the project explicitly decides to adopt the alpha.

## Published Pages

The VitePress sidebar exposes:

- `docs/index.md`
- `docs/INSTALL.md`
- `docs/USAGE.md`
- `docs/ANALYSIS.md`
- `docs/STUDIO.md`
- `docs/CLI_CONTRACT.md`
- `docs/ARTIFACT_SCHEMA.md`
- `docs/TESTING.md`
- `docs/RELEASE.md`
- `docs/ARCHITECTURE.md`
- `docs/ROADMAP.md`
- `docs/HANDOFF.md`
- ADRs under `docs/adr/`

## Exclusions

The site root is `docs/`, so the VitePress build does not publish repository-local files such as:

- source videos such as `~/Downloads/bug.mp4`
- generated artifact bundles
- `.glyphrun/`
- root `dist/` content
- local VecLite databases
- internal prompt files

## Verification

```bash
bun run docs:build
glyph spec verify specs/glyphrun/docs_site.yml --format md
glyph run specs/glyphrun/docs_site.yml --format md
```
