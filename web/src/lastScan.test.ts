import { afterEach, expect, test, vi } from 'vitest'

import { clearLastScan, loadLastScan, saveLastScan, type LastScan } from './lastScan'

const sample: LastScan = {
  subnet: '10.0.0.0/24',
  results: [{ ip: '10.0.0.1', alive: true }],
  timestamp: '2026-01-02T03:04:05.000Z',
}

function fakeStorage(init: Record<string, string> = {}) {
  const data = { ...init }
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => data[k] ?? null,
    setItem: (k: string, v: string) => void (data[k] = v),
    removeItem: (k: string) => void delete data[k],
  })
  return data
}

afterEach(() => vi.unstubAllGlobals())

test('save then load round-trips, clear removes it', () => {
  fakeStorage()
  expect(loadLastScan()).toBeNull()
  saveLastScan(sample)
  expect(loadLastScan()).toEqual(sample)
  clearLastScan()
  expect(loadLastScan()).toBeNull()
})

test('corrupt or malformed values load as null', () => {
  for (const bad of ['{nope', '"x"', '{}', JSON.stringify({ ...sample, timestamp: 'never' }), JSON.stringify({ ...sample, results: [{ ip: 1 }] })]) {
    fakeStorage({ 'subnet-scanner.last-scan': bad })
    expect(loadLastScan()).toBeNull()
  }
})

test('throwing storage never throws', () => {
  const boom = () => {
    throw new Error('denied')
  }
  vi.stubGlobal('localStorage', { getItem: boom, setItem: boom, removeItem: boom })
  expect(loadLastScan()).toBeNull()
  expect(() => saveLastScan(sample)).not.toThrow()
  expect(() => clearLastScan()).not.toThrow()
})
