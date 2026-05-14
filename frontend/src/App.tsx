import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { TenantProvider } from '@/components/TenantProvider'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TenantProvider>
        <div className="min-h-screen flex items-center justify-center">
          <Button>BilimBaga</Button>
        </div>
      </TenantProvider>
    </QueryClientProvider>
  )
}

export default App
