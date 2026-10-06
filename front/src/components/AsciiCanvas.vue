<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{ letterbox?: boolean }>()
const emit = defineEmits<{ resize: [dimensions: { cols: number; rows: number }] }>()

const FONT_SIZE = 11
const LINE_HEIGHT = 1.15
const FONT = `${FONT_SIZE}px "JetBrains Mono", "Courier New", monospace`
const BG = '#0a0b0d'
const DEFAULT_FG = '#e0e0e0'
const ANSI_BASE_COLORS = [
  '#000000', '#800000', '#008000', '#808000', '#000080', '#800080', '#008080', '#c0c0c0',
  '#808080', '#ff0000', '#00ff00', '#ffff00', '#0000ff', '#ff00ff', '#00ffff', '#ffffff',
]

function ansi256Color(code: number): string {
  if (code < 16) return ANSI_BASE_COLORS[code] ?? DEFAULT_FG
  if (code < 232) {
    const index = code - 16
    const channel = (value: number) => value === 0 ? 0 : 55 + value * 40
    return `rgb(${channel(Math.floor(index / 36))},${channel(Math.floor(index / 6) % 6)},${channel(index % 6)})`
  }
  const gray = 8 + (code - 232) * 10
  return `rgb(${gray},${gray},${gray})`
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const containerRef = ref<HTMLDivElement | null>(null)

let ctx: CanvasRenderingContext2D | null = null
let charW = 0
let charH = 0
let resizeObserver: ResizeObserver | null = null
let pendingFrame: string | null = null
let lastFrame: string | null = null
let rafId: number | null = null
let lastCols = 0
let lastRows = 0
let canvasInitialized = false

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
  const nextWidth = Math.floor(width)
  const nextHeight = Math.floor(height)
  if (canvasInitialized && canvasRef.value.width === nextWidth && canvasRef.value.height === nextHeight) return
  canvasRef.value.width = nextWidth
  canvasRef.value.height = nextHeight
  canvasInitialized = true
  // measureFont must be re-called after canvas resize (context state resets).
  measureFont()
  ctx.fillStyle = BG
  ctx.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
  if (lastFrame) renderFrame(lastFrame)
  const dimensions = getDimensions()
  if (dimensions && (dimensions.cols !== lastCols || dimensions.rows !== lastRows)) {
    lastCols = dimensions.cols
    lastRows = dimensions.rows
    emit('resize', dimensions)
  }
}

// Minimal ANSI parser — handles the sequences the backend emits:
//   \x1b[38;2;R;G;Bm / \x1b[48;2;R;G;Bm — 24-bit colors
//   \x1b[38;5;Nm / \x1b[48;5;Nm — 256-color palette
//   \x1b[0m / \x1b[49m — reset colors or background
//   \x1b[H             — cursor to (0, 0)
//   \x1b[2J            — clear screen
//   \n                 — newline
//   printable char     — draw at current cell position
function renderFrame(ansi: string) {
  if (!ctx || !canvasRef.value) return
  lastFrame = ansi

  // Clear canvas.
  ctx.fillStyle = BG
  ctx.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
  ctx.font = FONT

  let offsetX = 0
  let offsetY = 0
  if (props.letterbox) {
    const lines = ansi.replace(/\x1b\[[0-9;]*[A-Za-z]/g, '').split('\n')
    if (lines.at(-1) === '') lines.pop()
    const columns = lines.reduce((max, line) => Math.max(max, line.length), 0)
    offsetX = Math.max(0, Math.floor((canvasRef.value.width - columns * charW) / 2))
    offsetY = Math.max(0, Math.floor((canvasRef.value.height - lines.length * charH) / 2))
  }

  let col = 0
  let row = 0
  let fg = DEFAULT_FG
  let bg = BG
  let i = 0
  const len = ansi.length
  let runStart = -1
  let runCol = 0
  let runRow = 0

  function drawRun(end: number) {
    if (runStart < 0 || !ctx) return
    const x = offsetX + runCol * charW
    const y = offsetY + runRow * charH
    if (bg !== BG) {
      ctx.fillStyle = bg
      ctx.fillRect(x, y, (end - runStart) * charW, charH)
    }
    ctx.fillStyle = fg
    ctx.fillText(ansi.slice(runStart, end), x, y + FONT_SIZE)
    runStart = -1
  }

  while (i < len) {
    const ch = ansi.charAt(i)

    if (ch === '\x1b' && ansi.charAt(i + 1) === '[') {
      drawRun(i)
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
          bg = BG
        } else if (seq === '49') {
          bg = BG
        } else if (seq.startsWith('38;2;')) {
          const parts = seq.split(';')
          // parts: ['38','2','R','G','B']
          fg = `rgb(${parts[2]},${parts[3]},${parts[4]})`
        } else if (seq.startsWith('48;2;')) {
          const parts = seq.split(';')
          bg = `rgb(${parts[2]},${parts[3]},${parts[4]})`
        } else if (seq.startsWith('38;5;')) {
          fg = ansi256Color(Number(seq.slice(5)))
        } else if (seq.startsWith('48;5;')) {
          bg = ansi256Color(Number(seq.slice(5)))
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
      drawRun(i)
      row++
      col = 0
      i++
      continue
    }

    // Printable characters with the same colors share a canvas draw call.
    if (ch >= ' ') {
      if (runStart < 0) {
        runStart = i
        runCol = col
        runRow = row
      }
      col++
    } else {
      drawRun(i)
    }

    i++
  }
  drawRun(i)
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
  lastFrame = null
  if (ctx && canvasRef.value) {
    ctx.fillStyle = BG
    ctx.fillRect(0, 0, canvasRef.value.width, canvasRef.value.height)
  }
}

function getDimensions(): { cols: number; rows: number } | null {
  if (!canvasRef.value || charW === 0 || charH === 0) return null
  const cols = Math.floor(canvasRef.value.width / charW)
  const rows = Math.floor(canvasRef.value.height / charH)
  return cols > 0 && rows > 0 ? { cols, rows } : null
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
