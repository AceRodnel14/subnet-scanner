import type { ScanResult } from './api'

export type LastScan = { subnet: string; results: ScanResult[]; timestamp: string }

const KEY = 'subnet-scanner.last-scan'

function valid(v: unknown): v is LastScan {
  if (typeof v !== 'object' || v === null) return false
  const s = v as Partial<LastScan>
  return (
    typeof s.subnet === 'string' &&
    typeof s.timestamp === 'string' &&
    !Number.isNaN(Date.parse(s.timestamp)) &&
    Array.isArray(s.results) &&
    s.results.every((r) => typeof r === 'object' && r !== null && typeof r.ip === 'string' && typeof r.alive === 'boolean')
  )
}

/** The saved scan, or null when missing, corrupt or storage is unavailable. */
export function loadLastScan(): LastScan | null {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return null
    const v: unknown = JSON.parse(raw)
    return valid(v) ? v : null
  } catch {
    return null
  }
}

export function saveLastScan(s: LastScan): void {
  try {
    localStorage.setItem(KEY, JSON.stringify(s))
  } catch {
    /* storage unavailable: the scan just is not remembered */
  }
}

export function clearLastScan(): void {
  try {
    localStorage.removeItem(KEY)
  } catch {
    /* ignore */
  }
}
