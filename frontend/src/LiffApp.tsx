import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import CreateEvent from './pages/CreateEvent';
import ApprovePage from './pages/ApprovePage';
import EventsPage from './pages/EventsPage';

export default function LiffApp() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/events" replace />} />
        <Route path="/events" element={<EventsPage />} />
        <Route path="/create" element={<CreateEvent />} />
        <Route path="/approve" element={<ApprovePage />} />
      </Routes>
    </BrowserRouter>
  );
}
