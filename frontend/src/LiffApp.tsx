import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import CreateEvent from './pages/CreateEvent';
import ApprovePage from './pages/ApprovePage';
import EventsPage from './pages/EventsPage';
import CirclesPage from './pages/CirclesPage';

// エラー境界コンポーネント
interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

class ErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('React Error:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: '20px', textAlign: 'center' }}>
          <h2>エラーが発生しました</h2>
          <p style={{ color: 'red' }}>{this.state.error?.message}</p>
          <pre style={{ textAlign: 'left', background: '#f5f5f5', padding: '10px', overflow: 'auto' }}>
            {this.state.error?.stack}
          </pre>
          <button onClick={() => window.location.reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

export default function LiffApp() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Navigate to="/events" replace />} />
          <Route path="/events" element={<EventsPage />} />
          <Route path="/create" element={<CreateEvent />} />
          <Route path="/approve" element={<ApprovePage />} />
          <Route path="/circles" element={<CirclesPage />} />
        </Routes>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
