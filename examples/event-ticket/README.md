# Event ticket QR (example)

Mint a **printable event ticket** with the [QR.](https://qrdot.dev) API: capped scans, expiry, seat metadata — then **change the landing page** after print without reprinting the code.

Public mirror: [`intuitix/qrdot-clients`](https://github.com/intuitix/qrdot-clients/tree/main/examples/event-ticket)

## What you get

1. `POST /v1/qr` with `expireAt`, `maxScans`, and `metadata: { event_id, seat }`
2. PNG + PDF export for the ticket stub
3. `PATCH` destination when doors open (`npm run doors-open`)

Free plan image exports include a small `qrdot.dev` watermark (upgrade to remove).

## 5-minute path

1. Sign up at [app.qrdot.dev](https://app.qrdot.dev/?signup=1) → **API keys** → create `sk_live_…`
2. Clone this folder (from the public clients repo):

   ```bash
   git clone https://github.com/intuitix/qrdot-clients.git
   cd qrdot-clients/examples/event-ticket
   npm install
   cp .env.example .env
   # edit .env — set QRDOT_API_KEY
   ```

3. Issue a ticket, export art, open the short link, then flip the landing:

   ```bash
   npm run issue
   npm run image          # writes out/<event>_<seat>.png and .pdf
   open "$(node -e "console.log(require('./out/ticket.json').shortUrl)")"  # or paste shortUrl
   npm run doors-open     # PATCH targetUrl → DOORS_URL
   # scan / open shortUrl again — new landing, same print
   ```

4. Optional: three seats + ZIP for print

   ```bash
   npm run batch-sample
   ```

## Env

| Variable | Required | Notes |
|----------|----------|--------|
| `QRDOT_API_KEY` | yes | `sk_live_…` |
| `EVENT_ID` | yes | e.g. `summit-2026` |
| `SEAT` | no | default `A12` |
| `PRE_EVENT_URL` | yes | Landing while tickets are “preview” |
| `DOORS_URL` | yes | Landing after `doors-open` |
| `EXPIRE_AT` | no | ISO; default +7 days |
| `MAX_SCANS` | no | default `2` |

Idempotency: `issue` uses key `ticket-<event>-<seat>` so retries are safe for 24h.

## Scripts

| Command | Action |
|---------|--------|
| `npm run issue` | Create QR → `out/ticket.json` |
| `npm run image` | Download PNG + PDF |
| `npm run doors-open` | PATCH destination to `DOORS_URL` |
| `npm run batch-sample` | Batch 3 seats + PNG ZIP |
| `npm test` | Pure helper tests |

## Docs

- Recipes: https://qrdot.dev/docs/recipes/
- Quickstart: https://qrdot.dev/docs/quickstart/
- Node SDK: `npm i @qrdot/sdk`
