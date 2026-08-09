/**
 * Download PNG + PDF for the ticket in out/ticket.json.
 * Free plans include a small qrdot.dev watermark on every export.
 */
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import { Qrdot } from "@qrdot/sdk";
import { loadDotEnv, requireEnv } from "./lib/env.js";
import { loadTicket, outDir } from "./lib/state.js";

loadDotEnv();

const apiKey = requireEnv("QRDOT_API_KEY");
const ticket = loadTicket();
const qrdot = new Qrdot({ apiKey });
const dir = outDir();
const base = `${ticket.eventId}_${ticket.seat}`;

for (const format of /** @type {const} */ (["png", "pdf"])) {
  const img = await qrdot.qr.image(ticket.id, format);
  const file = join(dir, `${base}.${format}`);
  writeFileSync(file, img.bytes);
  console.log(`Wrote ${file} (${img.bytes.byteLength} bytes)`);
}

console.log("");
console.log(`Print ${base}.png or ${base}.pdf — short link stays ${ticket.shortUrl}`);
console.log("Free exports include a qrdot.dev watermark (removed on paid plans).");
