import { describe, expect, it } from "bun:test"
import {
  shouldBypassVisionPreanalysis,
  shouldUseLightweightTurn,
  type ModelRef,
} from "../../src/session/turn-mode"

describe("turn mode heuristics", () => {
  it("enables lightweight mode for basic greetings", () => {
    const enabled = shouldUseLightweightTurn({
      step: 1,
      text: "Hey",
      hasMediaAttachments: false,
      hasStructuredOutput: false,
      hasSubtasks: false,
    })

    expect(enabled).toBe(true)
  })

  it("enables lightweight mode for simple time/weather questions", () => {
    expect(
      shouldUseLightweightTurn({
        step: 1,
        text: "What time is it?",
        hasMediaAttachments: false,
        hasStructuredOutput: false,
        hasSubtasks: false,
      }),
    ).toBe(true)

    expect(
      shouldUseLightweightTurn({
        step: 1,
        text: "What is the weather?",
        hasMediaAttachments: false,
        hasStructuredOutput: false,
        hasSubtasks: false,
      }),
    ).toBe(true)
  })

  it("does not enable lightweight mode for coding requests", () => {
    const enabled = shouldUseLightweightTurn({
      step: 1,
      text: "Implement a new auth middleware and run tests",
      hasMediaAttachments: false,
      hasStructuredOutput: false,
      hasSubtasks: false,
    })

    expect(enabled).toBe(false)
  })

  it("disables lightweight mode when structured output or media is involved", () => {
    expect(
      shouldUseLightweightTurn({
        step: 1,
        text: "Hey",
        hasMediaAttachments: true,
        hasStructuredOutput: false,
        hasSubtasks: false,
      }),
    ).toBe(false)

    expect(
      shouldUseLightweightTurn({
        step: 1,
        text: "Hey",
        hasMediaAttachments: false,
        hasStructuredOutput: true,
        hasSubtasks: false,
      }),
    ).toBe(false)
  })

  it("bypasses vision preanalysis when coding and vision models match", () => {
    const coding: ModelRef = { providerID: "openai", modelID: "gpt-5" }
    const vision: ModelRef = { providerID: "openai", modelID: "gpt-5" }

    expect(shouldBypassVisionPreanalysis({ codingModel: coding, visionModel: vision })).toBe(true)
  })

  it("keeps vision preanalysis when models differ", () => {
    const coding: ModelRef = { providerID: "openai", modelID: "gpt-5" }
    const vision: ModelRef = { providerID: "anthropic", modelID: "claude-sonnet-4" }

    expect(shouldBypassVisionPreanalysis({ codingModel: coding, visionModel: vision })).toBe(false)
  })
})
