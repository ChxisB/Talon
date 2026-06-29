import { RGBA, TextAttributes } from "@tui/core"
import { useTheme } from "../context/theme"
import { useDialog } from "../ui/dialog"
import { useSDK } from "../context/sdk"
import { useKV } from "../context/kv"
import { useSync } from "../context/sync"
import { createMemo, For } from "solid-js"
import { DialogModel } from "./dialog-model"
import { DialogAgentModels } from "./dialog-agent-models"
import { DialogProvider } from "./dialog-provider"
import { DialogThemeList } from "./dialog-theme-list"
import { DialogAgent } from "./dialog-agent"
import { DialogMcp } from "./dialog-mcp"
import { DialogStatus } from "./dialog-status"
import { DialogHelp } from "../ui/dialog-help"
import { DialogPrompt } from "../ui/dialog-prompt"
import { useToast } from "../ui/toast"
import { useLocal } from "../context/local"
import open from "open"

const COMMON_PROVIDERS: { id: string; name: string; keyLabel: string }[] = [
  { id: "anthropic", name: "Anthropic", keyLabel: "sk-ant-..." },
  { id: "openai", name: "OpenAI", keyLabel: "sk-..." },
  { id: "openrouter", name: "OpenRouter", keyLabel: "sk-or-..." },
  { id: "google", name: "Google Gemini", keyLabel: "AI..." },
  { id: "groq", name: "Groq", keyLabel: "gsk_..." },
]

type ToggleSetting = {
  type: "toggle"
  label: string
  key: string
  defaultValue: boolean
  onToggle: (value: boolean) => void
}

type NavigateSetting = {
  type: "navigate"
  label: string
  trailing?: string
  onClick: () => void
}

type SettingItem = ToggleSetting | NavigateSetting

type SettingsSection = {
  title: string
  items: SettingItem[]
}

function SettingRow(props: {
  item: SettingItem
  fg: RGBA
  fgMuted: RGBA
  accent: RGBA
}) {
  const kv = useKV()

  function handleClick() {
    if (props.item.type === "toggle") {
      props.item.onToggle(!kv.get(props.item.key, props.item.defaultValue))
    } else {
      props.item.onClick()
    }
  }

  return (
    <box
      flexDirection="row"
      gap={1}
      paddingLeft={1}
      paddingRight={1}
      paddingTop={0}
      paddingBottom={0}
      onMouseUp={handleClick}
    >
      <text
        flexShrink={0}
        fg={props.accent}
        attributes={TextAttributes.BOLD}
        width={1}
      >
        {props.item.type === "toggle" ? "\u25C9" : "\u25B8"}
      </text>
      <text fg={props.fg} flexGrow={1}>
        {props.item.label}
      </text>
      <text fg={props.fgMuted}>
        {props.item.type === "toggle"
          ? kv.get(props.item.key, props.item.defaultValue)
            ? "On"
            : "Off"
          : ""}
      </text>
    </box>
  )
}

