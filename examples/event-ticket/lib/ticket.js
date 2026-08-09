/**
 * Idempotency key for a seat — same event+seat retries return the same QR for 24h.
 * @param {string} eventId
 * @param {string} seat
 */
export function seatIdempotencyKey(eventId, seat) {
  const e = eventId.trim().toLowerCase().replace(/[^a-z0-9_-]+/g, "-");
  const s = seat.trim().toUpperCase().replace(/[^A-Z0-9_-]+/g, "-");
  return `ticket-${e}-${s}`;
}

/**
 * @param {string} eventId
 * @param {string} seat
 */
export function ticketMetadata(eventId, seat) {
  return {
    event_id: eventId.trim(),
    seat: seat.trim().toUpperCase(),
  };
}

/**
 * Default expire: now + days (UTC ISO).
 * @param {number} [days=7]
 */
export function defaultExpireAt(days = 7) {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString();
}
