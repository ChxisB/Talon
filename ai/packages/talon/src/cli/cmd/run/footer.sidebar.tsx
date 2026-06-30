/** @jsxImportSource @tui/solid */
// Subagent sidebar — shows active subagents, their status, and costs.
// Renders as a right-side panel within the footer routing system.

import { Show, createMemo, type Accessor } from "solid-js"
import type { FooterSubagentTab, FooterSubagentState } from "./types"
import type { RunFooterTheme, RunTheme } from "./theme"
import type { ColorInput } from "@tui/core"

/** Width of the sidebar panel in terminal columns */
export const SUBAGENT_SIDEBAR_WIDTH = 36

/** Compact subagent entry with cost display */
interface SubagentEntryProps {
  tab: FooterSubagentTab
  theme: RunFooterTheme
  accentColor: ColorInput
}

function SubagentEntry(props: SubagentEntryProps) {
  const { tab, theme, accentColor } = props
  const statusIcon = () => {
    switch (tab.status) {
      case "running":
        return "◔"
      case "completed":
        return "●"
      case "cancelled":
        return "○"
      case "error":
        return "◍"
    }
  }

  const costLabel = () => {
    if (tab.cost !== undefined && tab.cost > 0) {
      return `$${tab.cost.toFixed(4)}`
    }
    if (tab.inputTokens !== undefined || tab.outputTokens !== undefined) {
      const input = tab.inputTokens ?? 0
      const output = tab.outputTokens ?? 0
      return `${input}i/${output}o`
    }
    return undefined
  }

  const callsLabel = () => {
    if (tab.toolCalls !== undefined && tab.toolCalls > 0) {
      return `${tab.toolCalls} tc`
    }
    return undefined
  }

  const descriptionTrimmed = () => {
    const desc = tab.description
    return desc.length > 26 ? desc.slice(0, 24) + ".." : desc
  }

  return (
    <box flexDirection="row" gap={1} paddingLeft={1} paddingRight={1} flexShrink={0}>
      {/* Status icon */}
      <text
        fg={tab.status === "running" ? accentColor : tab.status === "error" ? theme.error : theme.muted}
        wrapMode="none"
        flexShrink={0}
      >
        {statusIcon()}
      </text>
      {/* Label + description */}
      <box flexDirection="column" gap={0} flexGrow={1} flexShrink={1}>
        <text fg={tab.status === "running" ? theme.text : theme.muted} wrapMode="none" truncate flexShrink={1}>
          {tab.label}
        </text>
        <text fg={theme.muted} wrapMode="none" truncate flexShrink={1}>
          {descriptionTrimmed()}
        </text>
      </box>
      {/* Cost + calls */}
      <box flexDirection="column" gap={0} flexShrink={0}>
        <Show when={costLabel() !== undefined}>
          {(label) => (
            <text fg={theme.highlight} wrapMode="none" flexShrink={0}>
              {label()}
            </text>
          )}
        </Show>
        <Show when={callsLabel() !== undefined}>
          {(label) => (
            <text fg={theme.muted} wrapMode="none" flexShrink={0}>
              {label()}
            </text>
          )}
        </Show>
      </box>
    </box>
  )
}

export function RunSubagentSidebar(props: {
  subagent: Accessor<FooterSubagentState | undefined>
  theme: Accessor<RunTheme>
  accentColor: ColorInput
}) {
  const theme = createMemo(() => props.theme())
  const footer = createMemo(() => theme().footer)
  const subagent = createMemo(() => props.subagent())

  const tabs = createMemo(() => {
    const s = subagent()
    if (!s) return [] as FooterSubagentTab[]
    // Sort: running first, then by lastUpdatedAt desc
    return [...s.tabs].sort((a, b) => {
      const aActive = a.status === "running" ? 1 : 0
      const bActive = b.status === "running" ? 1 : 0
      if (aActive !== bActive) return bActive - aActive
      return b.lastUpdatedAt - a.lastUpdatedAt
    })
  })

  // Compute total cost across all tabs
  const totalCost = createMemo(() => {
    return tabs().reduce((sum, t) => sum + (t.cost ?? 0), 0)
  })

  const runningCount = createMemo(() => tabs().filter((t) => t.status === "running").length)
  const completedCount = createMemo(() => tabs().filter((t) => t.status === "completed").length)

  return (
    <box
      width={SUBAGENT_SIDEBAR_WIDTH}
      height="100%"
      flexDirection="column"
      backgroundColor={footer().surface}
      borderColor={footer().line}
      flexShrink={0}
    >
      {/* Header */}
      <box
        flexDirection="row"
        gap={1}
        paddingLeft={1}
        paddingRight={1}
        paddingTop={1}
        paddingBottom={1}
        flexShrink={0}
        backgroundColor={footer().surface}
      >
        <text fg={footer().text} wrapMode="none" flexShrink={0}>
          <span style={{ bold: true }}>Subagents</span>
        </text>
        <Show when={runningCount() > 0 || completedCount() > 0}>
          <text fg={footer().muted} wrapMode="none" flexShrink={0}>
            {runningCount() > 0 ? `${runningCount()} active` : ""}
            {runningCount() > 0 && completedCount() > 0 ? " \u00b7 " : ""}
            {completedCount() > 0 ? `${completedCount()} done` : ""}
          </text>
        </Show>
      </box>

      {/* Separator */}
      <box height={1} flexShrink={0}>
        <text fg={footer().line} wrapMode="none">
          {"\u2500".repeat(SUBAGENT_SIDEBAR_WIDTH - 2)}
        </text>
      </box>

      {/* Subagent list */}
      <box flexDirection="column" gap={0} flexGrow={1} flexShrink={1}>
        <Show
          when={tabs().length > 0}
          fallback={
            <box paddingLeft={1} paddingRight={1} flexShrink={0}>
              <text fg={footer().muted} wrapMode="none">
                No subagent activity yet
              </text>
            </box>
          }
        >
          {tabs().map((tab) => (
            <SubagentEntry tab={tab} theme={footer()} accentColor={props.accentColor} />
          ))}
        </Show>
      </box>

      {/* Footer with total cost */}
      <Show when={totalCost() > 0}>
        <box height={1} flexShrink={0}>
          <text fg={footer().line} wrapMode="none">
            {"\u2500".repeat(SUBAGENT_SIDEBAR_WIDTH - 2)}
          </text>
        </box>
        <box flexDirection="row" gap={1} paddingLeft={1} paddingRight={1} paddingBottom={1} flexShrink={0}>
          <text fg={footer().muted} wrapMode="none" flexShrink={0}>
            Total cost
          </text>
          <text fg={footer().highlight} wrapMode="none" flexShrink={0}>
            ${totalCost().toFixed(4)}
          </text>
        </box>
      </Show>
    </box>
  )
}
