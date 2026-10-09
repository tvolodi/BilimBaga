import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18n from '@/i18n'
import { LogoUploader } from './LogoUploader'

const CURRENT_LOGO = '/api/v1/tenant/logo'
const DROP_LABEL = 'Drag and drop or click to upload'
const TYPE_ERROR = 'Unsupported file type. Use PNG, JPEG, SVG or WebP.'
const SIZE_ERROR = 'File is too large. Maximum size is 1 MB.'
const READ_ERROR = 'Could not read file. Please try again.'
const MAX_BYTES = 1024 * 1024

function makeFile(type: string, content = 'abc', name = 'logo.png'): File {
  return new File([content], name, { type })
}

function makeSizedFile(type: string, size: number): File {
  return new File([new Uint8Array(size)], 'big.png', { type })
}

function renderUploader(onFile = vi.fn()) {
  const { container } = render(<LogoUploader currentLogoUrl={CURRENT_LOGO} onFile={onFile} />)
  const dropzone = screen.getByRole('button', { name: DROP_LABEL })
  const input = container.querySelector('input[type="file"]') as HTMLInputElement
  const img = screen.getByRole('img', { name: 'Logo' }) as HTMLImageElement
  return { onFile, dropzone, input, img }
}

function selectFile(input: HTMLInputElement, file: File | undefined) {
  fireEvent.change(input, { target: { files: file ? [file] : [] } })
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('LogoUploader', () => {
  it('shows the current logo until a new file is chosen', () => {
    const { img, onFile } = renderUploader()

    expect(img).toHaveAttribute('src', CURRENT_LOGO)
    expect(onFile).not.toHaveBeenCalled()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('reads a valid PNG chosen through the file input and emits its data URL', async () => {
    const { onFile, input, img } = renderUploader()

    selectFile(input, makeFile('image/png', 'abc'))

    await waitFor(() => expect(onFile).toHaveBeenCalledWith('data:image/png;base64,YWJj'))
    expect(img).toHaveAttribute('src', 'data:image/png;base64,YWJj')
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it.each([
    ['image/png', 'data:image/png;base64,YWJj'],
    ['image/jpeg', 'data:image/jpeg;base64,YWJj'],
    ['image/svg+xml', 'data:image/svg+xml;base64,YWJj'],
    ['image/webp', 'data:image/webp;base64,YWJj'],
  ])('accepts %s uploads', async (type, expected) => {
    const { onFile, input } = renderUploader()

    selectFile(input, makeFile(type, 'abc'))

    await waitFor(() => expect(onFile).toHaveBeenCalledWith(expected))
  })

  it('rejects an unsupported file type without emitting a logo', () => {
    const { onFile, input, img } = renderUploader()

    selectFile(input, makeFile('text/plain', 'abc', 'notes.txt'))

    expect(screen.getByRole('alert')).toHaveTextContent(TYPE_ERROR)
    expect(onFile).not.toHaveBeenCalled()
    expect(img).toHaveAttribute('src', CURRENT_LOGO)
  })

  it('rejects a file larger than 1 MB', () => {
    const { onFile, input, img } = renderUploader()

    selectFile(input, makeSizedFile('image/png', MAX_BYTES + 1))

    expect(screen.getByRole('alert')).toHaveTextContent(SIZE_ERROR)
    expect(onFile).not.toHaveBeenCalled()
    expect(img).toHaveAttribute('src', CURRENT_LOGO)
  })

  it('accepts a file of exactly 1 MB', async () => {
    const { onFile, input } = renderUploader()

    selectFile(input, makeSizedFile('image/png', MAX_BYTES))

    await waitFor(() => expect(onFile).toHaveBeenCalledTimes(1))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('reports the type error first when a large file is also the wrong type', () => {
    const { onFile, input } = renderUploader()

    selectFile(input, makeSizedFile('application/pdf', MAX_BYTES + 1))

    expect(screen.getByRole('alert')).toHaveTextContent(TYPE_ERROR)
    expect(screen.queryByText(SIZE_ERROR)).not.toBeInTheDocument()
    expect(onFile).not.toHaveBeenCalled()
  })

  it('shows a read error when the browser cannot read the file', async () => {
    class FailingFileReader {
      onerror: (() => void) | null = null
      onload: (() => void) | null = null
      result: string | null = null
      readAsDataURL() {
        queueMicrotask(() => this.onerror?.())
      }
    }
    vi.stubGlobal('FileReader', FailingFileReader)
    const { onFile, input, img } = renderUploader()

    selectFile(input, makeFile('image/png', 'abc'))

    expect(await screen.findByRole('alert')).toHaveTextContent(READ_ERROR)
    expect(onFile).not.toHaveBeenCalled()
    expect(img).toHaveAttribute('src', CURRENT_LOGO)
  })

  it('clears a previous validation error once a valid file is read', async () => {
    const { onFile, input } = renderUploader()

    selectFile(input, makeFile('text/plain', 'abc', 'notes.txt'))
    expect(screen.getByRole('alert')).toHaveTextContent(TYPE_ERROR)

    selectFile(input, makeFile('image/jpeg', 'abc', 'logo.jpg'))

    await waitFor(() => expect(onFile).toHaveBeenCalledTimes(1))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('does nothing when the file dialog closes without a selection', () => {
    const { onFile, input } = renderUploader()

    selectFile(input, undefined)

    expect(onFile).not.toHaveBeenCalled()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('accepts a dropped valid file and emits its data URL', async () => {
    const { onFile, dropzone, img } = renderUploader()

    fireEvent.drop(dropzone, { dataTransfer: { files: [makeFile('image/png', 'abc')] } })

    await waitFor(() => expect(onFile).toHaveBeenCalledWith('data:image/png;base64,YWJj'))
    expect(img).toHaveAttribute('src', 'data:image/png;base64,YWJj')
  })

  it('shows the validation error for a dropped unsupported file', () => {
    const { onFile, dropzone } = renderUploader()

    fireEvent.drop(dropzone, { dataTransfer: { files: [makeFile('text/plain', 'abc', 'x.txt')] } })

    expect(screen.getByRole('alert')).toHaveTextContent(TYPE_ERROR)
    expect(onFile).not.toHaveBeenCalled()
  })

  it('ignores a drop that carries no files', () => {
    const { onFile, dropzone } = renderUploader()

    fireEvent.drop(dropzone, { dataTransfer: { files: [] } })

    expect(onFile).not.toHaveBeenCalled()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('highlights the drop zone while a file is dragged over it and clears on drag leave', () => {
    const { dropzone } = renderUploader()

    expect(dropzone).not.toHaveClass('border-primary')

    fireEvent.dragOver(dropzone)
    expect(dropzone).toHaveClass('border-primary')

    fireEvent.dragLeave(dropzone)
    expect(dropzone).not.toHaveClass('border-primary')
  })

  it('clears the drag highlight after a drop', () => {
    const { dropzone } = renderUploader()

    fireEvent.dragOver(dropzone)
    expect(dropzone).toHaveClass('border-primary')

    fireEvent.drop(dropzone, { dataTransfer: { files: [] } })
    expect(dropzone).not.toHaveClass('border-primary')
  })

  it('opens the file picker when the drop zone is clicked', () => {
    const { dropzone, input } = renderUploader()
    const clickSpy = vi.spyOn(input, 'click').mockImplementation(() => {})

    fireEvent.click(dropzone)

    expect(clickSpy).toHaveBeenCalledTimes(1)
  })

  it.each(['Enter', ' '])('opens the file picker when the drop zone receives the %j key', (key) => {
    const { dropzone, input } = renderUploader()
    const clickSpy = vi.spyOn(input, 'click').mockImplementation(() => {})

    fireEvent.keyDown(dropzone, { key })

    expect(clickSpy).toHaveBeenCalledTimes(1)
  })

  it('does not open the file picker for other keys', () => {
    const { dropzone, input } = renderUploader()
    const clickSpy = vi.spyOn(input, 'click').mockImplementation(() => {})

    fireEvent.keyDown(dropzone, { key: 'a' })

    expect(clickSpy).not.toHaveBeenCalled()
  })

  it('hides the logo image when it fails to load', () => {
    const { img } = renderUploader()

    fireEvent.error(img)

    expect(img.style.visibility).toBe('hidden')
  })
})
