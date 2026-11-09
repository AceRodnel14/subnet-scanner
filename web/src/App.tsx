import { useEffect, useState, type FormEvent } from 'react'

import { getInfo, scan, type Info } from './api'
import { Appearance } from './components/Appearance'
import { Logo } from './components/Logo'
import { Menu } from './components/Menu'
import { ErrorNote, Modal, Spinner, errMsg } from './components/ui'
import { clearLastScan, loadLastScan, saveLastScan, type LastScan } from './lastScan'

/** Use the ICO favicon when the server has a custom icon; otherwise the template SVG stays. */
function useFavicon(hasIcon: boolean | undefined) {
  useEffect(() => {
    if (!hasIcon) return
    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    if (!link) return
    link.type = 'image/x-icon'
    link.href = '/favicon.ico'
  }, [hasIcon])
}

function Scanner({ info }: { info: Info }) {
  const [last, setLast] = useState<LastScan | null>(loadLastScan)
  const [subnet, setSubnet] = useState(last?.subnet || info.default_subnet)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function run(e: FormEvent) {
    e.preventDefault()
    const target = subnet.trim()
    if (!target || busy) return
    setBusy(true)
    setError(undefined)
    try {
      const results = await scan(target)
      const next = { subnet: target, results, timestamp: new Date().toISOString() }
      saveLastScan(next)
      setLast(next)
    } catch (err) {
      setError(errMsg(err))
    } finally {
      setBusy(false)
    }
  }

  function clear() {
    clearLastScan()
    setLast(null)
  }

  const online = last ? last.results.filter((r) => r.alive).length : 0

  return (
    <section className="stack">
      <form className="card scan-form" onSubmit={run}>
        <label className="field">
          <span className="field-label">Subnet (CIDR notation)</span>
          <input
            className="input mono"
            value={subnet}
            onChange={(e) => setSubnet(e.target.value)}
            placeholder="192.168.1.0/24"
            spellCheck={false}
            autoComplete="off"
            disabled={busy}
          />
        </label>
        <button type="submit" className="btn primary" disabled={busy || subnet.trim() === ''}>
          {busy ? 'Scanning...' : 'Scan'}
        </button>
      </form>

      {busy && <Spinner label="Scanning subnet, pinging each address (this can take up to 20 seconds)" />}
      {error && <ErrorNote error={error} />}

      {last && (
        <div className="card">
          <div className="row between">
            <span className="muted">
              {last.subnet} &middot; scanned {new Date(last.timestamp).toLocaleString()}
            </span>
            <button type="button" className="btn small" onClick={clear} disabled={busy}>
              Clear results
            </button>
          </div>
          <p className="summary" role="status">
            <strong>{online}</strong> online &middot; <strong>{last.results.length - online}</strong> offline &middot;{' '}
            <strong>{last.results.length}</strong> total
          </p>
          <div className="legend">
            <span className="ip-cell online" aria-hidden="true">Online</span>
            <span className="ip-cell offline" aria-hidden="true">Offline</span>
          </div>
          <ul className="ip-grid" aria-label="Scan results">
            {last.results.map((r) => (
              <li key={r.ip} className={`ip-cell ${r.alive ? 'online' : 'offline'}`} title={`${r.ip}: ${r.alive ? 'online' : 'offline'}`}>
                <span aria-hidden="true">{r.alive ? '✓' : '✗'}</span>
                <span>.{r.ip.split('.').pop()}</span>
                <span className="sr-only">{r.ip} {r.alive ? 'online' : 'offline'}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}

export function App() {
  const [info, setInfo] = useState<Info>()
  const [error, setError] = useState<string>()
  const [appearance, setAppearance] = useState(false)

  useEffect(() => {
    getInfo().then(setInfo, (e: unknown) => setError(errMsg(e)))
  }, [])
  useFavicon(info?.has_icon)

  return (
    <>
      <header className="topbar">
        <span className="brand">
          <Logo height={28} /> Subnet Scanner
        </span>
        <Menu label="Menu" items={[{ label: 'Appearance', onClick: () => setAppearance(true) }]} />
      </header>
      <main className="page">
        {error && <ErrorNote error={error} />}
        {!info && !error && <Spinner />}
        {info && <Scanner info={info} />}
      </main>
      {appearance && (
        <Modal title="Appearance" onClose={() => setAppearance(false)}>
          <Appearance />
        </Modal>
      )}
    </>
  )
}
