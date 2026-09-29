<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  content: string
  active?: boolean
}>()

const containerRef = ref<HTMLDivElement | null>(null)
let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (!containerRef.value) return

  term = new Terminal({
    fontFamily: '"Cascadia Code", "Fira Code", "JetBrains Mono", monospace',
    fontSize: 11,
    lineHeight: 1.0,
    theme: {
      background: '#0d0d0d',
      foreground: '#e0e0e0',
      cursor: '#00ff41',
    },
    convertEol: true,
    disableStdin: true,
    scrollback: 0,
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
  term.reset()
  term.write(text)
}

watch(
  () => props.content,
  (val) => {
    if (val) writeContent(val)
  },
)
</script>

<template>
  <div ref="containerRef" class="terminal-container" />
</template>

<style scoped>
.terminal-container {
  width: 100%;
  height: 100%;
  min-height: 300px;
  background: #0d0d0d;
  border-radius: 6px;
  overflow: hidden;
}
</style>
