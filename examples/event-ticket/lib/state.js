import { mkdirSync, readFileSync, writeFileSync, existsSync } from "node:fs";
import { dirname, resolve } from "node:path";

const STATE_PATH = resolve(process.cwd(), "out", "ticket.json");

/**
 * @typedef {{
 *   id: string,
 *   shortUrl: string,
 *   shortCode: string,
 *   eventId: string,
 *   seat: string,
 *   targetUrl: string,
 * }} TicketState
 */

/** @returns {TicketState} */
export function loadTicket() {
  if (!existsSync(STATE_PATH)) {
    console.error("No out/ticket.json — run npm run issue first.");
    process.exit(1);
  }
  return JSON.parse(readFileSync(STATE_PATH, "utf8"));
}

/** @param {TicketState} state */
export function saveTicket(state) {
  mkdirSync(dirname(STATE_PATH), { recursive: true });
  writeFileSync(STATE_PATH, `${JSON.stringify(state, null, 2)}\n`);
  return STATE_PATH;
}

export function outDir() {
  const dir = resolve(process.cwd(), "out");
  mkdirSync(dir, { recursive: true });
  return dir;
}
