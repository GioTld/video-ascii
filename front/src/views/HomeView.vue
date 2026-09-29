<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import TerminalOutput from '@/components/TerminalOutput.vue'
import { renderImage, uploadVideo, streamVideo } from '@/services/asciiService'

type LogType = 'info' | 'ok' | 'err' | 'cmd'
interface LogLine {
  id: number
  text: string
  type: LogType
}

interface Preset {
  label: string
  cols: number
  rows: number
}

let lineId = 0
function mkLine(text: string, type: LogType = 'info'): LogLine {
  return { id: lineId++, text, type }
}

const PRESETS: Preset[] = [
  { label: '40×20', cols: 40, rows: 20 },
  { label: '80×40', cols: 80, rows: 40 },
  { label: '120×60', cols: 120, rows: 60 },
  { label: '200×100', cols: 200, rows: 100 },
]

const FILTERS = [
  { label: 'none', value: 'none' },
  { label: 'grayscale', value: 'grayscale' },
  { label: 'sepia', value: 'sepia' },
  { label: 'invert', value: 'invert' },
]

const log = ref<LogLine[]>([
  mkLine('ASCII-RENDER v1.0.0 — ready', 'ok'),
  mkLine('drop a file or use [LOAD FILE] to begin', 'info'),
])
const logContainer = ref<HTMLDivElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const asciiContent = ref('')
const isProcessing = ref(false)
const mediaInfo = ref<string | null>(null)
const isVideo = ref(false)
const selectedFile = ref<File | null>(null)
const preset = ref<Preset>(PRESETS[1]!)
const colorMode = ref<'green' | 'amber' | 'white'>('green')
const selectedFilter = ref('none')
let activeStream: EventSource | null = null

const asciiClass = computed(() => {
  if (colorMode.value === 'amber') return 'ascii-output amber fade-in'
  if (colorMode.value === 'white') return 'ascii-output white fade-in'
  return 'ascii-output fade-in'
})

function addLog(text: string, type: LogType = 'info') {
  log.value = [...log.value.slice(-60), mkLine(text, type)]
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  })
}

function logColor(type: LogType): string {
  if (type === 'ok') return 'var(--green)'
  if (type === 'err') return 'var(--err)'
  if (type === 'cmd') return 'var(--cmd)'
  return 'var(--muted)'
}

function logPrefix(type: LogType): string {
  if (type === 'ok') return '[ok] '
  if (type === 'err') return '[!!] '
  if (type === 'cmd') return ''
  return '  >  '
}

function stopStream() {
  if (activeStream) {
    activeStream.close()
    activeStream = null
  }
  isProcessing.value = false
}

async function handleFile(file: File) {
  stopStream()
  selectedFile.value = file
  asciiContent.value = ''
  mediaInfo.value = null

  const type = file.type.startsWith('video/') ? 'video' : 'image'
  isVideo.value = type === 'video'

  addLog(`$ load "${file.name}"`, 'cmd')
  addLog(`type: ${type} | size: ${(file.size / 1024).toFixed(1)} KB`)

  isProcessing.value = true

  try {
    if (type === 'image') {
      addLog(`rendering at ${preset.value.cols}×${preset.value.rows} chars...`)
      const result = await renderImage(file, {
        width: preset.value.cols,
        height: preset.value.rows,
        filter: selectedFilter.value,
        color: true,
      })
      asciiContent.value = result
      addLog('render complete', 'ok')
      isProcessing.value = false
    } else {
      addLog('uploading video...')
      const id = await uploadVideo(file)
      addLog(`stream ready — ${preset.value.cols}×${preset.value.rows} chars`, 'ok')

      activeStream = streamVideo(
        id,
        { width: preset.value.cols, height: preset.value.rows, filter: selectedFilter.value, color: true },
        (frame) => {
          asciiContent.value = frame
        },
        () => {
          addLog('stream ended', 'ok')
          isProcessing.value = false
          activeStream = null
        },
      )
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'unknown error'
    addLog(msg, 'err')
    isProcessing.value = false
  }
}

function onFileInput(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) handleFile(file)
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  const file = e.dataTransfer?.files[0]
  if (file) handleFile(file)
}

// Re-render image on preset/filter change
watch([preset, selectedFilter], () => {
  if (selectedFile.value && !isVideo.value && !isProcessing.value) {
    handleFile(selectedFile.value)
  }
})
</script>

