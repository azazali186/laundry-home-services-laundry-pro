import { configureStore } from '@reduxjs/toolkit'
import { useDispatch, useSelector } from 'react-redux'
import type { TypedUseSelectorHook } from 'react-redux'
import authReducer from './auth.slice'
import uiReducer from './ui.slice'

export { loggedIn, loggedOut, userRefreshed } from './auth.slice'
export { sidebarToggled, mobileSidebarOpened, mobileSidebarClosed, mobileSidebarToggled } from './ui.slice'

export const store = configureStore({
  reducer: {
    auth: authReducer,
    ui: uiReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch

export const useAppDispatch: () => AppDispatch = useDispatch
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector
