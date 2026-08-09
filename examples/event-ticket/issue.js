/**
 * Mint a single-seat event ticket QR (expire + maxScans + metadata).
 * Safe to retry — same EVENT_ID + SEAT → same QR for 24h (Idempotency-Key).
 */
import { Qrdot } from "@qrdot/sdk";
import { loadDotEnv, optionalEnv, requireEnv } from "./lib/env.js";
import { saveTicket } from "./lib/state.js";
import {
  defaultExpireAt,
  seatIdempotencyKey,
  ticketMetadata,
} from "./lib/ticket.js";

loadDotEnv();

const apiKey = requireEnv("QRDOT_API_KEY");
const eventId = requireEnv("EVENT_ID");
const seat = optionalEnv("SEAT", "A12");
const preEventUrl = requireEnv("PRE_EVENT_URL");
const expireAt = optionalEnv("EXPIRE_AT", defaultExpireAt(7));
const maxScans = Number(optionalEnv("MAX_SCANS", "2"));

if (!Number.isInteger(maxScans) || maxScans < 1) {
  console.error("MAX_SCANS must be a positive integer");
  process.exit(1);
}

const qrdot = new Qrdot({ apiKey });
const metadata = ticketMetadata(eventId, seat);
const idempotencyKey = seatIdempotencyKey(eventId, seat);

const qr = await qrdot.qr.create(
  {
    targetUrl: preEventUrl,
    name: `${eventId} · ${metadata.seat}`,
    expireAt,
    maxScans,
    metadata,
  },
  { idempotencyKey },
);

const path = saveTicket({
  id: qr.id,
  shortUrl: qr.shortUrl,
  shortCode: qr.shortCode,
  eventId,
  seat: metadata.seat,
  targetUrl: qr.targetUrl,
});

console.log("Ticket issued");
console.log(`  id        ${qr.id}`);
console.log(`  shortUrl  ${qr.shortUrl}`);
console.log(`  seat      ${metadata.seat}`);
console.log(`  expireAt  ${expireAt}`);
console.log(`  maxScans  ${maxScans}`);
console.log(`  state     ${path}`);
console.log("");
console.log("Next: npm run image   then open the short URL once.");
console.log("Later: npm run doors-open  to change the landing without reprint.");