export function DialogSettings() {
  const dialog = useDialog()
  const { theme } = useTheme()
  const kv = useKV()
  const sync = useSync()
  const sdk = useSDK()
  const toast = useToast()
  const local = useLocal()

  const currentModelLabel = createMemo(() => {
    const parsed = local.model.parsed()
    return `${parsed.provider} / ${parsed.model}`
  })

  const visionModelLabel = createMemo(() => {
    // Read from sync data (merged config including global), with fallback from local KV
    const raw = (sync.data.config as Record<string, unknown>).vision_model as string | undefined
    const fallback = kv.get("model_image") as string | undefined
    const value = raw ?? fallback
    if (!value) return "Not set"
    const parts = value.split("/")
    if (parts.length < 2) return value
    const providerID = parts[0]
    const modelID = parts.slice(1).join("/")
    const provider = sync.data.provider.find((p) => p.id === providerID)
    const modelInfo = provider?.models[modelID]
    return modelInfo?.name ?? modelID
  })

  const connectedProviders = createMemo(() => sync.data.provider_next.connected)

  function isConnected(providerID: string) {
    return connectedProviders().includes(providerID)
  }

  async function promptApiKey(providerID: string, providerName: string, placeholder: string) {
    const value = await DialogPrompt.show(dialog, `${providerName} API Key`, {
      placeholder,
      description: () => (
        <text fg={theme.textMuted}>
          Enter your {providerName} API key. This will be stored securely.
        </text>
      ),
    })
    if (!value) return

    await sdk.client.auth.set({
      providerID,
      auth: { type: "api", key: value },
    })
    await sdk.client.instance.dispose()
    await sync.bootstrap()
    toast.show({
      variant: "info",
      message: `${providerName} API key saved`,
    })
  }

  function back() {
    dialog.replace(() => <DialogSettings />)
  }

  const sections: SettingsSection[] = [
    {
      title: "API Keys",
      items: [
        ...COMMON_PROVIDERS.map((p) => ({
          type: "navigate" as const,
          label: p.name,
          trailing: isConnected(p.id) ? "Connected" : "Not set",
          onClick: () => promptApiKey(p.id, p.name, p.keyLabel),
        })),
        {
          type: "navigate",
          label: "Other providers...",
          onClick: () => dialog.replace(() => <DialogProvider onBack={back} backLabel="Settings" />),
        },
      ],
    },
    {
      title: "Models",
      items: [
        {
            type: "navigate",
            label: "Coding",
            trailing: currentModelLabel(),
            onClick: () => {
              const codingCurrent = local.model.current()
              dialog.replace(() => (
                <DialogModel
                  current={codingCurrent}
                  onModelSelect={(providerID, modelID) => {
                    local.model.set({ providerID, modelID }, { recent: true })
                    const value = `${providerID}/${modelID}`
                    sdk.client.global.config
                      .update({ config: { model: value } as any })
                      .then(() => sync.bootstrap())
                      .then(() => {
                        toast.show({
                          variant: "info",
                          message: `Coding model set to ${value}`,
                          duration: 2000,
                        })
                      })
                      .catch((err) => {
                        toast.show({
                          variant: "warning",
                          message: `Failed to set coding model: ${err instanceof Error ? err.message : String(err)}`,
                          duration: 4000,
                        })
                      })
                  }}
                  onBack={back}
                  backLabel="Settings"
                />
              ))
            },
        },
        {
          type: "navigate",
          label: "Vision",
          trailing: visionModelLabel(),
          onClick: () => {
            const raw = (sync.data.config as Record<string, unknown>).vision_model as string | undefined
            const fallback = kv.get("model_image") as string | undefined
            const visionModelValue = raw ?? fallback
            const visionCurrent = visionModelValue
              ? (() => {
                  const parts = visionModelValue.split("/")
                  if (parts.length < 2) return undefined
                  return { providerID: parts[0], modelID: parts.slice(1).join("/") }
                })()
              : undefined
            dialog.replace(
              () => (
                <DialogModel
                  current={visionCurrent}
                      onModelSelect={(providerID, modelID) => {
                        const value = `${providerID}/${modelID}`
                        sdk.client.global.config
                          .update({ config: { vision_model: value } as any })
                        .then(() => sync.bootstrap())
                        .then(() => {
                          toast.show({ variant: "info", message: `Vision model set to ${value}`, duration: 2000 })
                        })
                        .catch((err) => {
                          toast.show({
                            variant: "warning",
                            message: `Failed to set vision model: ${err instanceof Error ? err.message : String(err)}`,
                            duration: 4000,
                          })
                        })
                    }}
                  onBack={back}
                  backLabel="Settings"
                />
              ),
            )
          },
        },
        {
          type: "navigate",
          label: "Agent Models",
          trailing: "Configure per-agent",
          onClick: () => {
            dialog.replace(() => <DialogAgentModels onBack={back} backLabel="Settings" />)
          },
        },
      ],
    },
    {
      title: "General",
      items: [
        {
          type: "toggle",
          label: "Animations",
          key: "animations_enabled",
          defaultValue: true,
          onToggle: (value) => kv.set("animations_enabled", value),
        },
        {
          type: "toggle",
          label: "File Context",
          key: "file_context_enabled",
          defaultValue: true,
          onToggle: (value) => kv.set("file_context_enabled", value),
        },
        {
          type: "toggle",
          label: "Paste Summary",
          key: "paste_summary_enabled",
          defaultValue: !sync.data.config.experimental?.disable_paste_summary,
          onToggle: (value) => kv.set("paste_summary_enabled", value),
        },
        {
          type: "toggle",
          label: "Terminal Title",
          key: "terminal_title_enabled",
          defaultValue: true,
          onToggle: (value) => kv.set("terminal_title_enabled", value),
        },
        {
          type: "toggle",
          label: "Session Directory Filter",
          key: "session_directory_filter_enabled",
          defaultValue: true,
          onToggle: (value) => {
            kv.set("session_directory_filter_enabled", value)
            void sync.session.refresh()
          },
        },
      ],
    },
    {
      title: "Theme",
      items: [
        {
          type: "navigate",
          label: "Switch Theme",
          onClick: () => dialog.replace(() => <DialogThemeList />),
        },
      ],
    },
    {
      title: "System",
      items: [
        {
          type: "navigate",
          label: "Status",
          onClick: () => dialog.replace(() => <DialogStatus />),
        },
        {
          type: "navigate",
          label: "MCP Servers",
          onClick: () => dialog.replace(() => <DialogMcp />),
        },
        {
          type: "navigate",
          label: "Help",
          onClick: () => dialog.replace(() => <DialogHelp />),
        },
        {
          type: "navigate",
          label: "Open Docs",
          onClick: () => {
            dialog.clear()
            open("https://talon.ai/docs").catch(() => {})
          },
        },
      ],
    },
  ]

  return (
    <box paddingLeft={2} paddingRight={2} gap={1} paddingBottom={1}>
      <box flexDirection="row" justifyContent="space-between">
        <text fg={theme.text} attributes={TextAttributes.BOLD}>
          Settings
        </text>
        <text fg={theme.textMuted} onMouseUp={() => dialog.clear()}>
          esc
        </text>
      </box>
      <For each={sections}>
        {(section) => (
          <box gap={0}>
            <text fg={theme.textMuted} attributes={TextAttributes.BOLD}>
              {section.title}
            </text>
            <For each={section.items}>
              {(item) => (
                <SettingRow
                  item={item}
                  fg={theme.text}
                  fgMuted={theme.textMuted}
                  accent={theme.primary}
                />
              )}
            </For>
          </box>
        )}
      </For>
    </box>
  )
}
