<script setup lang="ts">
import { ref } from 'vue'
import VideoPreview from '@/components/VideoPreview.vue'
import ProcessingControls from '@/components/ProcessingControls.vue'
import TerminalOutput from '@/components/TerminalOutput.vue'
import '@/styles/HomeView.css'
import type { ProcessingConfig } from '@/types/ProcessingConfig'
import { renderImage, uploadVideo, streamVideo } from '@/services/asciiService'

const selectedFile = ref<File | null>(null)
const previewUrl = ref<string | null>(null)
const asciiContent = ref('')
const outputStatus = ref('ESPERANDO RESULTADO')
const isProcessing = ref(false)
const errorMessage = ref('')
let activeStream: EventSource | null = null

const handleFileSelected = (file: File) => {
  selectedFile.value = file
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = URL.createObjectURL(file)
}

const handleProcess = async (config: ProcessingConfig) => {
  if (isProcessing.value) return
  if (!config.file) return

  stopStream()
  errorMessage.value = ''
  asciiContent.value = ''
  isProcessing.value = true
  outputStatus.value = 'PROCESANDO...'

  const [rawW, rawH] = config.resolution.split('x').map(Number)
  const width = rawW || 80
  const height = rawH || 40

  try {
    if (config.image === 'image') {
      const result = await renderImage(config.file, {
        width,
        height,
        filter: config.filter,
        color: true,
      })
      asciiContent.value = result
      outputStatus.value = 'LISTO'
    } else {
      outputStatus.value = 'SUBIENDO VIDEO...'
      const id = await uploadVideo(config.file)
      outputStatus.value = 'TRANSMITIENDO...'

      activeStream = streamVideo(
        id,
        { width, height, filter: config.filter, color: true },
        (frame) => {
          asciiContent.value = frame
          outputStatus.value = 'TRANSMITIENDO...'
        },
        () => {
          outputStatus.value = 'STREAM FINALIZADO'
          isProcessing.value = false
          activeStream = null
        },
      )
    }
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Error desconocido'
    outputStatus.value = 'ERROR'
  } finally {
    if (config.image === 'image') {
      isProcessing.value = false
    }
  }
}

const stopStream = () => {
  if (activeStream) {
    activeStream.close()
    activeStream = null
  }
}
</script>

<template>
  <div class="home">
    <header class="header">
      <div class="header-top">
        <div class="header-title">
          <span class="terminal-prefix">></span>
          <h1>ASCII Converter</h1>
        </div>
        <span class="status">● SISTEMA LISTO</span>
      </div>
      <p>Convierte imágenes y videos en caracteres ASCII</p>
    </header>

    <main class="main-content">
      <section class="preview-section">
        <h2>Previsualización</h2>
        <VideoPreview :file="selectedFile" :preview-url="previewUrl" />
      </section>

      <section class="controls-section">
        <h2>Configuración</h2>
        <ProcessingControls
          :file="selectedFile"
          :is-processing="isProcessing"
          @file-selected="handleFileSelected"
          @process="handleProcess"
          @stop="stopStream"
        />
        <p v-if="errorMessage" class="error-msg">{{ errorMessage }}</p>
      </section>
    </main>

    <section class="output-section">
      <div class="output-header">
        <h2>Resultado ASCII</h2>
        <span class="output-status">{{ outputStatus }}</span>
      </div>
      <div class="terminal-wrapper">
        <TerminalOutput :content="asciiContent" />
      </div>
    </section>
  </div>
</template>
