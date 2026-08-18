import { afterEach, describe, expect, it } from 'vitest'
import { useState } from 'react'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { Modal } from './modal'

afterEach(() => cleanup())

function Harness() {
  const [open, setOpen] = useState(false)
  return (
    <div>
      <button type="button" onClick={() => setOpen(true)}>
        Open modal
      </button>
      <Modal open={open} onClose={() => setOpen(false)} title="Test dialog">
        <input aria-label="Name" />
        <button type="button">Cancel</button>
      </Modal>
    </div>
  )
}

describe('Modal', () => {
  it('moves focus into the dialog when it opens', () => {
    render(<Harness />)
    fireEvent.click(screen.getByRole('button', { name: 'Open modal' }))
    expect(screen.getByRole('dialog')).toHaveFocus()
  })

  it('traps Tab and Shift+Tab within the dialog', () => {
    render(<Harness />)
    fireEvent.click(screen.getByRole('button', { name: 'Open modal' }))
    const dialog = screen.getByRole('dialog')
    const close = screen.getByRole('button', { name: 'Close dialog' })
    const cancel = screen.getByRole('button', { name: 'Cancel' })

    // Shift+Tab from the first focusable wraps to the last.
    close.focus()
    fireEvent.keyDown(dialog, { key: 'Tab', shiftKey: true })
    expect(cancel).toHaveFocus()

    // Tab from the last focusable wraps back to the first.
    fireEvent.keyDown(dialog, { key: 'Tab' })
    expect(close).toHaveFocus()
  })

  it('closes on Escape and restores focus to the opener', () => {
    render(<Harness />)
    const opener = screen.getByRole('button', { name: 'Open modal' })
    // jsdom's fireEvent.click doesn't move focus like a real browser click;
    // focus the opener explicitly so the restore target is exercised.
    opener.focus()
    fireEvent.click(opener)
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(opener).toHaveFocus()
  })
})
