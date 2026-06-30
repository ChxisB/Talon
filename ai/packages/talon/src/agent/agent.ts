import { LayerNode } from "@talon-ai/core/effect/layer-node"
import { PermissionV1 } from "@talon-ai/core/v1/permission"
import { Config } from "@/config/config"
import { serviceUse } from "@talon-ai/core/effect/service-use"
import { Provider } from "@/provider/provider"

import { generateObject, streamObject, type ModelMessage } from "ai"
import { Truncate } from "@/tool/truncate"
import { Auth } from "../auth"
import { ProviderTransform } from "@/provider/transform"

import PROMPT_GENERATE from "./generate.txt"
import PROMPT_COMPACTION from "./prompt/compaction.txt"
import PROMPT_EXPLORE from "./prompt/explore.txt"
import PROMPT_SUMMARY from "./prompt/summary.txt"
import PROMPT_TITLE from "./prompt/title.txt"
import PROMPT_ARCHITECT from "./prompt/architect.txt"
import PROMPT_REVIEWER from "./prompt/reviewer.txt"
import PROMPT_LIBRARIAN from "./prompt/librarian.txt"
import PROMPT_PLANNER from "./prompt/planner.txt"
import PROMPT_ORCHESTRATOR from "./prompt/orchestrator.txt"
import PROMPT_GHOST from "./prompt/ghost.txt"
import PROMPT_GHOST_GPT from "./prompt/ghost-gpt.txt"
import PROMPT_PROMETHEUS from "./prompt/prometheus.txt"
import PROMPT_VISION_ANALYST from "./prompt/vision-analyst.txt"
import { Permission } from "@/permission"
import { mergeDeep, pipe, sortBy, values } from "remeda"
import { Global } from "@talon-ai/core/global"
import path from "path"
import { Plugin } from "@/plugin"
import { Skill } from "../skill"
import { Effect, Context, Layer, Schema } from "effect"
import { InstanceState } from "@/effect/instance-state"
import * as Option from "effect/Option"
import * as OtelTracer from "@effect/opentelemetry/Tracer"
import { AbsolutePath, type DeepMutable } from "@talon-ai/core/schema"
import { ProviderV2 } from "@talon-ai/core/provider"
import { ModelV2 } from "@talon-ai/core/model"
import { LocationServiceMap } from "@talon-ai/core/location-layer"
import { PluginBoot } from "@talon-ai/core/plugin/boot"
import { Reference } from "@talon-ai/core/reference"
import { Location } from "@talon-ai/core/location"

export const Info = Schema.Struct({
  name: Schema.String,
  description: Schema.optional(Schema.String),
  mode: Schema.Literals(["subagent", "primary", "all"]),
  native: Schema.optional(Schema.Boolean),
  hidden: Schema.optional(Schema.Boolean),
  topP: Schema.optional(Schema.Finite),
  temperature: Schema.optional(Schema.Finite),
  color: Schema.optional(Schema.String),
  permission: PermissionV1.Ruleset,
  model: Schema.optional(
    Schema.Struct({
      modelID: ModelV2.ID,
      providerID: ProviderV2.ID,
    }),
  ),
  useSmallModel: Schema.optional(Schema.Boolean),
  variant: Schema.optional(Schema.String),
  prompt: Schema.optional(Schema.String),
  options: Schema.Record(Schema.String, Schema.Unknown),
  steps: Schema.optional(Schema.Finite),
}).annotate({ identifier: "Agent" })
export type Info = DeepMutable<Schema.Schema.Type<typeof Info>>

const GeneratedAgent = Schema.Struct({
  identifier: Schema.String,
  whenToUse: Schema.String,
  systemPrompt: Schema.String,
})

export interface Interface {
  readonly get: (agent: string) => Effect.Effect<Info>
  readonly list: () => Effect.Effect<Info[]>
  readonly defaultInfo: () => Effect.Effect<Info>
  readonly defaultAgent: () => Effect.Effect<string>
  readonly generate: (input: {
    description: string
    model?: { providerID: ProviderV2.ID; modelID: ModelV2.ID }
  }) => Effect.Effect<
    {
      identifier: string
      whenToUse: string
      systemPrompt: string
    },
    Provider.DefaultModelError
  >
}

