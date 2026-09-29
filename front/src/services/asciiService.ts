const BASE_URL = '/api'

export interface RenderImageConfig {
  width: number
  height: number
  filter?: string
  color?: boolean
}

export interface UploadVideoResult {
  id: string
}

export async function renderImage(file: File, config: RenderImageConfig): Promise<string> {
  const params = new URLSearchParams({
    width: String(config.width),
    height: String(config.height),
  })
  if (config.filter && config.filter !== 'none') {
    params.set('filter', config.filter)
  }
  if (config.color) {
    params.set('color', 'true')
  }

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
  config: RenderImageConfig,
  onFrame: (ansiText: string) => void,
  onError: (err: Event) => void,
): EventSource {
  const params = new URLSearchParams({
    id,
    width: String(config.width),
    height: String(config.height),
    color: config.color ? 'true' : 'false',
  })
  if (config.filter && config.filter !== 'none') {
    params.set('filter', config.filter)
  }

  const source = new EventSource(`${BASE_URL}/render/video?${params}`)

  source.addEventListener('frame', (e: MessageEvent) => {
    onFrame(e.data)
  })

  source.onerror = onError

  return source
}
