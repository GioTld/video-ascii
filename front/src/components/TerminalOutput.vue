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

onMounted(() => {
  if (!containerRef.value) return

  term = new Terminal({
    fontFamily: '"JetBrains Mono", "Cascadia Code", monospace',
    fontSize: 11,
    lineHeight: 1.0,
    theme: {
      background: '#0a0b0d',
      foreground: '#e0e0e0',
      cursor: '#00ff41',
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

  // WebGL renderer: GPU-accelerated draw — falls back to canvas on unsupported contexts.
  try {
    const webgl = new WebglAddon()
    webgl.onContextLoss(() => webgl.dispose())
    term.loadAddon(webgl)
  } catch {
    // Canvas fallback is already active, nothing to do.
  }

  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
  })
  resizeObserver.observe(containerRef.value)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  term?.dispose()
})

function write(text: string) {
  if (!term) return
  if (firstWrite) {
    term.reset()
    firstWrite = false
    term.write(text)
    return
  }
  term.write('\x1b[H' + text)
}

function reset() {
  firstWrite = true
  term?.reset()
}

// Returns the number of columns and rows the terminal can fit in its current
// container size. HomeView uses this to send exact dimensions to the backend.
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
