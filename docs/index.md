---
layout: home

hero:
  name: vidtrace
  text: Bug video evidence, timestamped.
  tagline: Turn screen recordings into structured evidence bundles that humans and coding agents can inspect. Local-first Go CLI — frames, OCR, transcripts, and a timeline that ties what was seen to what was said.
  image:
    src: /logo.svg
    alt: vidtrace logo
  actions:
    - theme: brand
      text: Install
      link: /install
    - theme: alt
      text: Quick start
      link: /usage
    - theme: alt
      text: GitHub
      link: https://github.com/abdul-hamid-achik/vidtrace

features:
  - icon: ⏱️
    title: Timestamped evidence
    details: Every frame, OCR block, and transcript segment is mapped to a second-accurate timeline. Cite what you saw and what was said by timestamp, not by vague notes.
    link: /artifact-schema
    linkText: See the bundle layout
  - icon: 🤖
    title: Agent-ready JSON
    details: Every command emits stable JSON. Agents read output_dir, then inspect metadata.json, timeline.json, OCR text, and selected frames — no TUI trapping.
    link: /cli-contract
    linkText: Read the CLI contract
  - icon: 🖥️
    title: Terminal Studio
    details: Browse timeline entries, view OCR and transcript side-by-side, open frames, reveal in Finder, and copy concise evidence summaries — all keyboard-first.
    link: /studio
    linkText: Studio guide
  - icon: 🔎
    title: Evidence search
    details: Index bundles into a VecLite database and search by keyword, semantic, or hybrid mode. Filter by bundle, source video, evidence source, and time window.
    link: /usage
    linkText: Search workflow
  - icon: 🎫
    title: Ticket comparison
    details: Compare a ticket description against extracted video evidence with confidence scores and term hits. Generate Markdown analysis reports for handoff.
    link: /analysis
    linkText: Analysis guide
  - icon: ✂️
    title: Clip, GIF, stitch
    details: Cut sub-clips at timestamp ranges, make lightweight GIFs for tickets and chat, and stitch clips into a summary video. Stash outputs to fcheap.
    link: /clip
    linkText: Clip commands
  - icon: 🔌
    title: MCP server
    details: Run a read-only Model Context Protocol server over stdio. Agents call validate, search, compare, analyze, and investigate tools with structured outputs.
    link: /cli-contract
    linkText: MCP tools
  - icon: 🏠
    title: Local-first
    details: Runs entirely on your machine with ffmpeg, tesseract, and whisper. No cloud uploads. Optional Ollama for semantic search. MIT licensed.
    link: /install
    linkText: Install guide
---

<script setup>
import TerminalDemo from "./.vitepress/components/TerminalDemo.vue";
import PipelineFlow from "./.vitepress/components/PipelineFlow.vue";
import FAQAccordion from "./.vitepress/components/FAQAccordion.vue";
import ComparisonTable from "./.vitepress/components/ComparisonTable.vue";
</script>

<div class="vt-section" style="padding-top: 56px;">
  <TerminalDemo />
  <div class="vt-install">
    <div class="vt-install-code" onclick="navigator.clipboard?.writeText('brew tap abdul-hamid-achik/tap && brew install --cask abdul-hamid-achik/tap/vidtrace')">
      <span class="vt-install-prompt">$</span>
      <span>brew tap abdul-hamid-achik/tap &amp;&amp; brew install --cask abdul-hamid-achik/tap/vidtrace</span>
      <span class="vt-install-copy">click to copy</span>
    </div>
  </div>
</div>

<div class="vt-section">
  <h2 class="vt-section-title">From <span class="vt-gradient-text">screen recording</span> to <span class="vt-gradient-text">citable evidence</span></h2>
  <p class="vt-section-sub">One command turns a bug video into a structured, timestamped bundle.</p>
  <PipelineFlow />
</div>

<div class="vt-section">
  <h2 class="vt-section-title">Why <span class="vt-gradient-text">vidtrace</span></h2>
  <p class="vt-section-sub">Stop scrubbing through recordings. Start citing evidence.</p>
  <ComparisonTable />
</div>

