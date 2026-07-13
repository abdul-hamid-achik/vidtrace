<script setup lang="ts">
/**
 * Visual pipeline: bug video -> extract -> evidence bundle -> search / analyze / investigate.
 * Pure CSS/SVG, no dependencies.
 */
</script>

<template>
  <div class="vt-pipeline">
    <div class="vt-pipe-node">
      <span class="vt-pipe-icon">🎥</span>
      <span class="vt-pipe-label">Bug video</span>
      <span class="vt-pipe-sub">screen recording</span>
    </div>
    <div class="vt-pipe-arrow">→</div>
    <div class="vt-pipe-node">
      <span class="vt-pipe-icon">⚙️</span>
      <span class="vt-pipe-label">extract</span>
      <span class="vt-pipe-sub">ffmpeg · tesseract · whisper</span>
    </div>
    <div class="vt-pipe-arrow">→</div>
    <div class="vt-pipe-node">
      <span class="vt-pipe-icon">📦</span>
      <span class="vt-pipe-label">Evidence bundle</span>
      <span class="vt-pipe-sub">frames · OCR · transcript · timeline</span>
    </div>
    <div class="vt-pipe-arrow">→</div>
    <div class="vt-pipe-node">
      <span class="vt-pipe-icon">🔎</span>
      <span class="vt-pipe-label">search · analyze</span>
      <span class="vt-pipe-sub">index · compare · investigate</span>
    </div>
  </div>
</template>