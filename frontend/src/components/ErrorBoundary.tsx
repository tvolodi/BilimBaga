import { Component, type ErrorInfo, type ReactNode } from 'react'
import { Button } from '@/components/ui/button'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  errorMessage: string
}

/**
 * Top-level error boundary that catches any uncaught render error and shows a
 * recoverable error screen instead of a blank page.
 *
 * ISS-027: without this boundary, React 18 in production mode silently unmounts
 * the entire tree on a render throw, leaving a blank white page.
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, errorMessage: '' }
  }

  static getDerivedStateFromError(error: unknown): State {
    const message =
      error instanceof Error ? error.message : String(error ?? 'Unknown error')
    return { hasError: true, errorMessage: message }
  }

  componentDidCatch(error: unknown, info: ErrorInfo) {
    // Log for observability — does not throw further.
    console.error('[ErrorBoundary] Uncaught render error:', error, info.componentStack)
  }

  handleReset = () => {
    this.setState({ hasError: false, errorMessage: '' })
    window.location.href = '/'
  }

  render() {
    if (this.state.hasError) {
      return (
        <main
          id="main-content"
          tabIndex={-1}
          className="min-h-screen flex flex-col items-center justify-center bg-background p-8 outline-none"
        >
          <div className="max-w-md w-full text-center space-y-4">
            <h1 className="text-2xl font-semibold text-foreground">
              Something went wrong
            </h1>
            <p className="text-sm text-muted-foreground">
              An unexpected error occurred. Please reload the page.
            </p>
            {this.state.errorMessage && (
              <p className="text-xs text-destructive font-mono break-all">
                {this.state.errorMessage}
              </p>
            )}
            <Button onClick={this.handleReset} className="mt-4">
              Reload page
            </Button>
          </div>
        </main>
      )
    }

    return this.props.children
  }
}
