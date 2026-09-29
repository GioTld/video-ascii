<script setup lang="ts">
import '@/styles/ProcessingControls.css'
import { ref } from 'vue'
import Button from 'primevue/button'
import Select from 'primevue/select'
import FileUpload from 'primevue/fileupload'
import type { FileUploadSelectEvent } from 'primevue/fileupload'
import type { ProcessingConfig } from '@/types/ProcessingConfig'

const resolutions = [
  { label: '40 x 20', value: '40x20' },
  { label: '80 x 40', value: '80x40' },
  { label: '120 x 60', value: '120x60' },
]

const inputTypes = [
  { label: 'Imagen', value: 'image' },
  { label: 'Video', value: 'video' },
]

const filters = [
  { label: 'Ninguno', value: 'none' },
  { label: 'Escala de grises', value: 'grayscale' },
  { label: 'Sepia', value: 'sepia' },
  { label: 'Invertir', value: 'invert' },
]

const props = defineProps<{
  file: File | null
  isProcessing?: boolean
}>()

const emit = defineEmits<{
  fileSelected: [file: File]
  process: [config: ProcessingConfig]
  stop: []
}>()

const inputType = ref<'image' | 'video'>('image')
const resolution = ref('80x40')
const filter = ref('none')
const errorMessage = ref('')

const handleFileChange = (event: FileUploadSelectEvent) => {
  const file = event.files[0]
  if (file) {
    errorMessage.value = ''
    emit('fileSelected', file)
  }
}

const formatFileSize = (size: number) => {
  const units = ['B', 'KB', 'MB', 'GB']
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size.toFixed(2)} ${units[unit]}`
}

const startProcessing = () => {
  if (props.isProcessing) return
  errorMessage.value = ''

  if (!props.file) {
    errorMessage.value = 'No se ha seleccionado ningún archivo'
    return
  }

  const isImage = props.file.type.startsWith('image/')
  const isVideo = props.file.type.startsWith('video/')

  if (inputType.value === 'image' && !isImage) {
    errorMessage.value = 'El archivo seleccionado no es una imagen'
    return
  }
  if (inputType.value === 'video' && !isVideo) {
    errorMessage.value = 'El archivo seleccionado no es un video'
    return
  }

  emit('process', {
    file: props.file,
    image: inputType.value,
    resolution: resolution.value,
    filter: filter.value,
  })
}
</script>

<template>
  <section class="stream-controls">
    <div>
      <label>Archivo</label>
      <FileUpload
        mode="basic"
        choose-label="Seleccionar archivo"
        accept="image/*,video/*"
        @select="handleFileChange"
      />
      <div v-if="props.file" class="selected-file">
        <div class="file-info">
          <p>Archivo seleccionado</p>
          <p>Nombre: {{ props.file.name }}</p>
        </div>
        <small>Tipo: {{ props.file.type }} | Tamaño: {{ formatFileSize(props.file.size) }}</small>
      </div>
    </div>

    <div>
      <label>Tipo de entrada</label>
      <Select
        v-model="inputType"
        :options="inputTypes"
        option-label="label"
        option-value="value"
        placeholder="Seleccionar Tipo"
      />
    </div>

    <div>
      <label>Resolución</label>
      <Select
        v-model="resolution"
        :options="resolutions"
        option-label="label"
        option-value="value"
        placeholder="Seleccionar resolución"
      />
    </div>

    <div>
      <label>Filtro</label>
      <Select
        v-model="filter"
        :options="filters"
        option-label="label"
        option-value="value"
        placeholder="Seleccionar filtro"
      />
    </div>

    <div class="action-buttons">
      <Button
        :label="props.isProcessing ? 'Procesando...' : 'Convertir'"
        :loading="props.isProcessing"
        :disabled="props.isProcessing"
        @click="startProcessing"
      />
      <Button
        v-if="props.isProcessing"
        label="Detener"
        severity="secondary"
        @click="emit('stop')"
      />
    </div>

    <p v-if="errorMessage" class="error-message">{{ errorMessage }}</p>
  </section>
</template>
