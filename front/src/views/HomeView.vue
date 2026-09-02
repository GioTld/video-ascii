<script setup lang="ts">
import { ref } from 'vue'
import VideoPreview from '@/components/VideoPreview.vue'
import ProcessingControls from '@/components/ProcessingControls.vue'
import '@/styles/HomeView.css'

const selectedFile = ref<File | null>(null)
const previewUrl = ref<string | null>(null)

const handleFileSelected = (file: File) => {
  selectedFile.value = file

  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
  }
  previewUrl.value = URL.createObjectURL(file)

  console.log('Archivo recibido en HomeView', file)
}
</script>

<template>
  <div class="home">
    <header class="header">
      <div class="header-top">
        <div>
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

        <ProcessingControls :file="selectedFile" @file-selected="handleFileSelected" />
      </section>
    </main>

    <section class="output-section">
      <div class="output-header">
        <h2>Resultado ASCII</h2>
        <span class="output-status">ESPERANDO RESULTADO</span>
      </div>

      <div class="ascii-output">
        <p>Aquí aparecerá el resultado ASCII</p>
      </div>
    </section>
  </div>
</template>
