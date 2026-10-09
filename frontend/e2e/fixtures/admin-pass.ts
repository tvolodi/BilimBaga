import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
export const ADMIN_PASS_FILE = path.join(__dirname, '..', '..', '.auth', 'admin-pass.txt')

/**
 * Admin password valid for form logins (ISS-160). global-setup may have changed the default
 * password (forced change) and records the effective one in .auth/admin-pass.txt; an explicit
 * E2E_ADMIN_PASS wins, then the recorded value, then the migration default.
 */
export function currentAdminPassword(): string {
  if (process.env.E2E_ADMIN_PASS) return process.env.E2E_ADMIN_PASS
  try {
    const recorded = fs.readFileSync(ADMIN_PASS_FILE, 'utf8').trim()
    if (recorded) return recorded
  } catch {
    /* not written yet */
  }
  return 'Admin1234!'
}
