<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
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
    // WebGL renderer is used automatically by xterm when available.
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(containerRef.value)
  fitAddon.fit()

  resizeObserver = new ResizeObserver(() => fitAddon?.fit())
  resizeObserver.observe(containerRef.value)
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  term?.dispose()
})

// write is called directly from the parent — no Vue reactivity in the hot path.
function write(text: string) {
  if (!term) return
  if (firstWrite) {
    term.reset()
    firstWrite = false
    term.write(text)
    return
  }
  // Overwrite in place: move cursor to origin, then paint the new frame.
  term.write('\x1b[H' + text)
}

function reset() {
  firstWrite = true
  term?.reset()
}

defineExpose({ write, reset })
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
