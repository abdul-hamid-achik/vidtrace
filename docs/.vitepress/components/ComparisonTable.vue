<script setup lang="ts">
/**
 * Before/after comparison table: manual workflow vs vidtrace.
 */

const rows = [
  {
    task: "Find the exact moment a bug occurs",
    without: "Scrub through a 10-minute recording, guess timestamps",
    with: "Jump to the timeline entry, OCR + transcript at that second",
  },
  {
    task: "Share evidence in a ticket",
    without: "Upload a huge video, write vague repro steps",
    with: "Link a timestamped frame, paste OCR + transcript text",
  },
  {
    task: "Hand off to a coding agent",
    without: "Agent can't watch video; paste screenshots manually",
    with: "Agent reads timeline.json, searches evidence, investigates code",
  },
  {
    task: "Search across past bug videos",
    without: "Re-watch every recording, no index",
    with: "Index all bundles, keyword/semantic/hybrid search in seconds",
  },
  {
    task: "Compare a ticket claim with the recording",
    without: "Manual cross-reference, subjective assessment",
    with: "vidtrace compare scores confidence with term hits",
  },
  {
    task: "Cut a clip for a README or chat",
    without: "Open a video editor, export, compress",
    with: "vidtrace clip cut --label in one terminal command",
  },
];
</script>

<template>
  <div class="vt-comparison">
    <div class="vt-comparison-row">
      <div class="vt-comparison-header col-task">Task</div>
      <div class="vt-comparison-header col-without">Without vidtrace</div>
      <div class="vt-comparison-header col-with">With vidtrace</div>
    </div>
    <div v-for="(row, i) in rows" :key="i" class="vt-comparison-row">
      <div class="vt-comparison-cell col-task">{{ row.task }}</div>
      <div class="vt-comparison-cell col-without">{{ row.without }}</div>
      <div class="vt-comparison-cell col-with">{{ row.with }}</div>
    </div>
  </div>
</template>