/**
 * Issue three seats, then download a ZIP of PNGs for print.
 */
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import { Qrdot } from "@qrdot/sdk";
import { loadDotEnv, optionalEnv, requireEnv } from "./lib/env.js";
import { outDir } from "./lib/state.js";
import {
  defaultExpireAt,
  seatIdempotencyKey,
  ticketMetadata,
} from "./lib/ticket.js";

loadDotEnv();

const apiKey = requireEnv("QRDOT_API_KEY");
const eventId = requireEnv("EVENT_ID");
const preEventUrl = requireEnv("PRE_EVENT_URL");
const expireAt = optionalEnv("EXPIRE_AT", defaultExpireAt(7));
const maxScans = Number(optionalEnv("MAX_SCANS", "2"));
const seats = ["A1", "A2", "A3"];

const qrdot = new Qrdot({ apiKey });

const items = seats.map((seat) => ({
  targetUrl: preEventUrl,
  name: `${eventId} · ${seat}`,
  expireAt,
  maxScans,
  metadata: ticketMetadata(eventId, seat),
}));

const { data, errors } = await qrdot.qr.batch(items, {
  idempotencyKey: seatIdempotencyKey(eventId, "batch-sample"),
});

if (errors?.length) {
  console.error("Partial batch errors:", errors);
}

console.log(`Created ${data.length} tickets`);
for (const qr of data) {
  console.log(`  ${qr.metadata?.seat ?? "?"}  ${qr.shortUrl}  ${qr.id}`);
}

const zip = await qrdot.qr.exportImages(
  data.map((q) => q.id),
  "png",
);
const zipPath = join(outDir(), `${eventId}_batch.png.zip`);
writeFileSync(zipPath, zip.bytes);
console.log(`Wrote ${zipPath}`);
