<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  content: string
  // When true, subsequent writes overwrite in place (video mode).
  // When false (default), the terminal is reset before each write (image mode).
  streaming?: boolean
}>()

const containerRef = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let firstWrite = true

onMounted(() => {
  if (!containerRef.value) return

  term = new Terminal({
    fontFamily: '"JetBrains Mono", "Cascadia Code", "Fira Code", monospace',
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
    // No scrollback — frames overwrite in place, scrollbar causes layout jitter.
    scrollback: 0,
    cursorStyle: 'block',
    cursorBlink: false,
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(containerRef.value)
  fitAddon.fit()

  resizeObserver = new ResizeObserver(() => fitAddon?.fit())
  resizeObserver.observe(containerRef.value)

  if (props.content) {
    writeContent(props.content)
  }
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  term?.dispose()
})

function writeContent(text: string) {
  if (!term) return

  if (firstWrite) {
    // Full reset only on the very first frame to set a clean baseline.
    term.reset()
    firstWrite = false
    term.write(text)
    return
  }

  // Subsequent frames: move cursor to top-left and overwrite in place.
  // \x1b[H  — cursor to row 1, col 1
  // \x1b[3J — clear scrollback (xterm.js extension, harmless if unsupported)
  // No full reset → no repaint flash.
  term.write('\x1b[H' + text)
}

watch(
  () => props.content,
  (val) => {
    if (val) writeContent(val)
  },
)

// Expose reset so parent can force a clean slate when switching files.
defineExpose({
  reset() {
    firstWrite = true
    term?.reset()
  },
})
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
