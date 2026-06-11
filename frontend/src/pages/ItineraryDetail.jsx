import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import Header from '../components/Header'
import { DatetimeInput } from '../components/DateInput'
import LocationInput from '../components/LocationInput'
import TripMap from '../components/TripMap'
import { useItinerary } from '../context/ItineraryContext'
import { useAuth } from '../context/AuthContext'
import { usePageMeta } from '../hooks/usePageMeta'
import { ACTIVITY_TYPES, formatActivityTypeLabel } from '../data/activityTypes'

const TICKET_STATUSES = ['Secured', 'Unbooked', 'Go Show']

/* ─── SVG Icons ─── */
function PencilIcon() {
  return (
    <svg className="svg-icon" viewBox="0 0 24 24">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  )
}

function ShareIcon() {
  return (
    <svg className="svg-icon" viewBox="0 0 24 24">
      <circle cx="18" cy="5" r="3" />
      <circle cx="6" cy="12" r="3" />
      <circle cx="18" cy="19" r="3" />
      <line x1="8.59" y1="13.51" x2="15.42" y2="17.49" />
      <line x1="15.41" y1="6.51" x2="8.59" y2="10.49" />
    </svg>
  )
}

/* ─── Helpers ─── */
function formatDatetimeLocal(date) {
  const pad = (n) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function getDefaultActivityDatetime(itinerary) {
  const sorted = [...(itinerary?.activities || [])].sort((a, b) => a.sortOrder - b.sortOrder)
  const previous = [...sorted].reverse().find((activity) => activity.datetime)

  if (previous?.datetime) {
    const previousDate = new Date(previous.datetime)
    if (!Number.isNaN(previousDate.getTime())) {
      previousDate.setHours(previousDate.getHours() + 1)
      return formatDatetimeLocal(previousDate)
    }
  }

  if (itinerary?.startDate) {
    return `${itinerary.startDate}T09:00`
  }

  return formatDatetimeLocal(new Date())
}

function formatTime(datetimeStr) {
  if (!datetimeStr) return ''
  const [, timePart] = datetimeStr.split('T')
  if (!timePart) return ''
  const [h, m] = timePart.split(':').map(Number)
  const ampm = h < 12 ? 'am' : 'pm'
  return `${String(h).padStart(2, '0')}.${String(m).padStart(2, '0')} ${ampm}`
}

function formatDate(dateStr) {
  const [y, mo, d] = dateStr.split('-').map(Number)
  return new Date(y, mo - 1, d).toLocaleDateString('en-GB', {
    day: 'numeric', month: 'long', year: 'numeric',
  })
}

function formatCost(amount, currency) {
  if (!amount) return ''
  return currency === 'IDR'
    ? `IDR ${Number(amount).toLocaleString('id-ID')},-`
    : `${currency} ${Number(amount).toLocaleString()}`
}

function formatEstimate(amount, currency) {
  if (!amount) return ''
  return currency === 'IDR'
    ? `IDR ${Number(amount).toLocaleString('id-ID')}`
    : `${currency} ${Number(amount).toLocaleString()}`
}

function formatTripLength(startDate, endDate) {
  if (!startDate || !endDate) return ''
  const days = Math.max(
    1,
    Math.ceil((new Date(endDate) - new Date(startDate)) / (1000 * 60 * 60 * 24)) + 1
  )
  const nights = Math.max(0, days - 1)
  return `${days}D ${nights}N`
}

function statusClass(status) {
  if (status === 'Secured') return 'ticket-secured'
  if (status === 'Unbooked') return 'ticket-unbooked'
  if (status === 'Go Show') return 'ticket-goshow'
  return ''
}

function groupByDate(activities) {
  const sorted = [...activities].sort((a, b) => a.sortOrder - b.sortOrder)
  const groups = {}
  for (const act of sorted) {
    const key = act.datetime ? act.datetime.slice(0, 10) : 'unscheduled'
    if (!groups[key]) groups[key] = []
    groups[key].push(act)
  }
  return Object.entries(groups).sort(([a], [b]) => a.localeCompare(b))
}

function getDayNumber(startDate, dateKey) {
  if (dateKey === 'unscheduled') return null
  const [sy, sm, sd] = startDate.split('-').map(Number)
  const [cy, cm, cd] = dateKey.split('-').map(Number)
  const start = new Date(sy, sm - 1, sd)
  const current = new Date(cy, cm - 1, cd)
  return Math.floor((current - start) / 86400000) + 1
}

function fmtCostInput(val) {
  const raw = String(val || '').replace(/[^\d]/g, '')
  if (!raw) return ''
  return parseInt(raw, 10).toLocaleString('en-US')
}

function parseCostInput(val) {
  return parseFloat(String(val || '').replace(/,/g, '')) || 0
}

function defaultForm(itinerary, act = null) {
  return {
    datetime: act?.datetime ?? getDefaultActivityDatetime(itinerary),
    type: act?.type ?? '',
    identification: act?.identification ?? '',
    location: act?.location ?? { name: '', address: '', lat: null, lng: null },
    cost: act?.cost ? fmtCostInput(act.cost) : '',
    ticketStatus: act?.ticketStatus ?? '',
    details: act?.details ?? '',
  }
}

/* ─── Inline Activity Form ─── */
function ActivityForm({ currency, itinerary, existing, onSave, onCancel, onDelete }) {
  const [form, setForm] = useState(() => defaultForm(itinerary, existing))
  const set = (k, v) => setForm((f) => ({ ...f, [k]: v }))

  const handleSave = () => {
    onSave({
      datetime: form.datetime || getDefaultActivityDatetime(itinerary),
      type: form.type || 'Other',
      identification: form.identification.trim(),
      location: form.location,
      cost: parseCostInput(form.cost),
      ticketStatus: form.ticketStatus || null,
      details: form.details.trim(),
    })
  }

  return (
    <div className="inline-form-card">
      <div className="activity-form-grid">
        <label className="activity-form-label">Date Time</label>
        <div className="activity-form-field">
          <DatetimeInput value={form.datetime} onChange={(e) => set('datetime', e.target.value)} />
        </div>

        <div /><div style={{ height: 8 }} />

        <label className="activity-form-label">Activity Type</label>
        <div className="activity-form-field">
          <select className="form-select" value={form.type} onChange={(e) => set('type', e.target.value)}>
            <option value="" disabled>Select activity type</option>
            {ACTIVITY_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>
        </div>

        <div />
        <div className="activity-form-field">
          <input
            type="text"
            className="form-input"
            placeholder="Identification (e.g. GA-417 for Flight)"
            value={form.identification}
            onChange={(e) => set('identification', e.target.value)}
          />
        </div>

        <div /><div style={{ height: 8 }} />

        <label className="activity-form-label">Location</label>
        <div className="activity-form-field">
          <LocationInput
            value={form.location}
            onSelect={(loc) => set('location', loc)}
          />
        </div>

        <div /><div style={{ height: 8 }} />

        <label className="activity-form-label">Cost</label>
        <div className="activity-form-field">
          <div className="cost-row">
            <span className="cost-currency-label">{currency}</span>
            <input
              type="text"
              inputMode="numeric"
              className="form-input"
              placeholder="0"
              value={form.cost}
              onChange={(e) => set('cost', fmtCostInput(e.target.value))}
            />
          </div>
          <p className="form-hint">Leave blank if free</p>
        </div>

        <div /><div style={{ height: 8 }} />

        <label className="activity-form-label">Ticket Status</label>
        <div className="activity-form-field">
          <select className="form-select" value={form.ticketStatus} onChange={(e) => set('ticketStatus', e.target.value)}>
            <option value="">Select ticket status</option>
            {TICKET_STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>

        <div /><div style={{ height: 8 }} />

        <label className="activity-form-label">Details</label>
        <div className="activity-form-field">
          <textarea
            className="form-textarea"
            value={form.details}
            onChange={(e) => set('details', e.target.value)}
            rows={4}
          />
        </div>
      </div>

      <div className="form-btn-row" style={{ justifyContent: 'center', marginTop: 20 }}>
        <button className="btn btn-primary" onClick={handleSave}>
          <span className="btn-icon-circle">+</span>
          Save
        </button>
        {onDelete && (
          <button className="btn btn-danger" onClick={onDelete}>Delete</button>
        )}
        <button className="btn btn-cancel" onClick={onCancel}>Cancel</button>
      </div>
    </div>
  )
}

/* ─── AI Activity Import ─── */
const VALID_TICKET = ['Secured', 'Unbooked', 'Go Show']

const AI_SCHEMA_SAMPLE = JSON.stringify(
  [
    {
      datetime: '2026-05-10T09:00',
      type: 'Flight',
      identification: 'GA-417',
      location: {
        name: 'Soekarno-Hatta Airport',
        address: 'Soekarno-Hatta International Airport, Tangerang, Banten, Indonesia',
        lat: -6.1256,
        lng: 106.6558,
      },
      cost: 1500000,
      ticketStatus: 'Secured',
      details: 'Check in 2 hours before departure',
    },
  ],
  null,
  2
)

function buildAiPrompt(currency, startDate) {
  return `Generate a list of trip activities in JSON format using this exact schema:

${AI_SCHEMA_SAMPLE}

Rules:
- "datetime" format: YYYY-MM-DDTHH:MM
- Trip starts on ${startDate} — use this as the base date for all activities
- "type" must be one of: Attraction, Beach, Bus, Car, Culinary, Culture, Cycling, Event, Explore, Ferry, Flight, Hiking, Motorscooter, Nature, Other, Shopping, Spa, Sport, Stay, Taxi, Train
- "ticketStatus" must be one of: "Secured", "Unbooked", "Go Show", or "" (empty string)
- "cost" is a number in ${currency}, use 0 if free
- "lat" and "lng" are decimal GPS coordinates (use null if unknown)
- Return ONLY a valid JSON array, no extra text`
}

function ActivityImport({ itineraryId, currency, startDate }) {
  const { addActivity } = useItinerary()
  const [open, setOpen] = useState(false)
  const [json, setJson] = useState('')
  const [parsed, setParsed] = useState(null)
  const [parseError, setParseError] = useState('')
  const [importing, setImporting] = useState(false)
  const [progress, setProgress] = useState({ done: 0, total: 0 })
  const [copied, setCopied] = useState(false)

  const handleJsonChange = (e) => {
    const val = e.target.value
    setJson(val)
    if (!val.trim()) { setParsed(null); setParseError(''); return }
    try {
      const data = JSON.parse(val)
      if (!Array.isArray(data)) throw new Error('Must be a JSON array [ ... ]')
      if (data.length === 0) throw new Error('Array is empty')
      setParsed(data)
      setParseError('')
    } catch (err) {
      setParsed(null)
      setParseError(err.message)
    }
  }

  const handleCopyPrompt = async () => {
    await navigator.clipboard.writeText(buildAiPrompt(currency, startDate))
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleImport = async () => {
    if (!parsed?.length) return
    setImporting(true)
    setProgress({ done: 0, total: parsed.length })
    let ok = 0
    for (let i = 0; i < parsed.length; i++) {
      const act = parsed[i]
      try {
        await addActivity(itineraryId, {
          datetime: act.datetime || '',
          type: ACTIVITY_TYPES.includes(act.type) ? act.type : 'Other',
          identification: act.identification || '',
          location: act.location && typeof act.location === 'object'
            ? {
                name: act.location.name || '',
                address: act.location.address || '',
                lat: act.location.lat ?? null,
                lng: act.location.lng ?? null,
              }
            : { name: '', address: '', lat: null, lng: null },
          cost: typeof act.cost === 'number' ? act.cost : 0,
          ticketStatus: VALID_TICKET.includes(act.ticketStatus) ? act.ticketStatus : null,
          details: act.details || '',
        })
        ok++
      } catch {
        // skip failed individual activity
      }
      setProgress({ done: i + 1, total: parsed.length })
    }
    setImporting(false)
    setJson('')
    setParsed(null)
    setOpen(false)
    alert(`Imported ${ok} of ${parsed.length} activities.`)
  }

  return (
    <div className="ai-import-wrap">
      <button
        type="button"
        className="ai-import-toggle"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? '▲ Close AI import' : '▼ Import activities from AI'}
      </button>

      {open && (
        <div className="ai-import-panel">
          <p className="ai-import-hint">
            Copy the schema prompt below, paste it into ChatGPT, Claude, or any AI chatbot along with your trip details, then paste the generated JSON here.
          </p>

          <div className="ai-import-schema-row">
            <pre className="ai-import-schema">{AI_SCHEMA_SAMPLE}</pre>
            <button
              type="button"
              className="ai-import-copy-btn"
              onClick={handleCopyPrompt}
            >
              {copied ? 'Copied!' : 'Copy AI prompt'}
            </button>
          </div>

          <textarea
            className="ai-import-textarea"
            placeholder="Paste the AI-generated JSON array here..."
            value={json}
            onChange={handleJsonChange}
            rows={8}
            disabled={importing}
          />

          {parseError && (
            <p className="ai-import-error">⚠ {parseError}</p>
          )}

          {parsed && !parseError && (
            <p className="ai-import-count">
              {parsed.length} {parsed.length === 1 ? 'activity' : 'activities'} ready to import
            </p>
          )}

          {importing && (
            <p className="ai-import-progress">
              Importing {progress.done} / {progress.total}...
            </p>
          )}

          <div className="ai-import-actions">
            <button
              type="button"
              className="btn btn-primary"
              onClick={handleImport}
              disabled={!parsed || importing}
            >
              {importing ? 'Importing...' : `Import ${parsed?.length ?? 0} activities`}
            </button>
            <button
              type="button"
              className="btn btn-cancel"
              onClick={() => { setOpen(false); setJson(''); setParsed(null); setParseError('') }}
              disabled={importing}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

/* ─── Day Table of Contents ─── */
function DayToC({ groups, startDate }) {
  const [activeKey, setActiveKey] = useState(null)

  useEffect(() => {
    if (groups.length <= 1) return
    const observers = groups.map(([dateKey]) => {
      const el = document.getElementById(`day-${dateKey}`)
      if (!el) return null
      const obs = new IntersectionObserver(
        ([entry]) => { if (entry.isIntersecting) setActiveKey(dateKey) },
        { rootMargin: '-10% 0px -80% 0px' }
      )
      obs.observe(el)
      return obs
    }).filter(Boolean)
    return () => observers.forEach((o) => o.disconnect())
  }, [groups])

  if (groups.length <= 1) return null

  const scrollTo = (dateKey) => {
    const el = document.getElementById(`day-${dateKey}`)
    if (!el) return
    const top = el.getBoundingClientRect().top + window.scrollY - 80
    window.scrollTo({ top, behavior: 'smooth' })
  }

  return (
    <nav className="day-toc">
      {groups.map(([dateKey]) => {
        const dayNum = getDayNumber(startDate, dateKey)
        const label = dateKey === 'unscheduled' ? '–' : `Day ${dayNum}`
        return (
          <button
            key={dateKey}
            className={`day-toc-btn${activeKey === dateKey ? ' day-toc-btn-active' : ''}`}
            onClick={() => scrollTo(dateKey)}
            title={dateKey === 'unscheduled' ? 'Unscheduled' : `Day ${dayNum}`}
          >
            {label}
          </button>
        )
      })}
    </nav>
  )
}

/* ─── Main Page ─── */
export default function ItineraryDetail() {
  const { id } = useParams()
  const { getItinerary, refreshItinerary, addActivity, updateActivity, deleteActivity, moveActivity, loading } = useItinerary()
  const { user } = useAuth()

  const [editingId, setEditingId] = useState(null)
  const [addingActivity, setAddingActivity] = useState(false)
  const [saving, setSaving] = useState(false)
  const [fetching, setFetching] = useState(false)
  const [fetchFailed, setFetchFailed] = useState(false)

  const itinerary = getItinerary(id)

  // Fetch directly if not in cache (shared link / unauthenticated access)
  useEffect(() => {
    if (!itinerary && !fetching && !fetchFailed && !loading) {
      setFetching(true)
      refreshItinerary(id)
        .catch(() => setFetchFailed(true))
        .finally(() => setFetching(false))
    }
  }, [id, itinerary, loading])

  const isOwner = !!(user && itinerary && itinerary.ownerEmail === user.email)

  usePageMeta({
    title: itinerary?.name,
    description: itinerary?.description
      || (itinerary ? `View the ${itinerary.name} travel itinerary` : undefined),
    image: itinerary?.image ?? undefined,
    type: 'article',
  })

  if (loading || fetching) {
    return (
      <>
        <Header />
        <div className="page" style={{ textAlign: 'center', marginTop: 60 }}>
          <p>Loading itinerary...</p>
        </div>
      </>
    )
  }

  if (!itinerary || fetchFailed) {
    return (
      <>
        <Header />
        <div className="page" style={{ textAlign: 'center', marginTop: 60 }}>
          <p>Itinerary not found.</p>
          <Link to="/" className="btn btn-primary" style={{ marginTop: 16 }}>Back to List</Link>
        </div>
      </>
    )
  }

  const groups = groupByDate(itinerary.activities)
  const sortedActivities = [...itinerary.activities].sort((a, b) => a.sortOrder - b.sortOrder)
  const currency = itinerary.currency || 'IDR'
  const totalCost = itinerary.activities.reduce((sum, a) => sum + (a.cost || 0), 0)

  const handleShare = () => {
    navigator.clipboard.writeText(window.location.href).then(() => alert('Link copied!'))
  }

  const handleAddSave = async (payload) => {
    setSaving(true)
    try {
      await addActivity(id, payload)
      setAddingActivity(false)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleEditSave = async (payload) => {
    setSaving(true)
    try {
      await updateActivity(id, editingId, payload)
      setEditingId(null)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleEditDelete = async () => {
    if (!window.confirm('Delete this activity?')) return
    setSaving(true)
    try {
      await deleteActivity(id, editingId)
      setEditingId(null)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  const openEdit = (actId) => {
    setAddingActivity(false)
    setEditingId(actId)
  }

  const openAdd = () => {
    setEditingId(null)
    setAddingActivity(true)
  }

  const handleMove = async (activityId, direction) => {
    setSaving(true)
    try {
      await moveActivity(id, activityId, direction)
    } catch (err) {
      alert(err.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <Header />

      <DayToC groups={groups} startDate={itinerary.startDate} />
      <div className="page-wide">
        {/* Trip header: info card (60%) + map (40%) */}
        <div className="trip-header-row">
          <div
            className="trip-info-card"
            style={itinerary.image ? {
              backgroundImage: `url(${itinerary.image})`,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
            } : {}}
          >
            {itinerary.image && <div className="trip-card-overlay" />}
            <div className="trip-info-content">
              <div className="trip-info-actions">
                {isOwner && (
                  <Link to={`/itinerary/${id}/edit`} className="icon-btn" title="Edit itinerary">
                    <PencilIcon />
                  </Link>
                )}
                <button className="icon-btn" onClick={handleShare} title="Share">
                  <ShareIcon />
                </button>
              </div>
              <h1 className="trip-title">{itinerary.name}</h1>
              {itinerary.description && (
                <p className="trip-description">{itinerary.description}</p>
              )}
              <div className="trip-meta">
                <div className="trip-meta-item">
                  <span className="trip-meta-label">Planned Date</span>
                  <span className="trip-meta-value">{formatDate(itinerary.startDate)}</span>
                </div>
                <div className="trip-meta-item">
                  <span className="trip-meta-label">Trip Length</span>
                  <span className="trip-meta-value">
                    {formatTripLength(itinerary.startDate, itinerary.endDate)}
                  </span>
                </div>
                {totalCost > 0 && (
                  <div className="trip-meta-item trip-meta-item-cost">
                    <span className="trip-meta-label">Estimated Cost</span>
                    <span className="trip-meta-value">{formatEstimate(totalCost, currency)}</span>
                  </div>
                )}
                {itinerary.ownerEmail && (
                  <div className="trip-meta-item">
                    <span className="trip-meta-label">Created by</span>
                    <span className="trip-meta-value">{itinerary.ownerEmail}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Day groups */}
        {groups.length === 0 && !addingActivity && (
          <p style={{ color: 'var(--text-gray)', textAlign: 'center', margin: '40px 0' }}>
            {isOwner
              ? 'No activities yet. Add your first activity below!'
              : 'No activities have been added to this itinerary yet.'}
          </p>
        )}

        {groups.map(([dateKey, acts]) => {
          const dayNum = getDayNumber(itinerary.startDate, dateKey)
          const dateLabel =
            dateKey === 'unscheduled'
              ? 'Unscheduled'
              : formatDate(dateKey)
          const mapActivities = acts.filter((act) => act.location?.lat && act.location?.lng)

          return (
            <div key={dateKey} id={`day-${dateKey}`} className="day-section">
              <p className="day-label">
                {dayNum != null && <span className="day-label-num">DAY {dayNum} </span>}
                <span>({dateLabel})</span>
              </p>

              {mapActivities.length > 0 && (
                <div className="day-map-wrapper">
                  <TripMap activities={mapActivities} />
                </div>
              )}

              <div className="timeline">
                {acts.map((act) => {
                  const globalIdx = sortedActivities.findIndex((a) => a.id === act.id)
                  const isFirst = globalIdx === 0
                  const isLast = globalIdx === sortedActivities.length - 1

                  if (isOwner && editingId === act.id) {
                    return (
                      <div key={act.id} className="timeline-item">
                        <div className="timeline-dot" />
                        <ActivityForm
                          currency={currency}
                          itinerary={itinerary}
                          existing={act}
                          onSave={handleEditSave}
                          onCancel={() => setEditingId(null)}
                          onDelete={handleEditDelete}
                        />
                      </div>
                    )
                  }

                  return (
                    <div key={act.id} className="timeline-item">
                      <div className="timeline-dot" />
                      <p className="timeline-time">{formatTime(act.datetime)}</p>

                      <div className="activity-card">
                        {isOwner && (
                          <div className="activity-card-actions">
                            {!isLast && (
                              <button className="arrow-btn" onClick={() => handleMove(act.id, 'down')} title="Move down" disabled={saving}>▼</button>
                            )}
                            {!isFirst && (
                              <button className="arrow-btn" onClick={() => handleMove(act.id, 'up')} title="Move up" disabled={saving}>▲</button>
                            )}
                            <button className="icon-btn" onClick={() => openEdit(act.id)} title="Edit activity" disabled={saving}>
                              <PencilIcon />
                            </button>
                          </div>
                        )}

                        <p className="activity-type">
                          <span className="activity-type-badge">
                            {formatActivityTypeLabel(act.type, act.identification)}
                          </span>
                        </p>

                        {act.location?.name && (
                          <div className="activity-row">
                            <span className="activity-icon" style={{ color: 'var(--red-icon)' }}>📍</span>
                            <div>
                              <p className="activity-location-name">{act.location.name}</p>
                              {act.location.address && (
                                <p className="activity-location-addr">{act.location.address}</p>
                              )}
                            </div>
                          </div>
                        )}

                        <div className="activity-row">
                          <span className="activity-icon" style={{ color: 'var(--orange)' }}>💸</span>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                            <span className="activity-cost">
                              {act.cost > 0 ? formatCost(act.cost, currency) : 'Free'}
                            </span>
                            {act.ticketStatus && (
                              <span className={`ticket-badge ${statusClass(act.ticketStatus)}`}>
                                {act.ticketStatus}
                              </span>
                            )}
                          </div>
                        </div>

                        {act.details && (
                          <div className="activity-row">
                            <span className="activity-icon" style={{ color: 'var(--orange)' }}>📝</span>
                            <p className="activity-details-text">
                              <span className="activity-details-label">Details:</span>
                              <br />
                              {act.details}
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )
        })}

        {/* Inline add form (owner only) */}
        {isOwner && addingActivity && (
          <ActivityForm
            currency={currency}
            itinerary={itinerary}
            onSave={handleAddSave}
            onCancel={() => setAddingActivity(false)}
          />
        )}

        {/* Add Activities button + AI import (owner only) */}
        {isOwner && !addingActivity && (
          <div className="add-activity-row">
            <button className="btn btn-primary" onClick={openAdd}>
              <span className="btn-icon-circle">+</span>
              Add Activities
            </button>
          </div>
        )}
        {isOwner && !addingActivity && !editingId && (
          <ActivityImport itineraryId={id} currency={currency} startDate={itinerary.startDate} />
        )}
      </div>
    </>
  )
}
