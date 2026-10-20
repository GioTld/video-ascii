<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{ letterbox?: boolean }>()
const emit = defineEmits<{ resize: [dimensions: { cols: number; rows: number }] }>()

const FONT_SIZE = 11
const LINE_HEIGHT = 1.15
const FONT = `${FONT_SIZE}px "JetBrains Mono", "Courier New", monospace`
const BG_R = 10 / 255
const BG_G = 11 / 255
const BG_B = 13 / 255
const DEFAULT_FG: [number, number, number] = [0.88, 0.88, 0.88]

const ANSI_BASE_COLORS: [number, number, number][] = [
  [0x00 / 255, 0x00 / 255, 0x00 / 255], [0x80 / 255, 0x00 / 255, 0x00 / 255],
  [0x00 / 255, 0x80 / 255, 0x00 / 255], [0x80 / 255, 0x80 / 255, 0x00 / 255],
  [0x00 / 255, 0x00 / 255, 0x80 / 255], [0x80 / 255, 0x00 / 255, 0x80 / 255],
  [0x00 / 255, 0x80 / 255, 0x80 / 255], [0xc0 / 255, 0xc0 / 255, 0xc0 / 255],
  [0x80 / 255, 0x80 / 255, 0x80 / 255], [0xff / 255, 0x00 / 255, 0x00 / 255],
  [0x00 / 255, 0xff / 255, 0x00 / 255], [0xff / 255, 0xff / 255, 0x00 / 255],
  [0x00 / 255, 0x00 / 255, 0xff / 255], [0xff / 255, 0x00 / 255, 0xff / 255],
  [0x00 / 255, 0xff / 255, 0xff / 255], [0xff / 255, 0xff / 255, 0xff / 255],
]

