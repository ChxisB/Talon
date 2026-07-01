import { createMemo, Show } from "solid-js"
import { useTheme } from "../../context/theme"
import { useSync } from "../../context/sync"
import { useRouteData } from "../../context/route"
import { useLocal } from "../../context/local"
import { Spinner } from "../../component/spinner"
import * as Model from "../../util/model"

export function SubagentStatus() {
  const routeData = useRouteData("session")
  const sync = useSync()
  const local = useLocal()
  const { theme } = useTheme()

  const sessionID = createMemo(() => routeData.sessionID)

  const busyChild = createMemo(() => {
    const id = sessionID()
    if (!id) return undefined
    return sync.data.session.find(
      (s) => s.parentID === id && sync.data.session_status[s.id]?.type === "busy",
    )
  })

  const agentName = createMemo(() => busyChild()?.agent)

  const agentColor = createMemo(() => {
    const name = agentName()
    return name ? local.agent.color(name) : undefined
  })

  const modelLabel = createMemo(() => {
    const name = agentName()
    if (!name) return ""
    const agent = sync.data.agent.find((a) => a.name === name)
    if (!agent) return ""
    const m = agent.model
    if (!m) return ""
    const modelName = Model.name(sync.data.provider, m.providerID, m.modelID)
    const providerName = sync.data.provider.find((p) => p.id === m.providerID)?.name ?? m.providerID
    return `${modelName} (${providerName})`
  })

  return (
    <Show when={busyChild() && agentName()}>
      <box paddingTop={1} flexShrink={0}>
        <Spinner color={agentColor() ?? theme.textMuted}>
          @{agentName()}{modelLabel() ? ` · ${modelLabel()}` : ""}
        </Spinner>
      </box>
    </Show>
  )
}
