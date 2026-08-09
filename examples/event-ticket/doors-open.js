/**
 * Point the printed ticket at the live doors page — no reprint.
 */
import { Qrdot } from "@qrdot/sdk";
import { loadDotEnv, requireEnv } from "./lib/env.js";
import { loadTicket, saveTicket } from "./lib/state.js";

loadDotEnv();

const apiKey = requireEnv("QRDOT_API_KEY");
const doorsUrl = requireEnv("DOORS_URL");
const ticket = loadTicket();
const qrdot = new Qrdot({ apiKey });

const qr = await qrdot.qr.update(ticket.id, { targetUrl: doorsUrl });

saveTicket({
  ...ticket,
  targetUrl: qr.targetUrl,
  shortUrl: qr.shortUrl,
});

console.log("Doors open — destination updated");
console.log(`  id         ${qr.id}`);
console.log(`  shortUrl   ${qr.shortUrl}  (unchanged print)`);
console.log(`  targetUrl  ${qr.targetUrl}`);
console.log("");
console.log("Scan or curl -I the short URL — Location should be the doors page.");
