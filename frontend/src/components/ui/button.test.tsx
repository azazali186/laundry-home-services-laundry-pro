import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { StatusBadge } from './badge'
import { Button } from './button'

describe('Button', () => {
  it('renders children and fires the click handler', async () => {
    const onClick = vi.fn()
    render(<Button onClick={onClick}>Save</Button>)

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it('is disabled while loading and never fires clicks', async () => {
    const onClick = vi.fn()
    render(
      <Button loading onClick={onClick}>
        Save
      </Button>,
    )

    const button = screen.getByRole('button', { name: 'Save' })
    expect(button).toBeDisabled()
    await userEvent.click(button)
    expect(onClick).not.toHaveBeenCalled()
  })
})

describe('StatusBadge', () => {
  it('humanizes the status value', () => {
    render(<StatusBadge status="is_active" />)
    expect(screen.getByText('Is Active')).toBeInTheDocument()
  })

  it('renders a dash when the status is missing', () => {
    render(<StatusBadge status={null} />)
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})