function ansi256ColorNorm(code: number): [number, number, number] {
  if (code < 16) return ANSI_BASE_COLORS[code] ?? DEFAULT_FG
  if (code < 232) {
    const idx = code - 16
    const ch = (v: number) => (v === 0 ? 0 : (55 + v * 40) / 255)
    return [ch(Math.floor(idx / 36)), ch(Math.floor(idx / 6) % 6), ch(idx % 6)]
  }
  const gray = (8 + (code - 232) * 10) / 255
  return [gray, gray, gray]
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
const containerRef = ref<HTMLDivElement | null>(null)

let gl: WebGLRenderingContext | null = null
let charW = 0
let charH = 0
let resizeObserver: ResizeObserver | null = null
let pendingFrame: string | null = null
let lastFrame: string | null = null
let rafId: number | null = null
let lastCols = 0
let lastRows = 0
let canvasInitialized = false

// Glyph Atlas specs (16x16 = 256 glyphs: 0-127 ASCII + half-block + braille 0x2800-0x287F)
const ATLAS_COLS = 16
const ATLAS_ROWS = 16
let atlasTex: WebGLTexture | null = null
let glProgram: WebGLProgram | null = null
let quadVbo: WebGLBuffer | null = null
let instVbo: WebGLBuffer | null = null
let extInst: ANGLE_instanced_arrays | null = null

// Instance buffer max capacity: 256 cols * 160 rows = 40960 cells
const MAX_CELLS = 40960
// Floats per cell: [posX, posY, glyphCol, glyphRow, fgR, fgG, fgB, bgR, bgG, bgB] = 10 floats
const FLOATS_PER_CELL = 10
const instanceData = new Float32Array(MAX_CELLS * FLOATS_PER_CELL)

// Shader sources
const VS_SRC = `
precision highp float;
attribute vec2 aCorner;
attribute vec2 aPos;
attribute vec2 aGlyphUV;
attribute vec3 aFgColor;
attribute vec3 aBgColor;

uniform vec2 uViewport;
uniform vec2 uCharSize;
uniform vec2 uOffset;

varying vec2 vUV;
varying vec3 vFgColor;
varying vec3 vBgColor;

void main() {
  vec2 pixelPos = uOffset + aPos * uCharSize + aCorner * uCharSize;
  vec2 clipPos = (pixelPos / uViewport) * 2.0 - 1.0;
  clipPos.y = -clipPos.y;
  gl_Position = vec4(clipPos, 0.0, 1.0);

  vec2 atlasCell = vec2(1.0 / ${ATLAS_COLS.toFixed(1)}, 1.0 / ${ATLAS_ROWS.toFixed(1)});
  vUV = aGlyphUV * atlasCell + aCorner * atlasCell;
  vFgColor = aFgColor;
  vBgColor = aBgColor;
}
`

const FS_SRC = `
precision mediump float;
uniform sampler2D uAtlas;
varying vec2 vUV;
varying vec3 vFgColor;
varying vec3 vBgColor;

void main() {
  float alpha = texture2D(uAtlas, vUV).a;
  vec3 color = mix(vBgColor, vFgColor, alpha);
  gl_FragColor = vec4(color, 1.0);
}
`

function buildGlyphAtlas(): HTMLCanvasElement {
  const atlasCanvas = document.createElement('canvas')
  atlasCanvas.width = Math.ceil(charW * ATLAS_COLS)
  atlasCanvas.height = Math.ceil(charH * ATLAS_ROWS)
  const actx = atlasCanvas.getContext('2d', { alpha: true })!
  actx.clearRect(0, 0, atlasCanvas.width, atlasCanvas.height)
  actx.font = FONT
  actx.textBaseline = 'top'
  actx.fillStyle = '#ffffff'

  // Slot 0-127: standard ASCII
  for (let code = 32; code < 127; code++) {
    const gx = code % ATLAS_COLS
    const gy = Math.floor(code / ATLAS_COLS)
    actx.fillText(String.fromCharCode(code), gx * charW, gy * charH)
  }

  // Slot 128: half-block '▀' (U+2580)
  const halfH = Math.ceil(charH / 2)
  const hx = 128 % ATLAS_COLS
  const hy = Math.floor(128 / ATLAS_COLS)
  actx.fillRect(hx * charW, hy * charH, charW, halfH)

  // Slot 129: lower half-block '▄' (U+2584)
  const lx = 129 % ATLAS_COLS
  const ly = Math.floor(129 / ATLAS_COLS)
  actx.fillRect(lx * charW, ly * charH + halfH, charW, charH - halfH)

  // Slot 130..255: Braille patterns (U+2800..U+287D)
  for (let b = 0; b < 126; b++) {
    const slot = 130 + b
    const bx = slot % ATLAS_COLS
    const by = Math.floor(slot / ATLAS_COLS)
    actx.fillText(String.fromCharCode(0x2800 + b), bx * charW, by * charH)
  }

  return atlasCanvas
}

function initWebGL() {
  if (!canvasRef.value) return
  gl = canvasRef.value.getContext('webgl', { alpha: false, depth: false, antialias: false })
  if (!gl) return

  extInst = gl.getExtension('ANGLE_instanced_arrays')
  if (!extInst) {
    console.warn('ANGLE_instanced_arrays unsupported, falling back')
    return
  }

  // Compile shaders
  const vs = gl.createShader(gl.VERTEX_SHADER)!
  gl.shaderSource(vs, VS_SRC)
  gl.compileShader(vs)

  const fs = gl.createShader(gl.FRAGMENT_SHADER)!
  gl.shaderSource(fs, FS_SRC)
  gl.compileShader(fs)

  glProgram = gl.createProgram()!
  gl.attachShader(glProgram, vs)
  gl.attachShader(glProgram, fs)
  gl.linkProgram(glProgram)
  gl.useProgram(glProgram)

  // Unit quad vertices [x, y] (2 triangles)
  const quadVerts = new Float32Array([
    0, 0,
    1, 0,
    0, 1,
    0, 1,
    1, 0,
    1, 1,
  ])
  quadVbo = gl.createBuffer()
  gl.bindBuffer(gl.ARRAY_BUFFER, quadVbo)
  gl.bufferData(gl.ARRAY_BUFFER, quadVerts, gl.STATIC_DRAW)

  const aCornerLoc = gl.getAttribLocation(glProgram, 'aCorner')
  gl.enableVertexAttribArray(aCornerLoc)
  gl.vertexAttribPointer(aCornerLoc, 2, gl.FLOAT, false, 0, 0)

  // Instance VBO
  instVbo = gl.createBuffer()
  gl.bindBuffer(gl.ARRAY_BUFFER, instVbo)
  gl.bufferData(gl.ARRAY_BUFFER, instanceData.byteLength, gl.DYNAMIC_DRAW)

  const stride = FLOATS_PER_CELL * 4 // 40 bytes
  const setupInstAttr = (name: string, size: number, offsetFloats: number) => {
    const loc = gl!.getAttribLocation(glProgram!, name)
    gl!.enableVertexAttribArray(loc)
    gl!.vertexAttribPointer(loc, size, gl!.FLOAT, false, stride, offsetFloats * 4)
    extInst!.vertexAttribDivisorANGLE(loc, 1)
  }

  setupInstAttr('aPos', 2, 0)
  setupInstAttr('aGlyphUV', 2, 2)
  setupInstAttr('aFgColor', 3, 4)
  setupInstAttr('aBgColor', 3, 7)

  // Upload Atlas texture
  const atlasCanvas = buildGlyphAtlas()
  atlasTex = gl.createTexture()
  gl.bindTexture(gl.TEXTURE_2D, atlasTex)
  gl.pixelStorei(gl.UNPACK_PREMULTIPLY_ALPHA_WEBGL, false)
  gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, atlasCanvas)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
  gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
}

