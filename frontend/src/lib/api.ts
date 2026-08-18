import axios, { AxiosError } from 'axios'
import type { Envelope, PageRequest, ListResult } from './types'
import { ApiError } from './types'

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

const TOKEN_KEY = 'laundrypro.token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

/** Global handler invoked on 401 so the app can clear the session. */
let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(fn: (() => void) | null) {
  onUnauthorized = fn
}

export const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30_000,
  headers: { 'Content-Type': 'application/json' },
})

// --- Request interceptor: attach bearer token --------------------------------
http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// --- Response interceptor: unwrap envelope, surface typed errors --------------
http.interceptors.response.use(
  (res) => {
    const env = res.data as Envelope
    // Non-envelope responses (shouldn't happen) pass through unchanged.
    if (env && typeof env === 'object' && 'code' in env && env.code !== 0) {
      throw new ApiError(res.status, env.code, env.message || 'Request failed', env.request_id)
    }
    return res
  },
  (error: AxiosError<Envelope>) => {
    if (error.response?.status === 401) {
      onUnauthorized?.()
    }
    const env = error.response?.data
    if (env && typeof env === 'object' && typeof env.code === 'number') {
      throw new ApiError(
        error.response?.status ?? 0,
        env.code,
        env.message || error.message,
        env.request_id,
      )
    }
    throw new ApiError(error.response?.status ?? 0, -1, error.message, undefined)
  },
)

/** POST a request body to a list endpoint and normalize the envelope. */
export async function listRequest<T, S = unknown>(
  path: string,
  req: PageRequest,
): Promise<ListResult<T, S>> {
  const res = await http.post<Envelope<T[], S>>(path, req)
  const env = res.data
  return {
    items: env.data ?? [],
    pagination: env.pagination ?? null,
    summary: env.summary ?? null,
  }
}

/** POST a single-entity request and return data (typed, may be null for 204-ish). */
export async function postRequest<TReq, TRes = unknown>(
  path: string,
  body: TReq,
): Promise<TRes> {
  const res = await http.post<Envelope<TRes>>(path, body)
  return res.data.data as TRes
}

/** PATCH a single-entity request (id always in body). */
export async function patchRequest<TReq, TRes = unknown>(
  path: string,
  body: TReq,
): Promise<TRes> {
  const res = await http.patch<Envelope<TRes>>(path, body)
  return res.data.data as TRes
}

/** DELETE a single-entity request (id in body). */
export async function deleteRequest<TReq>(path: string, body: TReq): Promise<void> {
  await http.delete<Envelope>(path, { data: body })
}

/** Extract a human-readable message from an unknown thrown value. */
export function errorMessage(err: unknown, fallback = 'Something went wrong'): string {
  if (err instanceof ApiError) return err.message
  if (err instanceof Error) return err.message
  if (typeof err === 'string') return err
  return fallback
}
