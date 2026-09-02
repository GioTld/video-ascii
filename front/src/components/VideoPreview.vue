<script setup lang="ts">
defineProps<{
  file: File | null
  previewUrl: string | null
}>()
</script>

<template>
  <section class="video-preview">
    <div class="preview-window">
      <div class="preview-toolbar">
        <div class="windows-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>
    </div>
    <div class="preview-screen">
      <div class="preview-header">
        <span class="preview-dot"></span>
        <span>PREVIEW</span>
      </div>
      <template v-if="file && previewUrl">
        <img v-if="file.type.startsWith('image/')" :src="previewUrl" :alt="file.name" />

        <video v-else-if="file.type.startsWith('video/')" :src="previewUrl" controls></video>
      </template>

      <p v-else>No hay ninguna imagen o video seleccionado</p>
    </div>
  </section>
</template>

<style scoped>
.video-preview {
  width: 100%;
}

.preview-window {
  overflow: hidden;
  border: 1px solid #2d2d2d;
  border-radius: 10px;
  background: #0b0b0b;
}

.preview-toolbar {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid #2d2d2d;
  background: #151515;
}

.windows-dots {
  display: flex;
  gap: 6px;
}

.windows-dots span {
  width: 7px;
  height: 7px;
  display: block;
  border-radius: 50%;
  background: #555;
}

.preview-label {
  color: #666;
  font-family: monospace;
  font-size: 11px;
  letter-spacing: 1px;
}

.preview-screen {
  min-height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #080808;
  border: 1px solid #333;
  border-radius: 10px;
  overflow: hidden;
  position: relative;
  box-shadow: inset 0 0 40px rgba(0, 0, 0, 0.35);
}

.preview-screen img,
.preview-screen video {
  display: block;
  max-width: 100%;
  max-height: 400px;
  object-fit: contain;
}

.preview-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #555;
}

.preview-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 36px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border-bottom: 1px solid #2d2d2d;
  background: #111;
  color: #777;
  font-family: monospace;
  font-size: 12px;
  letter-spacing: 1px;
}

.preview-screen p {
  margin: 0;
  color: #666;
  font-family: monospace;
  font-size: 14px;
}
</style>
