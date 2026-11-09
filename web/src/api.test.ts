import { expect, test } from 'vitest'

import { getInfo, scan } from './api'

test('getInfo returns the API answer', async () => {
  const body = { base_url: 'http://localhost:8080', default_subnet: '192.168.1.0/24', has_icon: false }
  const fetcher = (async () => Response.json(body)) as typeof fetch
  await expect(getInfo(fetcher)).resolves.toEqual(body)
})

test('getInfo fails on an error status', async () => {
  const fetcher = (async () => new Response('', { status: 503 })) as typeof fetch
  await expect(getInfo(fetcher)).rejects.toThrow('503')
})

test('scan posts the subnet and returns the results', async () => {
  const body = [{ ip: '10.0.0.1', alive: true }, { ip: '10.0.0.2', alive: false }]
  let seen: { url: string; init?: RequestInit } | undefined
  const fetcher = (async (url: string, init?: RequestInit) => {
    seen = { url, init }
    return Response.json(body)
  }) as unknown as typeof fetch
  await expect(scan('10.0.0.0/24', fetcher)).resolves.toEqual(body)
  expect(seen?.url).toBe('/api/scan')
  expect(seen?.init?.method).toBe('POST')
  expect(JSON.parse(seen?.init?.body as string)).toEqual({ subnet: '10.0.0.0/24' })
})

test('scan surfaces the plain 400 message', async () => {
  const fetcher = (async () => new Response('limit: IPv4, /24 or narrower\n', { status: 400 })) as typeof fetch
  await expect(scan('10.0.0.0/8', fetcher)).rejects.toThrow('limit: IPv4, /24 or narrower')
})

test('scan surfaces the 503 hint', async () => {
  const fetcher = (async () => new Response('ICMP not permitted', { status: 503 })) as typeof fetch
  await expect(scan('10.0.0.0/24', fetcher)).rejects.toThrow('ICMP not permitted')
})

test('scan falls back to the status when the body is empty', async () => {
  const fetcher = (async () => new Response('', { status: 500 })) as typeof fetch
  await expect(scan('10.0.0.0/24', fetcher)).rejects.toThrow('500')
})
