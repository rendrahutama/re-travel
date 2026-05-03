import { useState, useRef, useEffect } from 'react'

export default function LocationInput({ value, onSelect }) {
  const [query, setQuery] = useState(value?.name || '')
  const [suggestions, setSuggestions] = useState([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [showManual, setShowManual] = useState(false)
  const [manualLat, setManualLat] = useState(value?.lat ?? '')
  const [manualLng, setManualLng] = useState(value?.lng ?? '')
  const [reversing, setReversing] = useState(false)
  const debounceRef = useRef()

  // Sync display if parent resets the field (e.g. form clear)
  useEffect(() => {
    if (!value?.name) setQuery('')
  }, [value?.name])

  const search = (q) => {
    clearTimeout(debounceRef.current)
    if (q.trim().length < 3) {
      setSuggestions([])
      setOpen(false)
      return
    }
    debounceRef.current = setTimeout(async () => {
      setLoading(true)
      try {
        const url =
          `https://nominatim.openstreetmap.org/search` +
          `?q=${encodeURIComponent(q)}&format=json&limit=5&addressdetails=1`
        const res = await fetch(url, {
          headers: { 'Accept-Language': 'en' },
        })
        const data = await res.json()
        setSuggestions(data)
        setOpen(data.length > 0)
      } catch {
        setSuggestions([])
        setOpen(false)
      }
      setLoading(false)
    }, 600)
  }

  const handleChange = (e) => {
    const q = e.target.value
    setQuery(q)
    // clear saved coords while user is re-typing
    onSelect({ name: q, address: '', lat: null, lng: null })
    search(q)
  }

  const handleSelect = (place) => {
    const name = place.name || place.display_name.split(',')[0].trim()
    const location = {
      name,
      address: place.display_name,
      lat: parseFloat(place.lat),
      lng: parseFloat(place.lon),
    }
    setQuery(name)
    setSuggestions([])
    setOpen(false)
    onSelect(location)
  }

  const handleClear = () => {
    setQuery('')
    setSuggestions([])
    setOpen(false)
    onSelect({ name: '', address: '', lat: null, lng: null })
  }

  const handleLatPaste = (e) => {
    const text = e.clipboardData.getData('text').trim()
    const parts = text.split(',').map((s) => s.trim())
    if (parts.length === 2) {
      const lat = parseFloat(parts[0])
      const lng = parseFloat(parts[1])
      if (!isNaN(lat) && !isNaN(lng)) {
        e.preventDefault()
        setManualLat(String(lat))
        setManualLng(String(lng))
      }
    }
  }

  const handleLookup = async () => {
    const lat = parseFloat(manualLat)
    const lng = parseFloat(manualLng)
    if (isNaN(lat) || isNaN(lng)) return

    onSelect({ name: value?.name || query, address: value?.address || '', lat, lng })

    setReversing(true)
    try {
      const res = await fetch(
        `https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lng}&format=json`,
        { headers: { 'Accept-Language': 'en' } }
      )
      const data = await res.json()
      if (data.display_name) {
        const reversedName = data.name || data.display_name.split(',')[0].trim()
        onSelect({ name: reversedName, address: data.display_name, lat, lng })
        setQuery(reversedName)
      }
    } catch {
      // silently ignore
    } finally {
      setReversing(false)
    }
  }

  const handleBlur = () => {
    // delay so onMouseDown on a suggestion fires first
    setTimeout(() => setOpen(false), 180)
  }

  const hasCoords = value?.lat && value?.lng
  const hint = value?.address
    ? `${value.address}${hasCoords ? `  (${Number(value.lat).toFixed(5)}, ${Number(value.lng).toFixed(5)})` : ''}`
    : 'Location address will be here after selecting, it will save the lat, long also'

  return (
    <div className="location-wrapper">
      <div className="location-input-row">
        <input
          type="text"
          className="form-input"
          placeholder="Type location...."
          value={query}
          onChange={handleChange}
          onBlur={handleBlur}
          onFocus={() => suggestions.length > 0 && setOpen(true)}
          autoComplete="off"
        />
        {query && (
          <button type="button" className="location-clear-btn" onClick={handleClear} title="Clear location">
            ×
          </button>
        )}
      </div>

      {loading && <p className="form-hint">Searching...</p>}

      {open && suggestions.length > 0 && (
        <ul className="location-suggestions">
          {suggestions.map((place) => (
            <li key={place.place_id} onMouseDown={() => handleSelect(place)}>
              <span className="suggestion-name">
                {place.name || place.display_name.split(',')[0].trim()}
              </span>
              <span className="suggestion-addr">{place.display_name}</span>
            </li>
          ))}
        </ul>
      )}

      {!open && !loading && <p className="form-hint">{hint}</p>}

      <button
        type="button"
        className="location-manual-toggle"
        onClick={() => setShowManual((v) => !v)}
      >
        {showManual ? '▲ Hide coordinates' : '▼ Enter coordinates manually'}
      </button>

      {showManual && (
        <div className="location-manual-row">
          <label className="location-manual-label">
            Lat
            <input
              type="number"
              step="any"
              className="form-input location-manual-input"
              placeholder="e.g. -6.2088"
              value={manualLat}
              onChange={(e) => setManualLat(e.target.value)}
              onPaste={handleLatPaste}
            />
          </label>
          <label className="location-manual-label">
            Lng
            <input
              type="number"
              step="any"
              className="form-input location-manual-input"
              placeholder="e.g. 106.8456"
              value={manualLng}
              onChange={(e) => setManualLng(e.target.value)}
            />
          </label>
          <button
            type="button"
            className="location-lookup-btn"
            onClick={handleLookup}
            disabled={!manualLat || !manualLng || reversing}
            title="Lookup address from coordinates"
          >
            {reversing ? '...' : 'Lookup'}
          </button>
        </div>
      )}
    </div>
  )
}
