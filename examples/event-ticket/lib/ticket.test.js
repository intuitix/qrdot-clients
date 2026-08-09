import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  defaultExpireAt,
  seatIdempotencyKey,
  ticketMetadata,
} from "./ticket.js";

describe("seatIdempotencyKey", () => {
  it("normalizes event and seat", () => {
    assert.equal(
      seatIdempotencyKey("Summit 2026", "a12"),
      "ticket-summit-2026-A12",
    );
  });
});

describe("ticketMetadata", () => {
  it("uppercases seat", () => {
    assert.deepEqual(ticketMetadata("summit-2026", "a12"), {
      event_id: "summit-2026",
      seat: "A12",
    });
  });
});

describe("defaultExpireAt", () => {
  it("returns ISO string in the future", () => {
    const iso = defaultExpireAt(7);
    assert.ok(Date.parse(iso) > Date.now());
  });
});
