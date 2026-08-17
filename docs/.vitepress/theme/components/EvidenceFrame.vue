<template>
  <figure class="vt-frame" aria-label="Sample timeline evidence at 18.4 seconds">
    <div class="vt-frame-sprockets" aria-hidden="true">
      <span v-for="n in 9" :key="'l' + n" />
    </div>
    <div class="vt-frame-body">
      <header class="vt-frame-meta">
        <time class="vt-timecode" datetime="PT18.4S">00:18.40</time>
        <span class="vt-delta" title="visual_delta 0.31">~ 0.31</span>
      </header>
      <p class="vt-frame-path">frames/frame_0019.png</p>
      <dl class="vt-frame-evidence">
        <div>
          <dt>OCR</dt>
          <dd>Tickets &nbsp; OPG-14010 &nbsp; Task element</dd>
        </div>
        <div>
          <dt>Audio</dt>
          <dd>I clicked here and it does not work.</dd>
        </div>
      </dl>
    </div>
    <div class="vt-frame-sprockets" aria-hidden="true">
      <span v-for="n in 9" :key="'r' + n" />
    </div>
  </figure>
</template>
