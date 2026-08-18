import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { setToken } from '@/lib/api'
import type { UserView } from '@/services/auth.service'

export interface AuthState {
  token: string | null
  user: UserView | null
  /** Whether a /auth/me refresh has completed in this session. */
  hydrated: boolean
}

const TOKEN_KEY = 'laundrypro.token'
const USER_KEY = 'laundrypro.user'

function loadInitial(): AuthState {
  const token = localStorage.getItem(TOKEN_KEY)
  let user: UserView | null = null
  try {
    const raw = localStorage.getItem(USER_KEY)
    if (raw) user = JSON.parse(raw) as UserView
  } catch {
    user = null
  }
  return { token, user, hydrated: !!token }
}

const initialState: AuthState = loadInitial()

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    loggedIn(state, action: PayloadAction<{ token: string; user: UserView }>) {
      state.token = action.payload.token
      state.user = action.payload.user
      state.hydrated = true
      setToken(action.payload.token)
      localStorage.setItem(USER_KEY, JSON.stringify(action.payload.user))
    },
    userRefreshed(state, action: PayloadAction<UserView>) {
      state.user = action.payload
      state.hydrated = true
      localStorage.setItem(USER_KEY, JSON.stringify(action.payload))
    },
    loggedOut(state) {
      state.token = null
      state.user = null
      state.hydrated = false
      setToken(null)
      localStorage.removeItem(USER_KEY)
    },
  },
})

export const { loggedIn, userRefreshed, loggedOut } = authSlice.actions
export default authSlice.reducer
