import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from './context/AuthContext'
import { ItineraryProvider } from './context/ItineraryContext'
import ItineraryList from './pages/ItineraryList'
import ItineraryDetail from './pages/ItineraryDetail'
import EditItinerary from './pages/EditItinerary'
import EditActivity from './pages/EditActivity'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'

function RequireAuth({ children }) {
  const { isAuthenticated, loading } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div style={{ textAlign: 'center', color: 'var(--text-gray)', marginTop: 80 }}>
        Loading…
      </div>
    )
  }
  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }
  return children
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/" element={<ItineraryList />} />
      <Route path="/itinerary/new" element={<RequireAuth><EditItinerary /></RequireAuth>} />
      <Route path="/itinerary/:id" element={<ItineraryDetail />} />
      <Route path="/itinerary/:id/edit" element={<RequireAuth><EditItinerary /></RequireAuth>} />
      <Route path="/itinerary/:id/activity/new" element={<RequireAuth><EditActivity /></RequireAuth>} />
      <Route path="/itinerary/:id/activity/:activityId/edit" element={<RequireAuth><EditActivity /></RequireAuth>} />
    </Routes>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <ItineraryProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </ItineraryProvider>
    </AuthProvider>
  )
}
