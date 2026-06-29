import { describe, expect, it } from "bun:test"
import {
  DEFAULT_ENABLED_PROVIDER_IDS,
  normalizeProviderID,
  resolveEnabledProviders,
} from "../../src/provider/allowlist"

describe("provider allowlist", () => {
  it("defaults to the curated provider set when config is unset", () => {
    const enabled = resolveEnabledProviders(undefined)

    expect(enabled.has("google")).toBe(true)
    expect(enabled.has("openai")).toBe(true)
    expect(enabled.has("anthropic")).toBe(true)
    expect(enabled.has("openrouter")).toBe(true)
    expect(enabled.has("opencode-go")).toBe(true)
    expect(enabled.has("talon")).toBe(true)

    expect(enabled.has("deepseek")).toBe(false)
    expect(enabled.has("mistral")).toBe(false)
  })

  it("honors explicit enabled_providers override", () => {
    const enabled = resolveEnabledProviders(["openai", "deepseek"])
    expect(enabled).toEqual(new Set(["openai", "deepseek"]))
  })

  it("normalizes zen alias to talon provider", () => {
    expect(normalizeProviderID("zen")).toBe("talon")
    expect(DEFAULT_ENABLED_PROVIDER_IDS.includes("talon")).toBe(true)
  })
})
