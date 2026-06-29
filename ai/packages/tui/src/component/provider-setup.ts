export const OLLAMA_PROVIDER_ID = "ollama"
export const OLLAMA_LOCAL_OPTION_VALUE = "__talon_local_ollama__"
export const OLLAMA_CODING_MODEL = "qwen2.5-coder:7b"
export const OLLAMA_VISION_MODEL = "qwen2.5-vl:7b"

export const providerSetupChoices = [
  {
    value: "local",
    title: "Use local models (Ollama)",
    description: "Configure Ollama preset with Qwen coding + vision models",
  },
  {
    value: "provider",
    title: "Connect another provider",
    description: "Use API-key or OAuth providers",
  },
  {
    value: "none",
    title: "None for now",
    description: "Skip setup and keep using Talon defaults",
  },
] as const

type SetupConfig = {
  enabled_providers?: string[]
}

function asSetupConfig(input: unknown): SetupConfig {
  if (!input || typeof input !== "object") return {}
  const value = input as Record<string, unknown>
  const enabled = value.enabled_providers
  if (!Array.isArray(enabled) || enabled.length === 0) return {}
  return {
    enabled_providers: enabled.filter((item): item is string => typeof item === "string"),
  }
}

export function mergeEnabledProvidersWithOllama(input: string[] | undefined): string[] | undefined {
  if (!input || input.length === 0) return
  if (input.includes(OLLAMA_PROVIDER_ID)) return input
  return [...input, OLLAMA_PROVIDER_ID]
}

export function buildOllamaPresetConfigPatch(currentConfig: unknown) {
  const current = asSetupConfig(currentConfig)
  const enabledProviders = mergeEnabledProvidersWithOllama(current.enabled_providers)

  const config: Record<string, unknown> = {
    provider: {
      [OLLAMA_PROVIDER_ID]: {
        name: "Ollama",
        npm: "@ai-sdk/openai-compatible",
        api: "http://127.0.0.1:11434/v1",
        models: {
          [OLLAMA_CODING_MODEL]: {
            name: "Qwen2.5 Coder 7B (local)",
            modalities: {
              input: ["text"],
              output: ["text"],
            },
          },
          [OLLAMA_VISION_MODEL]: {
            name: "Qwen2.5 VL 7B (local)",
            modalities: {
              input: ["text", "image"],
              output: ["text"],
            },
          },
        },
      },
    },
    model: `${OLLAMA_PROVIDER_ID}/${OLLAMA_CODING_MODEL}`,
    vision_model: `${OLLAMA_PROVIDER_ID}/${OLLAMA_VISION_MODEL}`,
  }

  if (enabledProviders) config.enabled_providers = enabledProviders

  return { config }
}