function measureFont() {
  const c = document.createElement('canvas')
  const mctx = c.getContext('2d')!
  mctx.font = FONT
  charW = mctx.measureText('M').width
  charH = Math.ceil(FONT_SIZE * LINE_HEIGHT)
}

function fitCanvas() {
  if (!canvasRef.value || !containerRef.value) return
  const { width, height } = containerRef.value.getBoundingClientRect()
  const nextWidth = Math.floor(width)
  const nextHeight = Math.floor(height)
  if (canvasInitialized && canvasRef.value.width === nextWidth && canvasRef.value.height === nextHeight) return
  canvasRef.value.width = nextWidth
  canvasRef.value.height = nextHeight
  canvasInitialized = true

  if (gl) {
    gl.viewport(0, 0, nextWidth, nextHeight)
    gl.clearColor(BG_R, BG_G, BG_B, 1.0)
    gl.clear(gl.COLOR_BUFFER_BIT)
  }

  if (lastFrame) renderFrame(lastFrame)
  const dimensions = getDimensions()
  if (dimensions && (dimensions.cols !== lastCols || dimensions.rows !== lastRows)) {
    lastCols = dimensions.cols
    lastRows = dimensions.rows
    emit('resize', dimensions)
  }
}

function getGlyphSlot(ch: string, code: number): number {
  if (ch === '▀') return 128
  if (ch === '▄') return 129
  if (code >= 0x2800 && code <= 0x287d) return 130 + (code - 0x2800)
  if (code >= 32 && code < 127) return code
  return 32 // space
}

