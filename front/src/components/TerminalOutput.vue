<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import '@xterm/xterm/css/xterm.css'

const containerRef = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let firstWrite = true

// rAF-gating: only one frame is displayed per display refresh cycle.
// If frames arrive faster than the monitor refresh rate, intermediate
// frames are dropped and only the latest is rendered — no queuing, no noise.
let pendingFrame: string | null = null
let rafId: number | null = null

onMounted(() => {
  if (!containerRef.value) return

  term = new Terminal({
    fontFamily: '"JetBrains Mono", "Cascadia Code", monospace',
    fontSize: 11,
    lineHeight: 1.0,
    theme: {
      background: '#0a0b0d',
      foreground: '#e0e0e0',
      // Hide cursor to avoid a blinking artifact over ASCII content.
      cursor: '#0a0b0d',
      cursorAccent: '#0a0b0d',
    },
    convertEol: true,
    disableStdin: true,
    scrollback: 0,
    cursorStyle: 'block',
    cursorBlink: false,
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(containerRef.value)
  fitAddon.fit()

  // WebGL renderer for GPU-accelerated draw. Falls back to canvas on failure.
  try {
    const webgl = new WebglAddon()
    webgl.onContextLoss(() => webgl.dispose())
    term.loadAddon(webgl)
  } catch {
    // Canvas renderer is already active.
  }

  resizeObserver = new ResizeObserver(() => fitAddon?.fit())
  resizeObserver.observe(containerRef.value)
})

onUnmounted(() => {
  if (rafId !== null) cancelAnimationFrame(rafId)
  resizeObserver?.disconnect()
  term?.dispose()
})

function flushFrame() {
  rafId = null
  if (!term || pendingFrame === null) return

  const frame = pendingFrame
  pendingFrame = null

  if (firstWrite) {
    // Clean baseline on first frame.
    term.reset()
    firstWrite = false
    term.write(frame)
    return
  }

  // \x1b[2J clears the visible area.
  // \x1b[H  moves cursor to (1,1).
  // All in one write() call → xterm processes atomically, single render pass.
  term.write('\x1b[2J\x1b[H' + frame)
}

function write(text: string) {
  if (!term) return
  pendingFrame = text
  // Schedule a flush only if one is not already queued.
  if (rafId === null) {
    rafId = requestAnimationFrame(flushFrame)
  }
}

function reset() {
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  pendingFrame = null
  firstWrite = true
  term?.reset()
}

function getDimensions(): { cols: number; rows: number } | null {
  if (!fitAddon || !term) return null
  const dims = fitAddon.proposeDimensions()
  if (!dims) return null
  return { cols: dims.cols, rows: dims.rows }
}

defineExpose({ write, reset, getDimensions })
</script>

<template>
  <div ref="containerRef" class="terminal-container" />
</template>

<style scoped>
.terminal-container {
  width: 100%;
  height: 100%;
  background: #0a0b0d;
  overflow: hidden;
}
</style>
