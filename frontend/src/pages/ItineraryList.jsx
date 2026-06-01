import { Link, useNavigate } from 'react-router-dom'
import Header from '../components/Header'
import { useItinerary } from '../context/ItineraryContext'
import { useAuth } from '../context/AuthContext'
import { usePageMeta } from '../hooks/usePageMeta'

function daysToGo(dateStr) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const target = new Date(dateStr)
  target.setHours(0, 0, 0, 0)
  return Math.ceil((target - today) / (1000 * 60 * 60 * 24))
}

function formatDateRange(start, end) {
  const fmt = (d) =>
    new Date(d).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
  if (!end || end === start) return fmt(start)
  const days = Math.ceil((new Date(end) - new Date(start)) / (1000 * 60 * 60 * 24)) + 1
  const nights = days - 1
  return `${fmt(start)} - ${fmt(end)} (${days}D ${nights}N)`
}

function formatCost(amount, currency) {
  if (!amount) return ''
  return currency === 'IDR'
    ? `${currency} ${amount.toLocaleString('id-ID')}`
    : `${currency} ${amount.toLocaleString()}`
}

function computeCost(itinerary) {
  return itinerary.activities.reduce((sum, a) => sum + (a.cost || 0), 0)
}

function cardBgStyle(it) {
  if (it.image) {
    return {
      backgroundImage: `linear-gradient(to top, rgba(0,0,0,0.65) 0%, rgba(0,0,0,0.10) 55%, transparent 100%), linear-gradient(rgba(30,110,100,0.45), rgba(30,110,100,0.45)), url(${it.image})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
    }
  }
  return {}
}

function PencilIcon() {
  return (
    <svg className="svg-icon" viewBox="0 0 24 24">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  )
}

export default function ItineraryList() {
  const { itineraries, loading, error } = useItinerary()
  const { isAuthenticated, user } = useAuth()
  const navigate = useNavigate()

  usePageMeta({
    title: isAuthenticated ? 'My Itineraries' : 'Explore Itineraries',
    description: 'Plan and manage all your travel itineraries in one place with Re-Travel.',
  })

  const today = new Date()
  today.setHours(0, 0, 0, 0)

  const upcoming = itineraries
    .filter((it) => new Date(it.startDate) >= today)
    .sort((a, b) => new Date(a.startDate) - new Date(b.startDate))

  const others = itineraries
    .filter((it) => new Date(it.startDate) < today)
    .sort((a, b) => new Date(b.startDate) - new Date(a.startDate))

  const isOwner = (it) => isAuthenticated && user && String(it.ownerId) === String(user.id)

  const upcomingLabel = isAuthenticated ? 'Your Itineraries Ahead' : 'Upcoming Trips'
  const othersLabel = isAuthenticated ? 'Other Itineraries' : 'Past Trips'

  const headerRight = isAuthenticated ? (
    <Link to="/itinerary/new" className="btn-header">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <rect x="3" y="3" width="18" height="18" rx="4" ry="4"/>
        <line x1="12" y1="8" x2="12" y2="16"/>
        <line x1="8" y1="12" x2="16" y2="12"/>
      </svg>
      <span className="btn-header-label">Plan a New Itinerary</span>
    </Link>
  ) : null

  return (
    <>
      <Header right={headerRight} />

      <div className="page">
        {error && (
          <div style={{ textAlign: 'center', color: 'var(--red-icon)', marginTop: 24 }}>
            Failed to load itineraries: {error}
          </div>
        )}

        {loading && (
          <div style={{ textAlign: 'center', color: 'var(--text-gray)', marginTop: 80 }}>
            Loading itineraries...
          </div>
        )}

        {upcoming.length > 0 && (
          <>
            <p className="section-title">{upcomingLabel}</p>
            <div className="upcoming-grid">
              {upcoming.map((it) => {
                const days = daysToGo(it.startDate)
                const cost = computeCost(it)
                const canEdit = isOwner(it)
                return (
                  <Link
                    key={it.id}
                    to={`/itinerary/${it.slug}`}
                    className="upcoming-card"
                    style={cardBgStyle(it)}
                  >
                    {canEdit && (
                      <button
                        className="icon-btn upcoming-card-edit"
                        onClick={(e) => {
                          e.preventDefault()
                          e.stopPropagation()
                          navigate(`/itinerary/${it.slug}/edit`)
                        }}
                        title="Edit itinerary"
                      >
                        <PencilIcon />
                      </button>
                    )}
                    <div className="upcoming-card-body">
                      <p className="upcoming-card-days">
                        {days > 0
                          ? `${days} days to go!`
                          : days === 0
                          ? "Today's trip!"
                          : 'Trip passed'}
                      </p>
                      <p className="upcoming-card-name">{it.name}</p>
                      <p className="upcoming-card-dates">
                        {formatDateRange(it.startDate, it.endDate)}
                      </p>
                      {cost > 0 && (
                        <p className="upcoming-card-cost">
                          Estimated Cost: {formatCost(cost, it.currency)}
                        </p>
                      )}
                    </div>
                  </Link>
                )
              })}
            </div>
          </>
        )}

        {others.length > 0 && (
          <>
            <p className="section-title">{othersLabel}</p>
            <div className="other-list">
              {others.map((it) => {
                const canEdit = isOwner(it)
                return (
                  <Link
                    key={it.id}
                    to={`/itinerary/${it.slug}`}
                    className="other-list-item"
                  >
                    <span className="other-list-item-name">{it.name}</span>
                    {canEdit && (
                      <button
                        className="icon-btn"
                        onClick={(e) => {
                          e.preventDefault()
                          e.stopPropagation()
                          navigate(`/itinerary/${it.slug}/edit`)
                        }}
                        title="Edit itinerary"
                      >
                        <PencilIcon />
                      </button>
                    )}
                  </Link>
                )
              })}
            </div>
          </>
        )}

        {!isAuthenticated && !loading && itineraries.length > 0 && (
          <div className="public-banner">
            <span>Browsing public itineraries.</span>
            <Link to="/login" className="auth-link"> Log in</Link> to manage your own.
          </div>
        )}

        {!loading && itineraries.length === 0 && (
          <div style={{ textAlign: 'center', color: 'var(--text-gray)', marginTop: 80 }}>
            {isAuthenticated ? (
              <>
                <p style={{ fontSize: 16, marginBottom: 12 }}>No itineraries yet.</p>
                <Link to="/itinerary/new" className="btn btn-primary">
                  <span className="btn-icon-circle">+</span>
                  Plan a New Itinerary
                </Link>
              </>
            ) : (
              <>
                <p style={{ fontSize: 16, marginBottom: 12 }}>No public itineraries yet.</p>
                <Link to="/login" className="btn btn-primary">Sign in to create one</Link>
              </>
            )}
          </div>
        )}
      </div>
    </>
  )
}
