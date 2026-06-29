import type { TuiPlugin, TuiPluginApi } from "@talon-ai/plugin/tui"
import type { BuiltinTuiPlugin } from "../builtins"
import { createSignal, Show } from "solid-js"
import { Locale } from "../../util/locale"

const id = "internal:sidebar-artifacts"

function View(props: { api: TuiPluginApi; session_id: string }) {
  const [open, setOpen] = createSignal(true)
  const theme = () => props.api.theme.current
  const hint = () =>
    `Use /artifact or \`talon artifact\` from the command palette. ${Locale.truncateLeft(props.api.state.path.directory, 60)}`

  return (
    <box>
      <box flexDirection="row" gap={1} onMouseDown={() => setOpen((x) => !x)}>
        <text fg={theme().text}>{open() ? "▼" : "▶"}</text>
        <text fg={theme().text}>
          <b>Artifacts</b>
        </text>
      </box>
      <Show when={open()}>
        <text fg={theme().textMuted} wrapMode="none">
          {hint()}
        </text>
      </Show>
    </box>
  )
}

const tui: TuiPlugin = async (api) => {
  api.slots.register({
    order: 250,
    slots: {
      sidebar_content(_ctx, props) {
        return <View api={api} session_id={props.session_id} />
      },
    },
  })
}

const plugin: BuiltinTuiPlugin = {
  id,
  tui,
}

export default plugin
