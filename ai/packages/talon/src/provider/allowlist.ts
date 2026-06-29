const PROVIDER_ALIAS: Record<string, string> = {
  zen: "talon",
}

export const DEFAULT_ENABLED_PROVIDER_IDS = [
  "google",
  "openai",
  "anthropic",
  "openrouter",
  "opencode-go",
  "talon",
] as const

export function normalizeProviderID(input: string): string {
  const value = input.trim().toLowerCase()
  return PROVIDER_ALIAS[value] ?? value
}

/**
 * When enabled_providers is explicitly configured we honor it exactly.
 * Otherwise we apply Talon's curated default provider set.
 */
export function resolveEnabledProviders(input: string[] | undefined): Set<string> {
  if (input && input.length > 0) {
    return new Set(input.map(normalizeProviderID))
  }

  return new Set(DEFAULT_ENABLED_PROVIDER_IDS)
}
