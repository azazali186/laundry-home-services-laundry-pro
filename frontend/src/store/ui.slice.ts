import { createSlice, type PayloadAction } from '@reduxjs/toolkit'

export interface UIState {
  sidebarCollapsed: boolean
  mobileSidebarOpen: boolean
}

const initialState: UIState = {
  sidebarCollapsed: false,
  mobileSidebarOpen: false,
}

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    sidebarToggled(state) {
      state.sidebarCollapsed = !state.sidebarCollapsed
    },
    mobileSidebarOpened(state) {
      state.mobileSidebarOpen = true
    },
    mobileSidebarClosed(state) {
      state.mobileSidebarOpen = false
    },
    mobileSidebarToggled(state, action: PayloadAction<boolean | undefined>) {
      state.mobileSidebarOpen = action.payload ?? !state.mobileSidebarOpen
    },
  },
})

export const {
  sidebarToggled,
  mobileSidebarOpened,
  mobileSidebarClosed,
  mobileSidebarToggled,
} = uiSlice.actions
export default uiSlice.reducer
