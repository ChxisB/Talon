import { describe, expect, test } from "bun:test"
import {
  buildOllamaPresetConfigPatch,
  mergeEnabledProvidersWithOllama,
  providerSetupChoices,
} from "../../src/component/provider-setup"

describe("provider setup choices", () => {
  test("offers local, provider, and skip setup choices", () => {
    expect(providerSetupChoices.map((item) => item.value)).toEqual(["local", "provider", "none"])
  })
})

describe("Ollama preset config patch", () => {
  test("builds an OpenAI-compatible local ollama provider with qwen presets", () => {
    const patch = buildOllamaPresetConfigPatch({})
    expect(patch).toMatchObject({
      config: {
        provider: {
          ollama: {
            name: "Ollama",
            npm: "@ai-sdk/openai-compatible",
            api: "http://127.0.0.1:11434/v1",
          },
        },
        model: "ollama/qwen2.5-coder:7b",
        vision_model: "ollama/qwen2.5-vl:7b",
      },
    })

    const models = (patch.config.provider as Record<string, any>).ollama.models as Record<string, unknown>
    expect(models["qwen2.5-coder:7b"]).toBeDefined()
    expect(models["qwen2.5-vl:7b"]).toBeDefined()
  })

  test("preserves explicit enabled_providers and appends ollama once", () => {
    expect(mergeEnabledProvidersWithOllama(["openai", "anthropic"]))
      .toEqual(["openai", "anthropic", "ollama"])

    expect(mergeEnabledProvidersWithOllama(["openai", "ollama"])).toEqual(["openai", "ollama"])
  })

  test("leaves implicit provider defaults untouched", () => {
    expect(mergeEnabledProvidersWithOllama(undefined)).toBeUndefined()
    expect(mergeEnabledProvidersWithOllama([])).toBeUndefined()
  })
})