type State = Omit<Interface, "generate">

export class Service extends Context.Service<Service, Interface>()("@talon/Agent") {}

export const use = serviceUse(Service)

export const layer = Layer.effect(
  Service,
  Effect.gen(function* () {
    const config = yield* Config.Service
    const auth = yield* Auth.Service
    const plugin = yield* Plugin.Service
    const skill = yield* Skill.Service
    const provider = yield* Provider.Service
    const locations = yield* LocationServiceMap

    const state = yield* InstanceState.make<State>(
      Effect.fn("Agent.state")(function* (ctx) {
        const cfg = yield* config.get()
        const skillDirs = yield* skill.dirs()
        const referenceDirs = yield* Effect.gen(function* () {
          yield* (yield* PluginBoot.Service).wait()
          return (yield* (yield* Reference.Service).list()).map((reference) => reference.path)
        }).pipe(Effect.provide(locations.get(Location.Ref.make({ directory: AbsolutePath.make(ctx.directory) }))))
        const whitelistedDirs = [
          Truncate.GLOB,
          path.join(Global.Path.tmp, "*"),
          ...skillDirs.map((dir) => path.join(dir, "*")),
          ...referenceDirs.map((dir) => path.join(dir, "*")),
        ]
        const readonlyExternalDirectory = {
          "*": "ask",
          ...Object.fromEntries(whitelistedDirs.map((dir) => [dir, "allow"])),
        } satisfies Record<string, "allow" | "ask" | "deny">

        const defaults = Permission.fromConfig({
          "*": "allow",
          doom_loop: "ask",
          external_directory: {
            "*": "ask",
            ...Object.fromEntries(whitelistedDirs.map((dir) => [dir, "allow"])),
          },
          question: "deny",
          plan_enter: "deny",
          plan_exit: "deny",
          // mirrors github.com/github/gitignore Node.gitignore pattern for .env files
          read: {
            "*": "allow",
            "*.env": "ask",
            "*.env.*": "ask",
            "*.env.example": "allow",
          },
        })

        const user = Permission.fromConfig(cfg.permission ?? {})

        const agents: Record<string, Info> = {
          general: {
            name: "general-purpose",
            description: `General-purpose agent for researching complex questions, executing multi-step tasks, and synthesizing information across multiple sources. Use this agent to execute multiple units of work in parallel.`,
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                todowrite: "deny",
              }),
              user,
            ),
            options: {},
            mode: "subagent",
            native: true,
          },
          explore: {
            name: "codebase-explorer",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                grep: "allow",
                glob: "allow",
                list: "allow",
                bash: "allow",
                webfetch: "allow",
                websearch: "allow",
                read: "allow",
                external_directory: readonlyExternalDirectory,
              }),
              user,
            ),
            description: `Codebase exploration specialist. Quickly searches code by patterns, filenames, and content. Use this when you need to find files (e.g. find all React components), search for patterns (e.g. where is the auth middleware?), or investigate how code is organized. Specify thoroughness level: quick, medium, or very thorough.`,
            useSmallModel: true,
            prompt: PROMPT_EXPLORE,
            options: {},
            mode: "subagent",
            native: true,
          },
          compaction: {
            name: "compaction",
            mode: "primary",
            native: true,
            hidden: true,
            prompt: PROMPT_COMPACTION,
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
              }),
              user,
            ),
            options: {},
          },
          title: {
            name: "title",
            mode: "primary",
            options: {},
            native: true,
            hidden: true,
            temperature: 0.5,
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
              }),
              user,
            ),
            prompt: PROMPT_TITLE,
          },
          summary: {
            name: "summary",
            mode: "primary",
            options: {},
            native: true,
            hidden: true,
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
              }),
              user,
            ),
            prompt: PROMPT_SUMMARY,
          },

          // -- Discipline Agents (specialist roles for multi-agent orchestration) --

          architect: {
            name: "software-architect",
            description: "Software architecture consultant. Analyzes codebase structure, reviews design decisions, and provides architecture guidance. Use this for design reviews, technology choices, system decomposition, and architecture documentation.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                read: "allow",
                grep: "allow",
                glob: "allow",
                list: "allow",
                bash: "allow",
                webfetch: "allow",
                websearch: "allow",
                question: "allow",
                // Can write design documents to the plans directory
                edit: {
                  "*": "deny",
                  [path.join(".talon", "plans", "*.md")]: "allow",
                },
                external_directory: readonlyExternalDirectory,
              }),
              user,
            ),
            prompt: PROMPT_ARCHITECT,
            options: {},
            mode: "subagent",
            native: true,
          },

          reviewer: {
            name: "code-reviewer",
            description: "Code review specialist. Reviews code for correctness, security vulnerabilities, performance issues, and style violations. Provides actionable feedback with file paths and line numbers.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                read: "allow",
                grep: "allow",
                glob: "allow",
                list: "allow",
                bash: "allow",
                external_directory: readonlyExternalDirectory,
              }),
              user,
            ),
            prompt: PROMPT_REVIEWER,
            options: {},
            mode: "subagent",
            native: true,
          },

          librarian: {
            name: "codebase-researcher",
            description: "Deep codebase investigation specialist. Traces data flow across modules, maps dependencies, and understands complex interactions. Use this for understanding how a feature works end-to-end, finding all callers of a function, or mapping module dependencies.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                grep: "allow",
                glob: "allow",
                list: "allow",
                bash: "allow",
                read: "allow",
                webfetch: "allow",
                websearch: "allow",
                external_directory: readonlyExternalDirectory,
              }),
              user,
            ),
            prompt: PROMPT_LIBRARIAN,
            options: {},
            mode: "subagent",
            native: true,
          },

          planner: {
            name: "task-planner",
            description: "Strategic planning specialist. Breaks complex tasks into actionable, ordered steps with clear dependencies, parallelization opportunities, and acceptance criteria. Use this before starting multi-step implementations.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                read: "allow",
                grep: "allow",
                glob: "allow",
                list: "allow",
                bash: "allow",
                webfetch: "allow",
                websearch: "allow",
                question: "allow",
                // Can write plans to the plans directory
                edit: {
                  "*": "deny",
                  [path.join(".talon", "plans", "*.md")]: "allow",
                },
                external_directory: readonlyExternalDirectory,
              }),
              user,
            ),
            prompt: PROMPT_PLANNER,
            options: {},
            mode: "subagent",
            native: true,
          },

          orchestrator: {
            name: "multi-agent-orchestrator",
            hidden: true,
            description:
              "Multi-agent orchestrator. Decomposes complex tasks, delegates to specialist agents via the workflow tool, and synthesizes results. Coordinates parallel execution and manages dependencies between steps.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                // Orchestrator needs all tools including workflow
                workflow: "allow",
                question: "allow",
                plan_enter: "allow",
              }),
              user,
            ),
            prompt: PROMPT_ORCHESTRATOR,
            options: {},
            mode: "primary",
            native: true,
            color: "cyan",
          },

          ghost: {
            name: "ghost",
            description:
              "The sole AI agent. Decomposes complex tasks, delegates to specialist subagents, and drives work to completion with verification.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                workflow: "allow",
                pipeline: "allow",
                question: "allow",
              }),
              user,
            ),
            prompt: PROMPT_GHOST,
            options: {},
            mode: "primary",
            native: true,
            color: "#c792ea",
          },

          visionAnalyst: {
            name: "vision-analyst",
            description: "Image and document analyst. Analyzes images, screenshots, diagrams, and PDFs to extract information and provide detailed descriptions.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                read: "allow",
              }),
              user,
            ),
            prompt: PROMPT_VISION_ANALYST,
            options: {},
            mode: "subagent",
            native: true,
          },

          prometheus: {
            name: "strategy-consultant",
            description:
              "Strategic planning consultant. Interviews stakeholders, explores the codebase, and produces thorough adversarial plans before implementation begins. Identifies risks, edge cases, and hidden dependencies.",
            permission: Permission.merge(
              defaults,
              Permission.fromConfig({
                "*": "deny",
                read: "allow",
                grep: "allow",
                glob: "allow",
                list: "allow",
                bash: "allow",
                webfetch: "allow",
                websearch: "allow",
                question: "allow",
                edit: {
                  "*": "deny",
                  [path.join(".talon", "plans", "*.md")]: "allow",
                },
                external_directory: readonlyExternalDirectory,
              }),
              user,
            ),
            prompt: PROMPT_PROMETHEUS,
            options: {},
            mode: "subagent",
            native: true,
            color: "yellow",
          },
        }

        for (const [key, value] of Object.entries(cfg.agent ?? {})) {
          if (value.disable) {
            delete agents[key]
            // Also delete any existing agent whose display name matches this key,
            // preventing duplicate entries when the TUI saves per-agent configs
            // using agent.name (display name) as the config key.
            for (const existingKey of Object.keys(agents)) {
              if (agents[existingKey].name === key) {
                delete agents[existingKey]
              }
            }
            continue
          }
          let item = agents[key]
          if (!item) {
            // If a config key doesn't match any built-in agent key, check whether
            // any existing agent already has this display name. This prevents
            // duplicate entries when the TUI (or external config) saves per-agent
            // settings using agent.name (display name) as the config key instead
            // of the internal identifier key.
            const nameMatch = Object.entries(agents).find(([, v]) => v.name === key)
            if (nameMatch) {
              item = nameMatch[1]
            } else {
              item = agents[key] = {
                name: key,
                mode: "all",
                permission: Permission.merge(defaults, user),
                options: {},
                native: false,
              }
            }
          }
          if (value.model) item.model = Provider.parseModel(value.model)
          item.useSmallModel = value.use_small_model ?? item.useSmallModel
          item.variant = value.variant ?? item.variant
          item.prompt = value.prompt ?? item.prompt
          item.description = value.description ?? item.description
          item.temperature = value.temperature ?? item.temperature
          item.topP = value.top_p ?? item.topP
          item.mode = value.mode ?? item.mode
          item.color = value.color ?? item.color
          item.hidden = value.hidden ?? item.hidden
          item.name = value.name ?? item.name
          item.steps = value.steps ?? item.steps
          item.options = mergeDeep(item.options, value.options ?? {})
          item.permission = Permission.merge(item.permission, Permission.fromConfig(value.permission ?? {}))
        }

        // Ensure Truncate.GLOB is allowed unless explicitly configured
        for (const name in agents) {
          const agent = agents[name]
          const explicit = agent.permission.some((r) => {
            if (r.permission !== "external_directory") return false
            if (r.action !== "deny") return false
            return r.pattern === Truncate.GLOB
          })
          if (explicit) continue

          agents[name].permission = Permission.merge(
            agents[name].permission,
            Permission.fromConfig({ external_directory: { [Truncate.GLOB]: "allow" } }),
          )
        }

        const get = Effect.fnUntraced(function* (agent: string) {
          return agents[agent]
        })

        const list = Effect.fnUntraced(function* () {
          const cfg = yield* config.get()
          return pipe(
            agents,
            values(),
            sortBy(
              [(x) => (cfg.default_agent ? x.name === cfg.default_agent : x.name === "ghost"), "desc"],
              [(x) => x.name, "asc"],
            ),
          )
        })

        const defaultInfo = Effect.fnUntraced(function* () {
          const c = yield* config.get()
          if (c.default_agent) {
            const agent = agents[c.default_agent]
            if (!agent) throw new Error(`default agent "${c.default_agent}" not found`)
            if (agent.mode === "subagent") throw new Error(`default agent "${c.default_agent}" is a subagent`)
            if (agent.hidden === true) throw new Error(`default agent "${c.default_agent}" is hidden`)
            return agent
          }
          const visible = Object.values(agents).find((a) => a.mode !== "subagent" && a.hidden !== true)
          if (!visible) throw new Error("no primary visible agent found")
          return visible
        })

        const defaultAgent = Effect.fnUntraced(function* () {
          return (yield* defaultInfo()).name
        })

        return {
          get,
          list,
          defaultInfo,
          defaultAgent,
        } satisfies State
      }),
    )

    return Service.of({
      get: Effect.fn("Agent.get")(function* (agent: string) {
        return yield* InstanceState.useEffect(state, (s) => s.get(agent))
      }),
      list: Effect.fn("Agent.list")(function* () {
        return yield* InstanceState.useEffect(state, (s) => s.list())
      }),
      defaultInfo: Effect.fn("Agent.defaultInfo")(function* () {
        return yield* InstanceState.useEffect(state, (s) => s.defaultInfo())
      }),
      defaultAgent: Effect.fn("Agent.defaultAgent")(function* () {
        return yield* InstanceState.useEffect(state, (s) => s.defaultAgent())
      }),
      generate: Effect.fn("Agent.generate")(function* (input: {
        description: string
        model?: { providerID: ProviderV2.ID; modelID: ModelV2.ID }
      }) {
        const cfg = yield* config.get()
        const model = input.model ?? (yield* provider.defaultModel())
        const resolved = yield* provider.getModel(model.providerID, model.modelID)
        const language = yield* provider.getLanguage(resolved)
        const tracer = cfg.experimental?.openTelemetry
          ? Option.getOrUndefined(yield* Effect.serviceOption(OtelTracer.OtelTracer))
          : undefined

        const system = [PROMPT_GENERATE]
        yield* plugin.trigger("experimental.chat.system.transform", { model: resolved }, { system })
        const existing = yield* InstanceState.useEffect(state, (s) => s.list())

        // TODO: clean this up so provider specific logic doesnt bleed over
        const authInfo = yield* auth.get(model.providerID).pipe(Effect.orDie)
        const isOpenaiOauth = model.providerID === "openai" && authInfo?.type === "oauth"

        const params = {
          experimental_telemetry: {
            isEnabled: cfg.experimental?.openTelemetry,
            tracer,
            metadata: {
              userId: cfg.username ?? "unknown",
            },
          },
          temperature: 0.3,
          messages: [
            ...(isOpenaiOauth
              ? []
              : system.map(
                  (item): ModelMessage => ({
                    role: "system",
                    content: item,
                  }),
                )),
            {
              role: "user",
              content: `Create an agent configuration based on this request: "${input.description}".\n\nIMPORTANT: The following identifiers already exist and must NOT be used: ${existing.map((i) => i.name).join(", ")}\n  Return ONLY the JSON object, no other text, do not wrap in backticks`,
            },
          ],
          model: language,
          schema: Object.assign(
            Schema.toStandardSchemaV1(GeneratedAgent),
            Schema.toStandardJSONSchemaV1(GeneratedAgent),
          ),
        } satisfies Parameters<typeof generateObject>[0]

        if (isOpenaiOauth) {
          return yield* Effect.promise(async () => {
            const result = streamObject({
              ...params,
              providerOptions: ProviderTransform.providerOptions(resolved, {
                instructions: system.join("\n"),
                store: false,
              }),
              onError: () => {},
            })
            for await (const part of result.fullStream) {
              if (part.type === "error") throw part.error
            }
            return result.object
          })
        }

        return yield* Effect.promise(() => generateObject(params).then((r) => r.object))
      }),
    })
  }),
)

export const defaultLayer = layer.pipe(
  Layer.provide(Plugin.defaultLayer),
  Layer.provide(Provider.defaultLayer),
  Layer.provide(Auth.defaultLayer),
  Layer.provide(Config.defaultLayer),
  Layer.provide(Skill.defaultLayer),
  Layer.provide(LocationServiceMap.layer),
)

const locationServiceMapNode = LayerNode.make(LocationServiceMap.layer, [])

export const node = LayerNode.make(layer, [
  Config.node,
  Auth.node,
  Plugin.node,
  Skill.node,
  Provider.node,
  locationServiceMapNode,
])

export * as Agent from "./agent"
