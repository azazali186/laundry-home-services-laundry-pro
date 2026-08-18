import MockAdapter from 'axios-mock-adapter'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  errorMessage,
  getToken,
  http,
  listRequest,
  postRequest,
  setToken,
  setUnauthorizedHandler,
} from './api'
import { ApiError } from './types'

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(http)
  localStorage.clear()
})

afterEach(() => {
  mock.restore()
  vi.restoreAllMocks()
  setUnauthorizedHandler(null)
  setToken(null)
})

describe('listRequest', () => {
  it('unwraps the envelope into items, pagination and summary', async () => {
    mock.onPost('/users/list').reply(200, {
      code: 0,
      message: 'ok',
      data: [{ id: 'u1' }],
      pagination: { limit: 10, total: 1, has_more: false, offset: 0 },
      summary: { total_active: 1 },
    })

    const res = await listRequest<{ id: string }, { total_active: number }>('/users/list', {
      limit: 10,
    })
    expect(res.items).toHaveLength(1)
    expect(res.pagination?.total).toBe(1)
    expect(res.summary?.total_active).toBe(1)
  })

  it('defaults to an empty list when the envelope has no data', async () => {
    mock.onPost('/x/list').reply(200, { code: 0, message: 'ok' })
    const res = await listRequest('/x/list', {})
    expect(res.items).toEqual([])
    expect(res.pagination).toBeNull()
  })
})

describe('postRequest', () => {
  it('returns the envelope data', async () => {
    mock.onPost('/auth/login').reply(200, { code: 0, message: 'ok', data: { access_token: 't' } })
    const data = await postRequest<{ email: string }, { access_token: string }>('/auth/login', {
      email: 'a@b.c',
    })
    expect(data.access_token).toBe('t')
  })

  it('throws an ApiError when the envelope carries a business error code', async () => {
    mock.onPost('/users').reply(409, { code: 409, message: 'duplicate' })
    await expect(postRequest('/users', {})).rejects.toBeInstanceOf(ApiError)
  })
})

describe('error handling and 401', () => {
  it('surfaces typed ApiError messages', async () => {
    mock.onGet('/boom').reply(400, { code: 400, message: 'invalid payload' })
    await expect(http.get('/boom')).rejects.toMatchObject({ message: 'invalid payload' })
  })

  it('invokes the global handler on 401 responses', async () => {
    const handler = vi.fn()
    setUnauthorizedHandler(handler)
    mock.onGet('/expired').reply(401, { code: 401, message: 'token expired' })
    await expect(http.get('/expired')).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalledTimes(1)
  })

  it('attaches the bearer token to requests', async () => {
    setToken('jwt-123')
    let authHeader: string | undefined
    mock.onGet('/me').reply((config) => {
      authHeader = config.headers?.Authorization as string | undefined
      return [200, { code: 0, message: 'ok', data: {} }]
    })
    await http.get('/me')
    expect(authHeader).toBe('Bearer jwt-123')
    expect(getToken()).toBe('jwt-123')
  })
})

describe('errorMessage', () => {
  it('prefers the ApiError message', () => {
    expect(errorMessage(new ApiError(400, 400, 'bad request', 'rid'))).toBe('bad request')
  })

  it('falls back for unknown values', () => {
    expect(errorMessage(undefined, 'fallback')).toBe('fallback')
    expect(errorMessage('plain string')).toBe('plain string')
  })
})
