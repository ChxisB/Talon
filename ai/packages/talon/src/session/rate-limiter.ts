import { Effect } from "effect"

/**
 * Fixed-window counter for proactive rate limiting.
 *
 * Uses 60-second windows aligned to the clock minute (UTC).
 * - Each window tracks how many requests have been made.
 * - If the limit is reached, returns the ms until the next window.
 * - Simple, predictable, and race-condition-free.
 */

interface Bucket {
  /** Window key = Math.floor(now / 60_000) */
  readonly key: number
  /** Request count in the current window */
  count: number
}

const windows = new Map<string, Bucket>()

/**
 * Check whether a request for `providerId` can proceed immediately.
 *
 * @returns 0 if within the rate limit, or the number of ms to wait
 *          until the next window opens.
 */
export function check(providerId: string, requestsPerMinute: number): Effect.Effect<number> {
  return Effect.sync(() => {
    const now = Date.now()
    const windowKey = Math.floor(now / 60_000)

    let bucket = windows.get(providerId)

    // First request or new window — reset
    if (!bucket || bucket.key !== windowKey) {
      bucket = { key: windowKey, count: 1 }
      windows.set(providerId, bucket)
      return 0
    }

    // At the limit — tell caller how long to wait
    if (bucket.count >= requestsPerMinute) {
      const nextWindow = (windowKey + 1) * 60_000
      return nextWindow - now
    }

    // Within limit — count this request and proceed
    bucket.count++
    return 0
  })
}

/**
 * Reset the rate limiter for `providerId`. Useful during testing.
 */
export function reset(providerId: string): Effect.Effect<void> {
  return Effect.sync(() => {
    windows.delete(providerId)
  })
}

/**
 * Reset all rate limiter state. Useful during testing.
 */
export function resetAll(): void {
  windows.clear()
}
