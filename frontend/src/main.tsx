import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { i18nReady } from './i18n'
import './index.css'
import { clearStaleE2eToken } from './lib/e2eTokenSeed'

clearStaleE2eToken()

if (import.meta.env.MODE !== 'production') {
  import('@axe-core/react').then(({ default: axe }) => {
    axe(React, ReactDOM, 1000)
  })
}

// Render only once the persisted locale is active, so a kk/ru user never sees the first paint in English.
i18nReady.then(() => {
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>,
  )
})
