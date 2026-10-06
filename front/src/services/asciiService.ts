const BASE_URL = '/api'

export interface ProcessingConfig {
  width: number
  height: number
  filter?: string
  color?: boolean
  colorMode?: '24bit' | '256' | 'none'
  ramp?: string
  hdMode?: 'none' | 'half' | 'braille'
  fitMode?: 'contain' | 'cover'
}

export interface UploadVideoResult {
  id: string
}

function renderParams(config: ProcessingConfig): URLSearchParams {
  const params = new URLSearchParams({
    width: String(config.width),
    height: String(config.height),
  })
  if (config.filter && config.filter !== 'none') {
    params.set('filter', config.filter)
  }
  if (config.colorMode && config.colorMode !== 'none') {
    params.set('color', 'true')
    params.set('color_mode', config.colorMode)
  } else if (config.color && config.colorMode !== 'none') {
    params.set('color', 'true')
  }
  if (config.ramp) params.set('ramp', config.ramp)
  if (config.hdMode && config.hdMode !== 'none') params.set('hd_mode', config.hdMode)
  if (config.fitMode) params.set('fit_mode', config.fitMode)
  return params
}

export async function renderImage(file: File, config: ProcessingConfig): Promise<string> {
  const params = renderParams(config)

  const form = new FormData()
  form.append('file', file)

  const res = await fetch(`${BASE_URL}/render/image?${params}`, {
    method: 'POST',
    body: form,
  })

  if (!res.ok) {
    const msg = await res.text()
    throw new Error(`render failed: ${msg}`)
  }

  return res.text()
}

export async function uploadVideo(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)

  const res = await fetch(`${BASE_URL}/upload/video`, {
    method: 'POST',
    body: form,
  })

  if (!res.ok) {
    const msg = await res.text()
    throw new Error(`upload failed: ${msg}`)
  }

  const data: UploadVideoResult = await res.json()
  return data.id
}

export function streamVideo(
  id: string,
  config: ProcessingConfig,
  onFrame: (ansiText: string) => void,
  onError: (err: Event) => void,
): EventSource {
  const params = renderParams(config)
  params.set('id', id)

  const source = new EventSource(`${BASE_URL}/render/video?${params}`)

  source.addEventListener('frame', (e: MessageEvent) => {
    onFrame(e.data)
  })

  source.onerror = onError

  return source
}