<div class="vt-section">
  <h2 class="vt-section-title">Try it in <span class="vt-gradient-text">one command</span></h2>
  <p class="vt-section-sub">Extract, search, and investigate — all from the terminal.</p>
  <div class="vt-code-grid">
    <div class="vt-code-card">
      <h4>Extract evidence</h4>
      ```bash
      vidtrace extract bug.mp4 --json
      ```
      Produces frames, OCR, transcript, metadata, and timeline in a timestamped bundle directory.
    </div>
    <div class="vt-code-card">
      <h4>Search evidence</h4>
      ```bash
      vidtrace index bundle/ --db evidence.veclite --json
      vidtrace search evidence.veclite "ticket click does not work" --json
      ```
      BM25 keyword search by default, optional semantic and hybrid via Ollama.
    </div>
    <div class="vt-code-card">
      <h4>Investigate to code</h4>
      ```bash
      vidtrace investigate bundle/ \
        --query "ticket click does not work" \
        --codebase ./app --connect --json
      ```
      Video evidence plus real file:line code matches via vecgrep.
    </div>
    <div class="vt-code-card">
      <h4>Compare a ticket</h4>
      ```bash
      vidtrace analyze bundle/ --ticket ticket.md
      vidtrace compare bundle/ --ticket ticket.md --json
      ```
      Heuristic ticket-vs-video scoring with confidence and term hits.
    </div>
  </div>
</div>

<div class="vt-section">
  <h2 class="vt-section-title">Built for <span class="vt-gradient-text">three audiences</span></h2>
  <p class="vt-section-sub">Humans, developers, and coding agents — all from the same bundle.</p>
  <div class="vt-audience">
    <div class="vt-audience-card">
      <h3>QA &amp; Support</h3>
      <p>Receive a bug video, extract timestamped evidence, cut clips and GIFs for tickets, and compare what the reporter said against what the recording shows.</p>
    </div>
    <div class="vt-audience-card">
      <h3>Developers</h3>
      <p>Stop guessing from vague repro notes. Open the Studio, jump to the frame, read the transcript, and trace the evidence straight into a code fix.</p>
    </div>
    <div class="vt-audience-card">
      <h3>Coding agents</h3>
      <p>Agents can't watch video. They can read JSON. vidtrace gives them timeline.json, OCR text, transcripts, an MCP server, and investigation handoffs.</p>
    </div>
  </div>
</div>

<div class="vt-section">
  <div class="vt-stats">
    <div class="vt-stat">
      <div class="vt-stat-num">v0.19.0</div>
      <div class="vt-stat-label">Latest release</div>
    </div>
    <div class="vt-stat">
      <div class="vt-stat-num">11</div>
      <div class="vt-stat-label">CLI commands</div>
    </div>
    <div class="vt-stat">
      <div class="vt-stat-num">5</div>
      <div class="vt-stat-label">Transcript formats</div>
    </div>
    <div class="vt-stat">
      <div class="vt-stat-num">MIT</div>
      <div class="vt-stat-label">Open source</div>
    </div>
  </div>
</div>

<div class="vt-section">
  <h2 class="vt-section-title">Frequently asked <span class="vt-gradient-text">questions</span></h2>
  <p class="vt-section-sub">Everything you need to know before installing.</p>
  <FAQAccordion />
</div>

<div class="vt-section">
  <div class="vt-cta">
    <h2>Start extracting evidence in minutes</h2>
    <p>Install with Homebrew or build from source. ffmpeg, tesseract, and whisper are the only runtime deps.</p>
    <div class="vt-cta-btns">
      <a class="vt-cta-btn vt-cta-btn-primary" href="/install">Install guide</a>
      <a class="vt-cta-btn vt-cta-btn-ghost" href="https://github.com/abdul-hamid-achik/vidtrace">View on GitHub</a>
    </div>
  </div>
</div>

## Documentation

### Start

- [Install](/install) — install the CLI and runtime media tools.
- [Usage](/usage) — run human and agent workflows.
- [Analysis](/analysis) — compare tickets with extracted video evidence.
- [Studio](/studio) — inspect a generated bundle in the terminal Studio.
- [Clip](/clip) — cut clips, make GIFs, and stitch videos from timestamp ranges.

### Reference

- [CLI Contract](/cli-contract) — command surface, flags, exit codes, and JSON output.
- [Artifact Schema](/artifact-schema) — generated bundle layout and JSON schemas.
- [Testing](/testing) — local, CI, smoke, and glyphrun verification.
- [Release](/release) — GitHub Actions, GoReleaser, and Homebrew tap publishing.
- [Documentation Site](/site) — VitePress and Vercel deployment.

### Architecture

- [Architecture](/architecture) — component boundaries and pipeline shape.
- [Roadmap](/roadmap) — iteration plan.