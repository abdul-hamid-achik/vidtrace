<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";

/**
 * Animated terminal that types out the vidtrace extract command
 * and reveals realistic extraction output step by step.
 * Includes a replay button.
 */

const displayed = ref<string[]>([]);
const done = ref(false);

const lines: { text: string; cls: string; typed?: boolean }[] = [
  { text: "$ vidtrace extract bug.mp4 --json", cls: "cmd", typed: true },
  { text: "", cls: "out" },
  { text: "  extracting frames ........ 94 frames @ 1 fps", cls: "step" },
  { text: "  running OCR .............. 94 frame texts", cls: "step" },
  { text: "  transcribing audio ....... whisper small", cls: "step" },
  { text: "  building timeline ........ 94 entries", cls: "step" },
  { text: "", cls: "out" },
  { text: "  output_dir: bug_artifacts_20260712_143020", cls: "file" },
  { text: "  timeline.json  metadata.json  ocr/  frames/", cls: "out" },
  { text: "", cls: "out" },
  { text: '"ok": true', cls: "ok" },
];

let timers: ReturnType<typeof setTimeout>[] = [];

function clearTimers() {
  timers.forEach(clearTimeout);
  timers = [];
}

function schedule(lineIndex: number, charIndex: number) {
  const line = lines[lineIndex];
  if (!line) {
    done.value = true;
    return;
  }

  if (!line.typed) {
    timers.push(
      setTimeout(() => {
        displayed.value.push(line.text);
        schedule(lineIndex + 1, 0);
      }, 280),
    );
    return;
  }

  if (charIndex <= line.text.length) {
    displayed.value[lineIndex] = line.text.slice(0, charIndex);
    timers.push(
      setTimeout(() => schedule(lineIndex, charIndex + 1), 35 + Math.random() * 40),
    );
  } else {
    timers.push(
      setTimeout(() => schedule(lineIndex + 1, 0), 400),
    );
  }
}

function play() {
  clearTimers();
  displayed.value = [];
  done.value = false;
  timers.push(setTimeout(() => schedule(0, 0), 300));
}

function replay() {
  play();
}

onMounted(() => {
  timers.push(setTimeout(() => schedule(0, 0), 600));
});

onUnmounted(() => {
  clearTimers();
});
</script>

<template>
  <div class="vt-terminal">
    <div class="vt-terminal-bar">
      <span class="vt-terminal-dot red"></span>
      <span class="vt-terminal-dot yellow"></span>
      <span class="vt-terminal-dot green"></span>
      <span class="vt-terminal-title">vidtrace — bash</span>
      <button class="vt-terminal-replay" @click="replay" v-if="done">Replay</button>
    </div>
    <div class="vt-terminal-body">
      <div v-for="(line, i) in displayed" :key="i">
        <span v-if="i === 0" class="vt-terminal-prompt">$&nbsp;</span><span :class="'vt-terminal-' + lines[i].cls">{{ line }}</span><span v-if="i === displayed.length - 1 && !done" class="vt-terminal-cursor"></span>
      </div>
      <span v-if="done" class="vt-terminal-cursor"></span>
    </div>
  </div>
</template>