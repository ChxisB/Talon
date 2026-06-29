import { RGBA, TextAttributes } from "@tui/core"
import { useTheme } from "../context/theme"
import { useDialog } from "../ui/dialog"
import { useSDK } from "../context/sdk"
import { useSync } from "../context/sync"
import { createMemo, For } from "solid-js"
import { DialogModel } from "./dialog-model"
import { useToast } from "../ui/toast"
import { useLocal } from "../context/local"

export function DialogAgentModels(props: { onBack?: () => void; backLabel?: string }) {
  const dialog = useDialog()
  const { theme } = useTheme()
  const sync = useSync()
  const sdk = useSDK()
  const toast = useToast()
  const local = useLocal()

  const subagents = createMemo(() => sync.data.agent.filter((agent) => agent.mode === "subagent" && !agent.hidden))

  const modelLabel = createMemo(() => {
    const labels: Record<string, string> = {}
    for (const agent of subagents()) {
      const m = agent.model
      if (m) {
        const provider = sync.data.provider.find((p) => p.id === m.providerID)
        const modelInfo = provider?.models[m.modelID]
        labels[agent.name] = modelInfo?.name ?? m.modelID
      } else {
        labels[agent.name] = "Not set"
      }
    }
    return labels
  })

  function back() {
    if (props.onBack) props.onBack()
    else dialog.clear()
  }

  function handleAgentClick(agent: (typeof sync.data.agent)[number]) {
    dialog.replace(() => (
      <DialogModel
        current={agent.model}
        onModelSelect={(providerID, modelID) => {
          const value = `${providerID}/${modelID}`
          sdk.client.global.config
            .update({ config: { agent: { [agent.name]: { model: value } } } as any })
            .then(() => sync.bootstrap())
            .then(() => {
              toast.show({
                variant: "info",
                message: `${agent.name} model set to ${value}`,
                duration: 2000,
              })
            })
            .catch((err) => {
              toast.show({
                variant: "warning",
                message: `Failed to set ${agent.name} model: ${err instanceof Error ? err.message : String(err)}`,
                duration: 4000,
              })
            })
        }}
        onBack={() => dialog.replace(() => <DialogAgentModels onBack={props.onBack} backLabel={props.backLabel} />)}
        backLabel="Agent Models"
      />
    ))
  }

  return (
    <box paddingLeft={2} paddingRight={2} gap={1} paddingBottom={1}>
      <box flexDirection="row" justifyContent="space-between">
        <text fg={theme.text} attributes={TextAttributes.BOLD}>
          Agent Models
        </text>
        <text fg={theme.textMuted} onMouseUp={back}>
          {props.backLabel ?? "esc"}
        </text>
      </box>
      <For each={subagents()}>
        {(agent) => (
          <box
            flexDirection="row"
            gap={1}
            paddingLeft={1}
            paddingRight={1}
            paddingTop={0}
            paddingBottom={0}
            onMouseUp={() => handleAgentClick(agent)}
          >
            <text flexShrink={0} fg={theme.primary} attributes={TextAttributes.BOLD} width={1}>
              ▸
            </text>
            <text fg={local.agent.color(agent.name)} flexGrow={1}>
              {agent.name}
            </text>
            <text fg={theme.textMuted}>{modelLabel()[agent.name]}</text>
          </box>
        )}
      </For>
    </box>
  )
}
