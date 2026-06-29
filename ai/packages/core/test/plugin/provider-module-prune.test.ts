import { describe, expect, it } from "bun:test"
import { readdir } from "node:fs/promises"

describe("provider module pruning", () => {
  it("keeps only curated provider modules", async () => {
    const providerDir = new URL("../../src/plugin/provider", import.meta.url)
    const files = (await readdir(providerDir)).filter((file) => file.endsWith(".ts")).sort()

    expect(files).toEqual([
      "anthropic.ts",
      "dynamic.ts",
      "google.ts",
      "openai-auth.ts",
      "openai-compatible.ts",
      "openai.ts",
      "openrouter.ts",
      "talon.ts",
    ])
  })
})
