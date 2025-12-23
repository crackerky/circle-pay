import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import LiffApp from './LiffApp.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <LiffApp />
  </StrictMode>,
)
