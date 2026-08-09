import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

/** Load `.env` from cwd into process.env (does not override existing vars). */
export function loadDotEnv() {
  const path = resolve(process.cwd(), ".env");
  if (!existsSync(path)) return;
  for (const line of readFileSync(path, "utf8").split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eq = trimmed.indexOf("=");
    if (eq < 1) continue;
    const key = trimmed.slice(0, eq).trim();
    let value = trimmed.slice(eq + 1).trim();
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }
    if (process.env[key] === undefined) process.env[key] = value;
  }
}

export function requireEnv(name) {
  const v = process.env[name]?.trim();
  if (!v) {
    console.error(`Missing ${name}. Copy .env.example → .env and set your key.`);
    process.exit(1);
  }
  return v;
}

export function optionalEnv(name, fallback) {
  const v = process.env[name]?.trim();
  return v || fallback;
}