function renderFrame(ansi: string) {
  if (!gl || !extInst || !glProgram || !canvasRef.value) return
  lastFrame = ansi

  const w = canvasRef.value.width
  const h = canvasRef.value.height
  const len = ansi.length

  let offsetX = 0
  let offsetY = 0

  if (props.letterbox) {
    let maxCols = 0
    let curCols = 0
    let lineCount = 0
    let inEsc = false
    for (let j = 0; j < len; j++) {
      const code = ansi.charCodeAt(j)
      if (code === 27) inEsc = true
      else if (inEsc) {
        if ((code >= 65 && code <= 90) || (code >= 97 && code <= 122)) inEsc = false
      } else if (code === 10) {
        if (curCols > maxCols) maxCols = curCols
        curCols = 0
        lineCount++
      } else if (code >= 32) {
        curCols++
      }
    }
    if (curCols > 0) {
      if (curCols > maxCols) maxCols = curCols
      lineCount++
    }
    offsetX = Math.max(0, Math.floor((w - maxCols * charW) / 2))
    offsetY = Math.max(0, Math.floor((h - lineCount * charH) / 2))
  }

  let col = 0
  let row = 0
  let fgR = DEFAULT_FG[0], fgG = DEFAULT_FG[1], fgB = DEFAULT_FG[2]
  let bgR = BG_R, bgG = BG_G, bgB = BG_B
  let i = 0
  let cellCount = 0

  while (i < len && cellCount < MAX_CELLS) {
    const ch = ansi.charAt(i)

    if (ch === '\x1b' && ansi.charAt(i + 1) === '[') {
      i += 2
      let p0 = -1, p1 = -1, p2 = -1, p3 = -1, p4 = -1
      let currentVal = 0
      let hasVal = false
      let paramIdx = 0

      while (i < len) {
        const code = ansi.charCodeAt(i)
        if (code >= 48 && code <= 57) {
          currentVal = currentVal * 10 + (code - 48)
          hasVal = true
        } else if (code === 59) {
          if (paramIdx === 0) p0 = hasVal ? currentVal : 0
          else if (paramIdx === 1) p1 = hasVal ? currentVal : 0
          else if (paramIdx === 2) p2 = hasVal ? currentVal : 0
          else if (paramIdx === 3) p3 = hasVal ? currentVal : 0
          else if (paramIdx === 4) p4 = hasVal ? currentVal : 0
          paramIdx++
          currentVal = 0
          hasVal = false
        } else if ((code >= 65 && code <= 90) || (code >= 97 && code <= 122)) {
          if (hasVal) {
            if (paramIdx === 0) p0 = currentVal
            else if (paramIdx === 1) p1 = currentVal
            else if (paramIdx === 2) p2 = currentVal
            else if (paramIdx === 3) p3 = currentVal
            else if (paramIdx === 4) p4 = currentVal
          }
          const cmd = ansi.charAt(i++)
          if (cmd === 'm') {
            if (p0 === 0 || p0 === -1) {
              fgR = DEFAULT_FG[0]; fgG = DEFAULT_FG[1]; fgB = DEFAULT_FG[2]
              bgR = BG_R; bgG = BG_G; bgB = BG_B
            } else if (p0 === 49) {
              bgR = BG_R; bgG = BG_G; bgB = BG_B
            } else if (p0 === 39) {
              fgR = DEFAULT_FG[0]; fgG = DEFAULT_FG[1]; fgB = DEFAULT_FG[2]
            } else if (p0 === 38) {
              if (p1 === 2 && p2 >= 0 && p3 >= 0 && p4 >= 0) {
                fgR = p2 / 255; fgG = p3 / 255; fgB = p4 / 255
              } else if (p1 === 5 && p2 >= 0) {
                const c = ansi256ColorNorm(p2)
                fgR = c[0]; fgG = c[1]; fgB = c[2]
              }
            } else if (p0 === 48) {
              if (p1 === 2 && p2 >= 0 && p3 >= 0 && p4 >= 0) {
                bgR = p2 / 255; bgG = p3 / 255; bgB = p4 / 255
              } else if (p1 === 5 && p2 >= 0) {
                const c = ansi256ColorNorm(p2)
                bgR = c[0]; bgG = c[1]; bgB = c[2]
              }
            }
          } else if (cmd === 'H') {
            col = 0
            row = 0
          }
          break
        }
        i++
      }
      continue
    }

    if (ch === '\n') {
      row++
      col = 0
      i++
      continue
    }

    const code = ansi.charCodeAt(i)
    if (code >= 32 || ch === '▀' || ch === '▄') {
      const slot = getGlyphSlot(ch, code)
      const gx = slot % ATLAS_COLS
      const gy = Math.floor(slot / ATLAS_COLS)

      const offset = cellCount * FLOATS_PER_CELL
      instanceData[offset] = col
      instanceData[offset + 1] = row
      instanceData[offset + 2] = gx
      instanceData[offset + 3] = gy
      instanceData[offset + 4] = fgR
      instanceData[offset + 5] = fgG
      instanceData[offset + 6] = fgB
      instanceData[offset + 7] = bgR
      instanceData[offset + 8] = bgG
      instanceData[offset + 9] = bgB

      cellCount++
      col++
    }

    i++
  }

  // Upload dynamic instance data
  gl.bindBuffer(gl.ARRAY_BUFFER, instVbo)
  gl.bufferSubData(gl.ARRAY_BUFFER, 0, instanceData.subarray(0, cellCount * FLOATS_PER_CELL))

  // Render 1 single draw call
  gl.clearColor(BG_R, BG_G, BG_B, 1.0)
  gl.clear(gl.COLOR_BUFFER_BIT)

  gl.useProgram(glProgram)
  gl.uniform2f(gl.getUniformLocation(glProgram, 'uViewport'), w, h)
  gl.uniform2f(gl.getUniformLocation(glProgram, 'uCharSize'), charW, charH)
  gl.uniform2f(gl.getUniformLocation(glProgram, 'uOffset'), offsetX, offsetY)

  extInst.drawArraysInstancedANGLE(gl.TRIANGLES, 0, 6, cellCount)
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
  if (gl && canvasRef.value) {
    gl.clearColor(BG_R, BG_G, BG_B, 1.0)
    gl.clear(gl.COLOR_BUFFER_BIT)
  }
}

function getDimensions(): { cols: number; rows: number } | null {
  if (!canvasRef.value || charW === 0 || charH === 0) return null
  const cols = Math.floor(canvasRef.value.width / charW)
  const rows = Math.floor(canvasRef.value.height / charH)
  return cols > 0 && rows > 0 ? { cols, rows } : null
}

onMounted(() => {
  measureFont()
  initWebGL()
  fitCanvas()

  if (containerRef.value) {
    resizeObserver = new ResizeObserver(() => fitCanvas())
    resizeObserver.observe(containerRef.value)
  }
})

onUnmounted(() => {
  if (rafId !== null) cancelAnimationFrame(rafId)
  resizeObserver?.disconnect()
  if (gl) {
    if (quadVbo) gl.deleteBuffer(quadVbo)
    if (instVbo) gl.deleteBuffer(instVbo)
    if (atlasTex) gl.deleteTexture(atlasTex)
    if (glProgram) gl.deleteProgram(glProgram)
  }
})

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
