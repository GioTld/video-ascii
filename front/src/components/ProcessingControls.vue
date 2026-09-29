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
  { label: 'Escala de grises', value: 'greyscale' },
]
const props = defineProps<{
  file: File | null
}>()

const emit = defineEmits<{
  fileSelected: [file: File]
  process: [config: ProcessingConfig]
}>()

const image = ref<'image' | 'video'>('image')
const resolution = ref('80x40')
const filter = ref('none')
const errorMessage = ref('')

const isProcessing = ref(false)

const handleFileChange = (event: FileUploadSelectEvent) => {
  const file = event.files[0]
  if (file) {
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
  if (isProcessing.value) {
    return
  }
  errorMessage.value = ''
  if (!props.file) {
    errorMessage.value = 'No se ha seleccionado ningun archivo'
    return
  }

  const isImage = props.file.type.startsWith('image/')
  const isVideo = props.file.type.startsWith('video/')

  if (image.value === 'image' && !isImage) {
    errorMessage.value = 'El archivo seleccionado no es una imagen'
    return
  }

  if (image.value === 'video' && !isVideo) {
    errorMessage.value = 'El archivo seleccionado no es un video'
    return
  }

  isProcessing.value = true

  const processingConfig: ProcessingConfig = {
    file: props.file,
    image: image.value,
    resolution: resolution.value,
    filter: filter.value,
  }
  emit('process', processingConfig)

  setTimeout(() => {
    isProcessing.value = false
  }, 2000)
}
</script>

<template>
  <section class="stream-controls">
    <div>
      <label for="file">Archivo</label>
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
        <small>
          Tipo : {{ props.file.type }} | Tamaño: {{ formatFileSize(props.file.size) }}
        </small>
      </div>
    </div>

    <div>
      <label for="image">Tipo de entrada</label>

      <Select
        v-model="image"
        :options="inputTypes"
        option-label="label"
        option-value="value"
        input-id="image"
        placeholder="Seleccionar Tipo"
      ></Select>
    </div>

    <div>
      <label for="resolution">Resolución</label>

      <Select
        v-model="resolution"
        :options="resolutions"
        option-label="label"
        option-value="value"
        input-id="resolution"
        placeholder="Seleccionar resolucion"
      ></Select>
    </div>

    <div>
      <label for="filter">Filtro</label>

      <Select
        v-model="filter"
        :options="filters"
        optionLabel="label"
        optionValue="value"
        input-id="filter"
        placeholder="Seleccionar filtro"
      />
    </div>

    <Button
      :label="isProcessing ? 'Procesando...' : 'Convertir'"
      :loading="isProcessing"
      :disabled="isProcessing"
      @click="startProcessing"
    />
    <p v-if="errorMessage" class="error-message">
      {{ errorMessage }}
    </p>
  </section>
</template>
