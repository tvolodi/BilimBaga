import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { TenantProvider } from '@/components/TenantProvider'
import { UsersListPage } from '@/pages/users/UsersListPage'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <TenantProvider>
        <BrowserRouter>
          <Routes>
            <Route
              path="/"
              element={
                <div className="min-h-screen flex items-center justify-center">
                  <Button>BilimBaga</Button>
                </div>
              }
            />
            <Route path="/users" element={<UsersListPage />} />
          </Routes>
        </BrowserRouter>
      </TenantProvider>
    </QueryClientProvider>
  )
}

export default App

