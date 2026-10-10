import { beforeEach, describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import i18n from '@/i18n'
import { BrandingPreview } from './BrandingPreview'

const DEFAULT_PRIMARY = '#2E6DB4'
const DEFAULT_ACCENT = '#C8A84B'

function renderPreview(config: Parameters<typeof BrandingPreview>[0]['config']) {
  render(<BrandingPreview config={config} />)
  const header = screen.getByTestId('branding-preview-header')
  const button = screen.getByTestId('branding-preview-accent-button')
  return { header, button }
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

describe('BrandingPreview', () => {
  it('renders the preview heading and body copy', () => {
    renderPreview({ app_name: 'Acme LMS' })

    expect(screen.getByRole('heading', { name: 'Live Preview' })).toBeInTheDocument()
    expect(screen.getByText('Preview of the main content area.')).toBeInTheDocument()
  })

  it('shows the application name and the sample navigation items in the header', () => {
    const { header } = renderPreview({ app_name: 'Acme LMS' })

    expect(within(header).getByText('Acme LMS', { selector: 'span.font-semibold' })).toBeInTheDocument()
    expect(within(header).getByText('Dashboard')).toBeInTheDocument()
    expect(within(header).getByText('Users')).toBeInTheDocument()
    expect(within(header).getByText('Exams')).toBeInTheDocument()
  })

  it('paints the header with the configured primary colour', () => {
    const { header } = renderPreview({ app_name: 'Acme LMS', primary_color: '#123456' })

    expect(header).toHaveStyle({ backgroundColor: '#123456' })
  })

  it('falls back to the default primary colour when none is configured', () => {
    const { header } = renderPreview({ app_name: 'Acme LMS' })

    expect(header).toHaveStyle({ backgroundColor: DEFAULT_PRIMARY })
  })

  it('paints the sample button with the configured accent colour', () => {
    const { button } = renderPreview({ app_name: 'Acme LMS', accent_color: '#ff0000' })

    expect(button).toHaveStyle({ backgroundColor: '#ff0000' })
    expect(button).toHaveTextContent('Sample Button')
  })

  it('falls back to the default accent colour when none is configured', () => {
    const { button } = renderPreview({ app_name: 'Acme LMS' })

    expect(button).toHaveStyle({ backgroundColor: DEFAULT_ACCENT })
  })

  it('shows the uploaded logo override in the header', () => {
    renderPreview({ app_name: 'Acme LMS', logo: 'data:image/png;base64,AAAA' })

    expect(screen.getByRole('img', { name: 'Acme LMS' })).toHaveAttribute(
      'src',
      'data:image/png;base64,AAAA',
    )
  })

  it('falls back to the tenant logo endpoint when no logo override is set', () => {
    renderPreview({ app_name: 'Acme LMS' })

    expect(screen.getByRole('img', { name: 'Acme LMS' })).toHaveAttribute(
      'src',
      '/api/v1/tenant/logo',
    )
  })

  it('renders with an empty config using the default colours and the logo placeholder', () => {
    const { header, button } = renderPreview({})

    expect(header).toHaveStyle({ backgroundColor: DEFAULT_PRIMARY })
    expect(button).toHaveStyle({ backgroundColor: DEFAULT_ACCENT })
    expect(screen.getByRole('img', { name: 'Logo' })).toBeInTheDocument()
  })
})
