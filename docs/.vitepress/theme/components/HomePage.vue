<script setup lang="ts">
import { ref } from "vue";
import EvidenceFrame from "./EvidenceFrame.vue";

const brew =
  "brew tap abdul-hamid-achik/tap && brew install --cask abdul-hamid-achik/tap/vidtrace";

const copied = ref(false);

async function copyInstall() {
  try {
    await navigator.clipboard.writeText(brew);
    copied.value = true;
    window.setTimeout(() => {
      copied.value = false;
    }, 1600);
  } catch {
    copied.value = false;
  }
}

const loop = [
  { t: "Record", d: "A screen recording of the bug." },
  { t: "Extract", d: "Frames, OCR, transcript, timeline." },
  { t: "Cite", d: "A second, a frame, spoken words." },
  { t: "Connect", d: "Optional file:line in the repo." },
];

const moreCommands = [
  { label: "extract", line: "vidtrace extract bug.mp4 --json" },
  { label: "search", line: 'vidtrace search evidence.veclite "ticket click" --json' },
  { label: "compare", line: "vidtrace compare bundle/ --ticket ticket.md --json" },
  { label: "studio", line: "vidtrace studio bundle/" },
];

const audiences = [
  {
    title: "QA and support",
    body: "Turn a reporter video into timestamped evidence, then cut a clip or GIF for the ticket.",
  },
  {
    title: "Developers",
    body: "Stop guessing from vague notes. Open the frame, read the transcript, and jump to code.",
  },
  {
    title: "Coding agents",
    body: "Agents cannot watch a video. They can read timeline.json, OCR, and MCP tool results.",
  },
];

const faqs = [
  {
    q: "What does vidtrace actually do?",
    a: "It takes a bug screen recording and writes a local evidence bundle: frames, per-frame OCR, a Whisper transcript, ffprobe metadata, and timeline.json that maps each frame to overlapping speech.",
  },
  {
    q: "What runtime tools do I need?",
    a: "ffmpeg, ffprobe, tesseract, and whisper. Run vidtrace doctor after install. Optional: Ollama, fcheap, vecgrep, and codemap.",
  },
  {
    q: "Can coding agents use it?",
    a: "Yes. Every mutating or query command supports --json. vidtrace mcp exposes read-only tools over stdio (validate, search, compare, analyze, investigate, timeline, frame, and more).",
  },
  {
    q: "Does it upload my videos?",
    a: "No. Extraction runs on your machine. Optional Ollama embeddings also stay local.",
  },
  {
    q: "How do I install it?",
    a: "Homebrew cask is the fastest path on macOS. Linux .deb and .rpm packages ship with each release. Or build from source with task build.",
  },
  {
    q: "What is timeline.json?",
    a: "The main agent-facing artifact. Each entry is a timestamp, a frame path, OCR text, overlapping transcript segments, and visual_delta when the UI changed.",
  },
];
</script>

<template>
  <div class="vt-home">
    <section class="vt-hero">
      <div class="vt-hero-copy">
        <h1>Bug video evidence, timestamped.</h1>
        <p class="vt-hero-sub">
          Turn a screen recording into frames, OCR, transcript, and a timeline agents can quote.
        </p>
        <div class="vt-hero-actions">
          <a class="vt-btn vt-btn-primary" href="/install">Install</a>
          <a class="vt-btn vt-btn-ghost" href="/usage">Quick start</a>
        </div>
      </div>
      <EvidenceFrame />
    </section>

    <div class="vt-install">
      <button type="button" class="vt-install-btn" @click="copyInstall">
        <span class="vt-prompt">$</span>
        <code>{{ brew }}</code>
        <span class="vt-install-hint" aria-live="polite">{{
          copied ? "Copied" : "Copy"
        }}</span>
      </button>
    </div>

    <section class="vt-section">
      <h2>From recording to a citable second</h2>
      <ol class="vt-strip">
        <li v-for="step in loop" :key="step.t">
          <span class="vt-strip-title">{{ step.t }}</span>
          <span class="vt-strip-body">{{ step.d }}</span>
        </li>
      </ol>
    </section>

    <section class="vt-section vt-split-cite">
      <div class="vt-cite-col">
        <h2>The ticket</h2>
        <p class="vt-cite-ticket">
          Clicking a task does not take me to the assessment.
        </p>
      </div>
      <div class="vt-cite-col">
        <h2>The second</h2>
        <p class="vt-cite-hit">
          <span class="vt-timecode">00:18.40</span>
          <span class="vt-cite-ocr">OPG-14010 &nbsp; Task element</span>
          <span class="vt-cite-audio">I clicked here and it does not work.</span>
        </p>
      </div>
    </section>

    <section class="vt-section">
      <h2>One command from a raw video</h2>
      <pre class="vt-code" tabindex="0"><code>vidtrace investigate --video bug.mp4 \
  --query "ticket click does not work" \
  --json</code></pre>
      <pre class="vt-code vt-code-out" tabindex="0"><code>{
  "ok": true,
  "query": "ticket click does not work",
  "mode": "keyword",
  "evidence": [
    {
      "time_seconds": 18.4,
      "frame": "frames/frame_0019.png",
      "ocr": "Tickets OPG-14010 Task element",
      "transcript": "I clicked here and it does not work."
    }
  ]
}</code></pre>
      <ul class="vt-cmd-list">
        <li v-for="item in moreCommands" :key="item.label">
          <span>{{ item.label }}</span>
          <code>{{ item.line }}</code>
        </li>
      </ul>
    </section>

    <section class="vt-section">
      <h2>Same bundle, three readers</h2>
      <ul class="vt-audience">
        <li v-for="item in audiences" :key="item.title">
          <h3>{{ item.title }}</h3>
          <p>{{ item.body }}</p>
        </li>
      </ul>
    </section>

    <section class="vt-section">
      <h2>Questions</h2>
      <dl class="vt-faq">
        <template v-for="item in faqs" :key="item.q">
          <dt>{{ item.q }}</dt>
          <dd>{{ item.a }}</dd>
        </template>
      </dl>
    </section>

    <section class="vt-section vt-close">
      <h2>Install and read the contract</h2>
      <p>Homebrew or source. ffmpeg, tesseract, and whisper are the runtime tools.</p>
      <div class="vt-hero-actions">
        <a class="vt-btn vt-btn-primary" href="/install">Install</a>
        <a class="vt-btn vt-btn-ghost" href="/cli-contract">CLI contract</a>
      </div>
    </section>
  </div>
</template>
