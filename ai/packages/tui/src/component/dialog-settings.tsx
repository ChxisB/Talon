import { RGBA, TextAttributes } from "@tui/core"
import { useTheme } from "../context/theme"
import { useDialog } from "../ui/dialog"
import { useKV } from "../context/kv"
import { useSync } from "../context/sync"
import { For } from "solid-js"
import { DialogModel } from "./dialog-model"
import { DialogProvider } from "./dialog-provider"
import { DialogThemeList } from "./dialog-theme-list"
import { DialogAgent } from "./dialog-agent"
import { DialogMcp } from "./dialog-mcp"
import { DialogStatus } from "./dialog-status"
import { DialogHelp } from "../ui/dialog-help"
import open from "open"

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

  const sections: SettingsSection[] = [
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
      title: "Provider",
      items: [
        {
          type: "navigate",
          label: "Connect Provider",
          onClick: () => dialog.replace(() => <DialogProvider />),
        },
        {
          type: "navigate",
          label: "Switch Model",
          onClick: () => dialog.replace(() => <DialogModel />),
        },
        {
          type: "navigate",
          label: "Switch Agent",
          onClick: () => dialog.replace(() => <DialogAgent />),
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
