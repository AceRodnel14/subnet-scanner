// Typed client for the Go API. Same origin in production (embedded UI), proxied in dev.
export type Info = { base_url: string; default_subnet: string; has_icon: boolean }
export type ScanResult = { ip: string; alive: boolean }

/** Error text from a failed response: the server sends a plain message (400) or a hint (503). */
async function failure(what: string, res: Response): Promise<Error> {
  let text = ''
  try {
    text = (await res.text()).trim()
  } catch {
    /* unreadable body: fall back to the status */
  }
  return new Error(text || `${what}: ${res.status}`)
}

export async function getInfo(fetcher: typeof fetch = fetch): Promise<Info> {
  const res = await fetcher('/api/info')
  if (!res.ok) throw await failure('GET /api/info', res)
  return (await res.json()) as Info
}

export async function scan(subnet: string, fetcher: typeof fetch = fetch): Promise<ScanResult[]> {
  const res = await fetcher('/api/scan', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ subnet }),
  })
  if (!res.ok) throw await failure('POST /api/scan', res)
  return (await res.json()) as ScanResult[]
}
