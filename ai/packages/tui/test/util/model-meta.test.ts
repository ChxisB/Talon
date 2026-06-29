import { describe, expect, it } from "bun:test"
import { shouldMergeCodingAndVisionRows } from "../../src/component/prompt/model-meta"

describe("prompt model metadata rows", () => {
  it("merges coding/vision rows when both models are the same", () => {
    const merged = shouldMergeCodingAndVisionRows({
      coding: { providerID: "openai", modelID: "gpt-5" },
      vision: { providerID: "openai", modelID: "gpt-5" },
    })

    expect(merged).toBe(true)
  })

  it("keeps separate rows when coding/vision models differ", () => {
    const merged = shouldMergeCodingAndVisionRows({
      coding: { providerID: "openai", modelID: "gpt-5" },
      vision: { providerID: "anthropic", modelID: "claude-sonnet-4" },
    })

    expect(merged).toBe(false)
  })

  it("keeps separate rows when one model is missing", () => {
    expect(
      shouldMergeCodingAndVisionRows({
        coding: undefined,
        vision: { providerID: "openai", modelID: "gpt-5" },
      }),
    ).toBe(false)
  })
})
