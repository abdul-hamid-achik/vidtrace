<script setup lang="ts">
import { ref } from "vue";

/**
 * Interactive FAQ accordion with smooth expand/collapse.
 */

const openIndex = ref<number | null>(0);

const faqs = [
  {
    q: "What does vidtrace actually do?",
    a: "It takes a screen recording of a bug and produces a structured evidence bundle: extracted frames (PNG), OCR text per frame, a Whisper transcript in 5 formats, ffprobe metadata, and a timeline.json that maps every frame to overlapping transcript segments. Everything is timestamped and citable.",
  },
  {
    q: "What runtime tools do I need?",
    a: "ffmpeg, ffprobe, tesseract, and whisper. Run `vidtrace doctor` after install to verify. Optional tools: Ollama for semantic search, fcheap for bundle stashing, vecgrep for codebase search, codemap for structural code graph queries.",
  },
  {
    q: "Can coding agents use vidtrace?",
    a: "Yes. Every command emits stable JSON with `--json`. Agents read `output_dir` from stdout, then inspect `timeline.json`, `metadata.json`, OCR text, and selected frames. The MCP server (`vidtrace mcp`) exposes read-only tools over stdio for validate, search, compare, analyze, and investigate.",
  },
  {
    q: "Does it upload my videos anywhere?",
    a: "No. vidtrace is local-first. All extraction runs on your machine with ffmpeg, tesseract, and whisper. No cloud uploads. Optional Ollama embeddings also run locally.",
  },
  {
    q: "How do I install it?",
    a: "The fastest path is Homebrew: `brew tap abdul-hamid-achik/tap && brew install --cask abdul-hamid-achik/tap/vidtrace`. Linux .deb and .rpm packages are also published. Or build from source with `task build`.",
  },
  {
    q: "What is timeline.json?",
    a: "The main agent-facing artifact. It maps every extracted frame to its OCR text and any overlapping transcript segments, with second-accurate timestamps. It is the structured, citable evidence that replaces vague reproduction notes.",
  },
  {
    q: "Can I search across multiple bug videos?",
    a: "Yes. `vidtrace index` accepts multiple bundle paths (shell globs) and indexes them into one VecLite database. Search with keyword, semantic, or hybrid mode, filtered by bundle, source video, evidence source, and time window.",
  },
];

function toggle(i: number) {
  openIndex.value = openIndex.value === i ? null : i;
}
</script>

<template>
  <div class="vt-faq">
    <div
      v-for="(faq, i) in faqs"
      :key="i"
      class="vt-faq-item"
      :class="{ 'vt-faq-open': openIndex === i }"
    >
      <button class="vt-faq-q" @click="toggle(i)">
        <span>{{ faq.q }}</span>
        <span class="vt-faq-icon">+</span>
      </button>
      <div class="vt-faq-a">
        <div class="vt-faq-a-inner" v-html="faq.a"></div>
      </div>
    </div>
  </div>
</template>