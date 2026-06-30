import { createMemo, Match, onCleanup, onMount, Show, Switch } from "solid-js"
import { useTheme } from "../../context/theme"
import { useSync } from "../../context/sync"
import { useDirectory } from "../../context/directory"
import { useConnected } from "../../component/use-connected"
import { createStore } from "solid-js/store"
import { useRoute, useRouteData } from "../../context/route"
import { useLocal } from "../../context/local"

export function Footer() {
  const { theme } = useTheme()
  const sync = useSync()
  const route = useRoute()
  const local = useLocal()
  const mcp = createMemo(() => Object.values(sync.data.mcp).filter((x) => x.status === "connected").length)
  const mcpError = createMemo(() => Object.values(sync.data.mcp).some((x) => x.status === "failed"))
  const lsp = createMemo(() => Object.keys(sync.data.lsp))
  const permissions = createMemo(() => {
    if (route.data.type !== "session") return []
    return sync.data.permission[route.data.sessionID] ?? []
  })
  const directory = useDirectory()
  const connected = useConnected()
  const agentName = createMemo(() => local.agent.current()?.name)
  const agentColor = createMemo(() => {
    const name = agentName()
    return name ? local.agent.color(name) : theme.textMuted
  })
  const modelInfo = createMemo(() => local.model.parsed())
  const modelVariant = createMemo(() => local.model.variant.current())

  const routeData = useRouteData("session")
  const activeSubagentName = createMemo(() => {
    const sessionID = routeData.sessionID
    if (!sessionID) return undefined

    // 1. Check for busy child sessions — covers pipeline() orchestration
    //    (which creates child sessions directly without task tool parts)
    const busyChild = sync.data.session.find(
      (s) => s.parentID === sessionID && sync.data.session_status[s.id]?.type === "busy",
    )
    if (busyChild?.agent) return busyChild.agent

    // 2. Fallback: check task tool parts — covers manual task() calls
    const messages = sync.data.message[sessionID] ?? []
    for (const msg of messages) {
      const parts = sync.data.part[msg.id] ?? []
      for (const part of parts) {
        if (part.type === "tool" && part.tool === "task" && part.state.status === "running") {
          const subagentType = (part.state as Record<string, unknown>).input as
            | Record<string, unknown>
            | undefined
          if (subagentType?.subagent_type && typeof subagentType.subagent_type === "string") {
            return subagentType.subagent_type
          }
        }
      }
    }

    return undefined
  })
  const activeSubagentColor = createMemo(() => {
    const name = activeSubagentName()
    return name ? local.agent.color(name) : undefined
  })

  const [store, setStore] = createStore({
    welcome: false,
  })

  onMount(() => {
    // Track all timeouts to ensure proper cleanup
    const timeouts: ReturnType<typeof setTimeout>[] = []

    function tick() {
      if (connected()) return
      if (!store.welcome) {
        setStore("welcome", true)
        timeouts.push(setTimeout(() => tick(), 5000))
        return
      }

      if (store.welcome) {
        setStore("welcome", false)
        timeouts.push(setTimeout(() => tick(), 10_000))
        return
      }
    }
    timeouts.push(setTimeout(() => tick(), 10_000))

    onCleanup(() => {
      timeouts.forEach(clearTimeout)
    })
  })

  return (
    <box flexDirection="row" justifyContent="space-between" gap={1} flexShrink={0}>
      <box flexDirection="row" gap={1} flexShrink={0}>
        <text fg={theme.textMuted}>{directory()}</text>
        <Show when={agentName()}>
          <text fg={theme.textMuted}>
            <span style={{ fg: agentColor() }}>@{agentName()}</span>
            <Show when={activeSubagentName()}>
              {(name) => (
                <span>
                  <span> → </span>
                  <span style={{ fg: activeSubagentColor() }}>@{name()}</span>
                </span>
              )}
            </Show>
            <Show when={modelInfo().model !== "No provider selected"}>
              <span> · {modelInfo().model}</span>
              <span style={{ fg: theme.textMuted }}> ({modelInfo().provider})</span>
              <Show when={modelVariant()}>
                {(variant) => <span> · {variant()}</span>}
              </Show>
            </Show>
          </text>
        </Show>
      </box>
      <box gap={2} flexDirection="row" flexShrink={0}>
        <Switch>
          <Match when={store.welcome}>
            <text fg={theme.text}>
              Get started <span style={{ fg: theme.textMuted }}>/connect</span>
            </text>
          </Match>
          <Match when={connected()}>
            <Show when={permissions().length > 0}>
              <text fg={theme.warning}>
                <span style={{ fg: theme.warning }}>△</span> {permissions().length} Permission
                {permissions().length > 1 ? "s" : ""}
              </text>
            </Show>
            <text fg={theme.text}>
              <span style={{ fg: lsp().length > 0 ? theme.success : theme.textMuted }}>•</span> {lsp().length} LSP
            </text>
            <Show when={mcp()}>
              <text fg={theme.text}>
                <Switch>
                  <Match when={mcpError()}>
                    <span style={{ fg: theme.error }}>⊙ </span>
                  </Match>
                  <Match when={true}>
                    <span style={{ fg: theme.success }}>⊙ </span>
                  </Match>
                </Switch>
                {mcp()} MCP
              </text>
            </Show>
            <text fg={theme.textMuted}>/status</text>
          </Match>
        </Switch>
      </box>
    </box>
  )
}
