import { useState, useRef, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useItinerary } from '../context/ItineraryContext'

function UserMenu({ user, onLogout }) {
  const [open, setOpen] = useState(false)
  const menuRef = useRef()

  useEffect(() => {
    if (!open) return
    const handleClick = (e) => {
      if (!menuRef.current?.contains(e.target)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  return (
    <div className="user-menu" ref={menuRef}>
      <button className="user-menu-btn" onClick={() => setOpen(!open)}>
        <span>{user.name}</span>
        <svg width="10" height="6" viewBox="0 0 10 6" fill="none" style={{ opacity: 0.7, transition: 'transform 0.15s', transform: open ? 'rotate(180deg)' : 'none' }}>
          <path d="M1 1l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      </button>
      {open && (
        <div className="user-menu-dropdown">
          <p className="user-menu-email">{user.email}</p>
          <button className="user-menu-logout" onClick={onLogout}>Logout</button>
        </div>
      )}
    </div>
  )
}

export default function Header({ right }) {
  const { user, logout } = useAuth()
  const { clearItineraries, reloadItineraries } = useItinerary()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    clearItineraries()
    await reloadItineraries()
    navigate('/')
  }

  return (
    <header className="header">
      <Link to="/" className="header-logo">
        <i>Re-Itinerary</i>
      </Link>
      <div className="header-right">
        {right}
        {user ? (
          <UserMenu user={user} onLogout={handleLogout} />
        ) : (
          <Link to="/login" className="header-link">Login</Link>
        )}
      </div>
    </header>
  )
}
