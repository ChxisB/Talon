import { describe, expect, it } from "bun:test"

describe("ProviderPlugins allowlist", () => {
  it("registers only curated providers", async () => {
    const { ProviderPlugins } = await import(new URL("../../src/plugin/provider.ts", import.meta.url).href)
    const ids: string[] = []
    for (const item of ProviderPlugins as Array<{ id: unknown }>) {
      ids.push(String(item.id))
    }

    expect(ids).toEqual([
      "anthropic",
      "google",
      "openai-compatible",
      "openai",
      "openrouter",
      "talon",
      "dynamic-provider",
    ])
  })
})
