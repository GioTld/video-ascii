<script setup lang="ts">
defineProps<{
  file: File | null
  previewUrl: string | null
}>()

const formatFileSize = (size: number) => {
  const units = ['B', 'KB', 'MB', 'GB']
  let unit = 0

  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }

  return `${size.toFixed(2)} ${units[unit]}`
}
</script>

<template>
  <section class="video-preview">
    <div class="preview-window">
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

      <div class="preview-info">
        <span v-if="file"> FILE: {{ file.name }} </span>

        <span v-else> FILE: NO SELECCIONADO </span>

        <span v-if="file">
          {{ formatFileSize(file.size) }}
        </span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.video-preview {
  width: 100%;
}

.preview-window {
  overflow: hidden;
  border: 1px solid #29352d;
  border-radius: 10px;
  background: #0b100c;
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.3);
}

.preview-toolbar {
  height: 38px;
  display: flex;
  align-items: center;
  padding: 0 14px;
  border-bottom: 1px solid #253129;
  background: #0d120f;
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
  background: #4f6255;
}

.preview-title {
  margin-left: 14px;
  color: #63806b;
  font-family: monospace;
  font-size: 11px;
  letter-spacing: 1px;
}

.preview-status {
  margin-left: auto;
  color: #63a878;
  font-family: monospace;
  font-size: 10px;
  letter-spacing: 0.8px;
}

.preview-screen {
  position: relative;
  height: 400px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: #050805;
  box-shadow: inset 0 0 40px rgba(80, 170, 100, 0.04);
}

.preview-screen::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(rgba(80, 170, 100, 0.018) 1px, transparent 1px),
    linear-gradient(90deg, rgba(80, 170, 100, 0.018) 1px, transparent 1px);
  background-size: 24px 24px;
}

.preview-screen img,
.preview-screen video {
  position: relative;
  z-index: 1;
  display: block;
  max-width: 100%;
  max-height: 364px;
  object-fit: contain;
}

.preview-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 2;
  height: 36px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border-bottom: 1px solid #253129;
  background: rgba(11, 16, 12, 0.94);
  color: #63806b;
  font-family: monospace;
  font-size: 11px;
  letter-spacing: 1px;
}

.preview-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #63a878;
  box-shadow: 0 0 8px rgba(80, 170, 100, 0.35);
}

.preview-screen p {
  position: relative;
  z-index: 1;
  margin: 0;
  color: #4f6255;
  font-family: monospace;
  font-size: 13px;
  letter-spacing: 0.5px;
  text-align: center;
}

.preview-info {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 14px;
  border-top: 1px solid #253129;
  background: #0b100c;
  color: #63806b;
  font-family: monospace;
  font-size: 10px;
  letter-spacing: 0.6px;
}
</style>
