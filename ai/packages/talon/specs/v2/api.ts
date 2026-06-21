// @ts-nocheck

import { OpenCode } from "@talon-ai/core"
import { ReadTool } from "@talon-ai/core/tools"

const talon = OpenCode.make({})

talon.tool.add(ReadTool)

talon.tool.add({
  name: "bash",
  schema: {
    type: "object",
    properties: {
      command: {
        type: "string",
        description: "The command to run.",
      },
    },
    required: ["command"],
  },
  execute(input, ctx) {},
})

talon.auth.add({
  provider: "openai",
  type: "api",
  value: process.env.OPENAI_API_KEY,
})

talon.agent.add({
  name: "build",
  permissions: [],
  model: {
    id: "gpt-5-5",
    provider: "openai",
    variant: "xhigh",
  },
})

const sessionID = await talon.session.create({
  agent: "build",
})

talon.subscribe((event) => {
  console.log(event)
})

await talon.session.prompt({
  sessionID,
  text: "hey what is up",
})

await talon.session.prompt({
  sessionID,
  text: "what is up with this",
  files: [
    {
      mime: "image/png",
      uri: "data:image/png;base64,xxxx",
    },
  ],
})

await talon.session.wait()

console.log(await talon.session.messages(sessionID))
