import { readFileSync } from 'fs'
import { join, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))

type JsonValue = string | number | boolean | null | JsonValue[] | Record<string, JsonValue>

function flattenKeys(obj: Record<string, JsonValue>, prefix = ''): Set<string> {
  const keys = new Set<string>()
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${k}` : k
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
      for (const nested of flattenKeys(v as Record<string, JsonValue>, fullKey)) {
        keys.add(nested)
      }
    } else {
      keys.add(fullKey)
    }
  }
  return keys
}

const localeDir = join(__dirname, '..', 'src', 'locales')

const localeNames = ['kk', 'ru', 'en'] as const
const keysets: Record<string, Set<string>> = {}

for (const name of localeNames) {
  const raw = readFileSync(join(localeDir, `${name}.json`), 'utf-8')
  const data = JSON.parse(raw) as Record<string, JsonValue>
  keysets[name] = flattenKeys(data)
}

let failed = false
const allKeys = new Set([...keysets.kk, ...keysets.ru, ...keysets.en])

for (const key of allKeys) {
  const missing = localeNames.filter((name) => !keysets[name].has(key))
  if (missing.length > 0) {
    console.error(`MISSING key "${key}" in: ${missing.join(', ')}`)
    failed = true
  }
}

if (failed) {
  console.error('\nLocale key parity check FAILED')
  process.exit(1)
} else {
  console.log(`All ${allKeys.size} locale keys are present in all 3 locales. ✓`)
}
