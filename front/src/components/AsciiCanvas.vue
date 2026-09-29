<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const FONT_SIZE = 11
const LINE_HEIGHT = 1.15
const FONT = `${FONT_SIZE}px "JetBrains Mono", "Courier New", monospace`
const BG = '#0a0b0d'
const DEFAULT_FG = '#e0e0e0'

const canvasRef = ref<HTMLCanvasElement | null>(null)
const containerRef = ref<HTMLDivElement | null>(null)

let ctx: CanvasRenderingContext2D | null = null
let charW = 0
let charH = 0
let resizeObserver: ResizeObserver | null = null
let pendingFrame: string | null = null
let rafId: number | null = null

onMounted(() => {
  if (!canvasRef.value || !containerRef.value) return
  ctx = canvasRef.value.getContext('2d', { alpha: false })!
  measureFont()
  fitCanvas()

  resizeObserver = new ResizeObserver(() => fitCanvas())
  resizeObserver.observe(containerRef.value)
})

onUnmounted(() => {
  if (rafId !== null) cancelAnimationFrame(rafId)
  resizeObserver?.disconnect()
})

function measureFont() {
  if (!ctx) return
  ctx.font = FONT
  charW = ctx.measureText('M').width
  charH = Math.ceil(FONT_SIZE * LINE_HEIGHT)
}

function fitCanvas() {
  if (!canvasRef.value || !containerRef.value || !ctx) return
  const { width, height } = containerRef.value.getBoundingClientRect()
  canvasRef.value.width = Math.floor(width)
  canvasRef.value.height = Math.floor(height)
  // measureFont must be re-called after canvas resize (context state resets).
  measureFont()
  ctx.fillStyle = BG
  ctx.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
}

// Minimal ANSI parser — handles exactly the sequences the backend emits:
//   \x1b[38;2;R;G;Bm  — 24-bit foreground color
//   \x1b[0m            — reset foreground to default
//   \x1b[H             — cursor to (0, 0)
//   \x1b[2J            — clear screen
//   \n                 — newline
//   printable char     — draw at current cell position
function renderFrame(ansi: string) {
  if (!ctx || !canvasRef.value) return

  // Clear canvas.
  ctx.fillStyle = BG
  ctx.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
  ctx.font = FONT

  let col = 0
  let row = 0
  let fg = DEFAULT_FG
  let i = 0
  const len = ansi.length

  while (i < len) {
    const ch = ansi.charAt(i)

    if (ch === '\x1b' && ansi.charAt(i + 1) === '[') {
      // CSI sequence: collect up to the final byte (a letter).
      i += 2
      let seq = ''
      while (i < len && !/[A-Za-z]/.test(ansi.charAt(i))) {
        seq += ansi.charAt(i++)
      }
      const cmd = ansi.charAt(i++) // the final letter

      if (cmd === 'm') {
        if (seq === '0' || seq === '') {
          fg = DEFAULT_FG
        } else if (seq.startsWith('38;2;')) {
          const parts = seq.split(';')
          // parts: ['38','2','R','G','B']
          fg = `rgb(${parts[2]},${parts[3]},${parts[4]})`
        }
      } else if (cmd === 'H') {
        col = 0
        row = 0
      } else if (cmd === 'J') {
        // Already cleared at the top; skip.
      }
      continue
    }

    if (ch === '\n') {
      row++
      col = 0
      i++
      continue
    }

    // Printable character — draw it.
    if (ch >= ' ') {
      const x = col * charW
      const y = row * charH + FONT_SIZE
      ctx.fillStyle = fg
      ctx.fillText(ch, x, y)
      col++
    }

    i++
  }
}

function flushFrame() {
  rafId = null
  if (pendingFrame === null) return
  const frame = pendingFrame
  pendingFrame = null
  renderFrame(frame)
}

function write(text: string) {
  pendingFrame = text
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
  if (ctx && canvasRef.value) {
    ctx.fillStyle = BG
    ctx.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
  }
}

function getDimensions(): { cols: number; rows: number } | null {
  if (!canvasRef.value || charW === 0 || charH === 0) return null
  return {
    cols: Math.floor(canvasRef.value.width / charW),
    rows: Math.floor(canvasRef.value.height / charH),
  }
}

defineExpose({ write, reset, getDimensions })
</script>

<template>
  <div ref="containerRef" class="ascii-canvas-container">
    <canvas ref="canvasRef" class="ascii-canvas" />
  </div>
</template>

<style scoped>
.ascii-canvas-container {
  width: 100%;
  height: 100%;
  background: #0a0b0d;
  overflow: hidden;
}

.ascii-canvas {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