<template>
  <div
    class="shell"
    @drop.prevent="onDrop"
    @dragover.prevent
  >
    <!-- Top bar -->
    <div class="topbar">
      <span class="topbar-title">ASCII-RENDER — terminal</span>
      <span v-if="mediaInfo" class="topbar-info">[{{ mediaInfo }}]</span>
      <div class="topbar-spacer" />
      <span class="topbar-dims">{{ preset.cols }}×{{ preset.rows }} chars</span>
    </div>

    <!-- Body -->
    <div class="body">
      <!-- Sidebar -->
      <aside class="sidebar">
        <!-- FILE -->
        <div class="side-section">
          <div class="side-label">FILE</div>
          <div class="side-row">
            <input
              ref="fileInput"
              type="file"
              accept="image/*,video/*"
              style="display: none"
              @change="onFileInput"
            />
            <button class="term-btn full" @click="fileInput?.click()">[LOAD FILE]</button>
          </div>
          <div v-if="selectedFile" class="side-row file-info">
            <span>{{ selectedFile.name }}</span>
            <span class="muted">{{ (selectedFile.size / 1024).toFixed(1) }} KB</span>
          </div>
        </div>

        <!-- RESOLUTION -->
        <div class="side-section">
          <div class="side-label">RESOLUTION</div>
          <button
            v-for="p in PRESETS"
            :key="p.label"
            class="term-btn full"
            :class="{ active: preset.label === p.label }"
            @click="preset = p"
          >
            {{ p.label }}
          </button>
        </div>

        <!-- FILTER -->
        <div class="side-section">
          <div class="side-label">FILTER</div>
          <button
            v-for="f in FILTERS"
            :key="f.value"
            class="term-btn full"
            :class="{ active: selectedFilter === f.value }"
            @click="selectedFilter = f.value"
          >
            {{ f.label }}
          </button>
        </div>

        <!-- COLOR -->
        <div class="side-section">
          <div class="side-label">COLOR</div>
          <button
            class="term-btn full"
            :class="{ active: colorMode === 'green' }"
            @click="colorMode = 'green'"
          >● phosphor</button>
          <button
            class="term-btn full"
            :class="{ active: colorMode === 'amber' }"
            @click="colorMode = 'amber'"
          >● amber</button>
          <button
            class="term-btn full"
            :class="{ active: colorMode === 'white' }"
            @click="colorMode = 'white'"
          >● white</button>
        </div>

        <!-- PLAYBACK (solo video) -->
        <div v-if="isVideo" class="side-section">
          <div class="side-label">PLAYBACK</div>
          <button
            class="term-btn full active"
            @click="stopStream"
          >[■] stop</button>
        </div>

        <div class="sidebar-spacer" />
        <div class="drop-hint">drag &amp; drop<br />image or video</div>
      </aside>

      <!-- Main terminal area -->
      <div class="main">
        <!-- ASCII viewport -->
        <div class="viewport">
          <div v-if="isProcessing && !asciiContent" class="viewport-status glow-text">
            processing<span class="cursor" />
          </div>
          <div v-else-if="!asciiContent" class="viewport-empty">
            <div class="empty-icon">▒░▒</div>
            <div>no file loaded</div>
            <div class="empty-hint">use [LOAD FILE] or drop a file here</div>
          </div>
          <pre v-else :class="asciiClass">{{ asciiContent }}</pre>
        </div>

        <!-- Log console -->
        <div ref="logContainer" class="log-console">
          <div v-for="line in log" :key="line.id" class="log-line">
            <span class="log-prefix">{{ logPrefix(line.type) }}</span>
            <span :style="{ color: logColor(line.type) }">{{ line.text }}</span>
          </div>
          <div class="log-line">
            <span class="log-prefix">  &gt;  </span>
            <span class="cursor" style="color: var(--green)" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.shell {
  height: 100vh;
  width: 100vw;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  font-family: var(--font-mono);
  background: var(--bg);
}

/* Top bar */
.topbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
  flex-shrink: 0;
}

.topbar-title {
  color: var(--muted);
  font-size: 11px;
  letter-spacing: 0.12em;
}

.topbar-info {
  color: var(--green);
  font-size: 10px;
}

.topbar-spacer {
  flex: 1;
}

.topbar-dims {
  color: var(--muted);
  font-size: 10px;
}

/* Body layout */
.body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* Sidebar */
.sidebar {
  width: 192px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border);
  background: var(--surface);
  overflow-y: auto;
  font-size: 11px;
}

.side-section {
  border-bottom: 1px solid var(--border);
  padding-bottom: 6px;
}

.side-label {
  color: var(--muted);
  font-size: 9px;
  letter-spacing: 0.15em;
  text-transform: uppercase;
  padding: 8px 12px 4px;
}

.side-row {
  padding: 2px 8px;
}

.file-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 9px;
  color: var(--text);
  word-break: break-all;
}

.file-info .muted {
  color: var(--muted);
}

.term-btn.full {
  display: block;
  width: calc(100% - 16px);
  margin: 2px 8px;
  text-align: left;
}

.sidebar-spacer {
  flex: 1;
}

.drop-hint {
  padding: 12px;
  text-align: center;
  color: var(--muted);
  font-size: 9px;
  letter-spacing: 0.05em;
  line-height: 1.6;
}

/* Main */
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Viewport */
.viewport {
  flex: 1;
  overflow: auto;
  background: var(--bg);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.viewport-status {
  color: var(--green);
  font-size: 12px;
}

.viewport-empty {
  text-align: center;
  color: var(--muted);
  font-size: 11px;
  line-height: 2;
  letter-spacing: 0.06em;
}

.empty-icon {
  font-size: 40px;
  margin-bottom: 12px;
}

.empty-hint {
  font-size: 10px;
  margin-top: 4px;
}

/* Log console */
.log-console {
  flex-shrink: 0;
  height: 110px;
  overflow-y: auto;
  border-top: 1px solid var(--border);
  background: var(--surface);
  padding: 6px 12px;
  font-size: 10px;
  line-height: 1.7;
}

.log-line {
  display: flex;
}

.log-prefix {
  color: var(--muted);
  white-space: pre;
}
</style>
