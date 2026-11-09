/** App mark: white rounded square with a green "IP" and a red "?". Fixed colours, so it looks the same in both themes; keep in sync with public/favicon.svg. */
export function Logo({ height = 32 }: { height?: number }) {
  return (
    <svg className="logo" viewBox="0 0 24 24" height={height} width={height} aria-hidden="true" focusable="false">
      <rect width="24" height="24" rx="4.5" fill="#fff" />
      <path fill="#20885a" fillRule="evenodd" d="M3 6h2.8v12H3zM7.2 6h3.9a3.1 3.1 0 0 1 0 6.2H10V18H7.2zM10 8.4v1.5h1.1a.75.75 0 0 0 0-1.5z" />
      <path fill="#dd413a" d="M14.8 9.3a3.3 3.3 0 1 1 6.6 0c0 1.8-1.5 2.4-2.1 3.1v1.2h-2.4v-1.2c0-1.1.8-1.8 1.4-2.4.4-.4.7-.6.7-.7a.9.9 0 0 0-1.8 0zM18.1 15.4a1.3 1.3 0 1 0 0 2.6 1.3 1.3 0 0 0 0-2.6z" />
    </svg>
  )
}
