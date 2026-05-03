export const ACTIVITY_TYPES = [
  'Attraction',
  'Beach',
  'Bus',
  'Car',
  'Culinary',
  'Culture',
  'Cycling',
  'Event',
  'Explore',
  'Ferry',
  'Flight',
  'Hiking',
  'Motorscooter',
  'Nature',
  'Other',
  'Shopping',
  'Spa',
  'Sport',
  'Stay',
  'Taxi',
  'Train',
]

export const ACTIVITY_TYPE_EMOJIS = {
  Attraction:   '🎟️',
  Beach:        '🏖️',
  Bus:          '🚌',
  Car:          '🚗',
  Culinary:     '🍜',
  Culture:      '🎭',
  Cycling:      '🚴',
  Event:        '🎫',
  Explore:      '🧭',
  Ferry:        '⛴️',
  Flight:       '✈️',
  Hiking:       '🥾',
  Motorscooter: '🛵',
  Nature:       '🌿',
  Other:        '📍',
  Shopping:     '🛍️',
  Spa:          '💆',
  Sport:        '🏃',
  Stay:         '🛏️',
  Taxi:         '🚕',
  Train:        '🚆',
}

export function formatActivityTypeLabel(type, identification = '') {
  const emoji = ACTIVITY_TYPE_EMOJIS[type] || ACTIVITY_TYPE_EMOJIS.Other
  return identification ? `${emoji} ${type}: ${identification}` : `${emoji} ${type}`
}
