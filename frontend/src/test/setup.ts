import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Vitest globals are disabled, so Testing Library's automatic cleanup hook
// never registers itself — wire it explicitly to keep renders isolated.
afterEach(() => {
  cleanup()
})

// jsdom's Storage can surface as an empty plain object under Vitest 3 (URL
// applied, Storage prototype methods missing). Install a deterministic
// in-memory shim so token/cache code behaves exactly like a browser.
const store = new Map<string, string>()
const localStorageShim: Storage = {
  get length() {
    return store.size
  },
  clear: () => store.clear(),
  getItem: (key) => store.get(key) ?? null,
  key: (index) => [...store.keys()][index] ?? null,
  removeItem: (key) => {
    store.delete(key)
  },
  setItem: (key, value) => {
    store.set(key, String(value))
  },
}
Object.defineProperty(globalThis, 'localStorage', { value: localStorageShim, configurable: true })
Object.defineProperty(window, 'localStorage', { value: localStorageShim, configurable: true })

afterEach(() => {
  store.clear()
})
