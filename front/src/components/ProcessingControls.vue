<script setup lang="ts">
import '@/styles/ProcessingControls.css'
import { ref, watch } from 'vue'
import Button from 'primevue/button'
import Select from 'primevue/select'
import FileUpload from 'primevue/fileupload'
import type { FileUploadSelectEvent } from 'primevue/fileupload'
interface ProcessingConfig {
  file: File
  image: 'image' | 'video'
  resolution: string
  filter: string
}

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
}>()

const image = ref<'image' | 'video'>('image')
const resolution = ref('80x40')
const filter = ref('none')
const errorMessage = ref('')

const isProcessing = ref(false)

watch(image, (newValue) => {
  console.log('Tipo de entrada:', newValue)
})

watch(resolution, (newValue) => {
  console.log('Resolucion:', newValue)
})

watch(filter, (newValue) => {
  console.log('Filtro:', newValue)
})

const handleFileChange = (event: FileUploadSelectEvent) => {
  const file = event.files[0]
  if (file) {
    emit('fileSelected', file)
    console.log('Archivo seleccionado', file)
    console.log('Nombre:', file.name)
    console.log('Tipo:', file.type)
    console.log('Tamaño', file.size)
  }
}

const formatFileSize = (size: number) => {
  if (size < 1024) {
    return `${size} B`
  }

  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(2)} KB`
  }

  if (size < 1024 * 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(2)} MB`
  }

  return `${(size / (1024 * 1024 * 1024)).toFixed(2)} GB`
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
  console.log('Configuracion lista para procesar:', processingConfig)

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
