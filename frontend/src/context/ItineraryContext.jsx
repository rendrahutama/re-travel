import { createContext, useContext, useEffect, useState } from 'react'

const ItineraryContext = createContext()
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/$/, '')

function parseResponsePayload(text) {
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

function getToken() {
  return localStorage.getItem('auth_token')
}

async function request(path, options = {}) {
  const token = getToken()
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options.headers || {}),
    },
    ...options,
  })

  const text = await response.text()
  const payload = parseResponsePayload(text)

  if (!response.ok) {
    throw new Error(payload?.error || 'Request failed')
  }

  return payload
}

function mergeItinerary(prev, next) {
  const index = prev.findIndex((item) => item.id === next.id)
  if (index === -1) return [...prev, next]

  const items = [...prev]
  items[index] = next
  return items
}

export function ItineraryProvider({ children }) {
  const [itineraries, setItineraries] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadItineraries = async () => {
    setLoading(true)
    try {
      const items = await request('/api/itineraries')
      setItineraries(items)
      setError('')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadItineraries()
  }, [])

  const clearItineraries = () => {
    setItineraries([])
    setError('')
  }

  const getItinerary = (idOrSlug) => itineraries.find((it) => it.slug === idOrSlug || it.id === idOrSlug)

  const refreshItinerary = async (id) => {
    const item = await request(`/api/itineraries/${id}`)
    setItineraries((prev) => mergeItinerary(prev, item))
    return item
  }

  const addItinerary = async (data) => {
    const item = await request('/api/itineraries', {
      method: 'POST',
      body: JSON.stringify(data),
    })
    setItineraries((prev) => [...prev, item])
    setError('')
    return item.slug
  }

  const updateItinerary = async (id, updates) => {
    const item = await request(`/api/itineraries/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    })
    setItineraries((prev) => prev.map((it) => (it.id === id ? item : it)))
    setError('')
    return item
  }

  const deleteItinerary = async (id) => {
    await request(`/api/itineraries/${id}`, {
      method: 'DELETE',
    })
    setItineraries((prev) => prev.filter((it) => it.id !== id))
    setError('')
  }

  const addActivity = async (itineraryId, data) => {
    const activity = await request(`/api/itineraries/${itineraryId}/activities`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
    await refreshItinerary(itineraryId)
    setError('')
    return activity.id
  }

  const updateActivity = async (itineraryId, activityId, updates) => {
    const activity = await request(`/api/itineraries/${itineraryId}/activities/${activityId}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    })
    await refreshItinerary(itineraryId)
    setError('')
    return activity
  }

  const deleteActivity = async (itineraryId, activityId) => {
    await request(`/api/itineraries/${itineraryId}/activities/${activityId}`, {
      method: 'DELETE',
    })
    await refreshItinerary(itineraryId)
    setError('')
  }

  const moveActivity = async (itineraryId, activityId, direction) => {
    await request(`/api/itineraries/${itineraryId}/activities/${activityId}/move`, {
      method: 'POST',
      body: JSON.stringify({ direction }),
    })
    const item = await refreshItinerary(itineraryId)
    setError('')
    return item
  }

  return (
    <ItineraryContext.Provider
      value={{
        itineraries,
        loading,
        error,
        apiBaseUrl: API_BASE_URL,
        reloadItineraries: loadItineraries,
        clearItineraries,
        getItinerary,
        refreshItinerary,
        addItinerary,
        updateItinerary,
        deleteItinerary,
        addActivity,
        updateActivity,
        deleteActivity,
        moveActivity,
      }}
    >
      {children}
    </ItineraryContext.Provider>
  )
}

export function useItinerary() {
  return useContext(ItineraryContext)
}
