import type { Event } from "@talon-ai/sdk/v2"
import type { TuiAttentionSoundName, TuiPlugin, TuiPluginApi } from "@talon-ai/plugin/tui"
import type { BuiltinTuiPlugin } from "../builtins"

const id = "internal:notifications"

type SessionError = Extract<Event, { type: "session.error" }>["properties"]["error"]

function notify(api: TuiPluginApi, sessionID: string | undefined, message: string, sound: TuiAttentionSoundName) {
  const session = sessionID ? api.state.session.get(sessionID) : undefined
  const isSubagent = session?.parentID !== undefined
  void api.attention.notify({
    title: session?.title,
    message,
    notification: isSubagent ? false : { when: "blurred" },
    sound: { name: sound, when: "always" },
  })
}

function sessionErrorMessage(error: SessionError) {
  if (error?.name === "MessageAbortedError") return "Session aborted"
  const data = error?.data
  if (data && typeof data === "object" && "message" in data && data.message === "SSE read timed out") {
    return "Model stopped responding"
  }
  return "Session error"
}

const tui: TuiPlugin = async (api) => {
  const active = new Set<string>()
  const errored = new Set<string>()
  const questions = new Set<string>()
  const permissions = new Set<string>()

  api.event.on("question.asked", (event) => {
    if (questions.has(event.properties.id)) return
    questions.add(event.properties.id)
    notify(api, event.properties.sessionID, "Question needs input", "question")
  })

  api.event.on("question.replied", (event) => {
    questions.delete(event.properties.requestID)
  })

  api.event.on("question.rejected", (event) => {
    questions.delete(event.properties.requestID)
  })

  api.event.on("permission.asked", (event) => {
    if (permissions.has(event.properties.id)) return
    permissions.add(event.properties.id)
    notify(api, event.properties.sessionID, "Permission needs input", "permission")
  })

  api.event.on("permission.replied", (event) => {
    permissions.delete(event.properties.requestID)
  })

  api.event.on("session.status", (event) => {
    const sessionID = event.properties.sessionID
    const session = api.state.session.get(sessionID)
    const agentName = session?.agent

    if (event.properties.status.type === "busy" || event.properties.status.type === "retry") {
      const wasActive = active.has(sessionID)
      active.add(sessionID)
      errored.delete(sessionID)
      if (!wasActive && session?.parentID && agentName) {
        notify(api, sessionID, `Subagent started: ${agentName}`, "subagent_done")
      }
      return
    }

    if (event.properties.status.type !== "idle") return
    if (!active.has(sessionID)) return
    active.delete(sessionID)

    if (errored.has(sessionID)) {
      errored.delete(sessionID)
      return
    }

    if (session?.parentID && agentName) {
      notify(api, sessionID, `Subagent finished: ${agentName}`, "subagent_done")
    } else {
      notify(api, sessionID, "Session done", "done")
    }
  })

  api.event.on("session.error", (event) => {
    const sessionID = event.properties.sessionID
    if (!sessionID) return
    if (!active.has(sessionID)) return
    errored.add(sessionID)
    notify(api, sessionID, sessionErrorMessage(event.properties.error), "error")
  })
}

const plugin: BuiltinTuiPlugin = {
  id,
  tui,
}

export default plugin
