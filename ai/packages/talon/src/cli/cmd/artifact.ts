import type { Argv } from "yargs"
import { Effect } from "effect"
import { cmd } from "./cmd"
import { effectCmd, fail } from "../effect-cmd"
import { UI } from "../ui"
import type { Interface as ArtifactInterface } from "@talon-ai/core/artifact/index"
import { ArtifactV2 } from "@talon-ai/core/artifact"
import { InstanceState } from "@/effect/instance-state"
import { resolveArtifactService } from "@/tool/artifact-helper"
import type { ArtifactType, ArtifactStatus } from "@talon-ai/core/artifact/index"

const FORMAT_CHOICES = ["table", "json"] as const

const ARTIFACT_TYPE_CHOICES = ["spec", "ticket", "story", "review", "plan", "note"] as const
const ARTIFACT_STATUS_CHOICES = ["todo", "in_progress", "done", "blocked", "archived"] as const

export const ArtifactCommand = cmd({
  command: "artifact",
  describe: "manage artifacts",
  builder: (yargs: Argv) =>
    yargs
      .command(ArtifactCreateCommand)
      .command(ArtifactListCommand)
      .command(ArtifactGetCommand)
      .command(ArtifactUpdateCommand)
      .command(ArtifactRemoveCommand)
      .demandCommand(),
  async handler() {},
})

export const ArtifactCreateCommand = effectCmd({
  command: "create <type> <title>",
  describe: "create a new artifact",
  builder: (yargs) =>
    yargs
      .positional("type", {
        describe: "artifact type",
        type: "string",
        choices: ARTIFACT_TYPE_CHOICES,
        demandOption: true,
      })
      .positional("title", {
        describe: "artifact title",
        type: "string",
        demandOption: true,
      })
      .option("body", {
        describe: "artifact body content",
        type: "string",
      })
      .option("status", {
        describe: "artifact status",
        type: "string",
        choices: ARTIFACT_STATUS_CHOICES,
        default: "todo" as const,
      })
      .option("parent-id", {
        describe: "parent artifact ID",
        type: "string",
      })
      .option("task-id", {
        describe: "task ID",
        type: "string",
      })
      .option("assignee", {
        describe: "assignee",
        type: "string",
      }),
  handler: Effect.fn("Cli.artifact.create")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: ArtifactInterface = yield* resolveArtifactService(ctx)
    const artifact = yield* svc.create({
      type: args.type as ArtifactType,
      title: args.title,
      body: args.body,
      status: args.status as ArtifactStatus,
      parentID: args.parentId,
      taskID: args.taskId,
      assignee: args.assignee,
    })
    UI.println(
      UI.Style.TEXT_SUCCESS_BOLD + `Artifact created: ${artifact.id}` + UI.Style.TEXT_NORMAL,
    )
    UI.println(`  Type: ${artifact.type}`)
    UI.println(`  Title: ${artifact.title}`)
    UI.println(`  Status: ${artifact.status}`)
    UI.println(`  Path: ${artifact.path}`)
  }),
})

export const ArtifactListCommand = effectCmd({
  command: "list",
  describe: "list artifacts",
  builder: (yargs) =>
    yargs
      .option("type", {
        describe: "filter by artifact type",
        type: "string",
        choices: ARTIFACT_TYPE_CHOICES,
      })
      .option("status", {
        describe: "filter by status",
        type: "string",
        choices: ARTIFACT_STATUS_CHOICES,
      })
      .option("task-id", {
        describe: "filter by task ID",
        type: "string",
      })
      .option("format", {
        describe: "output format",
        type: "string",
        choices: FORMAT_CHOICES,
        default: "table" as const,
      }),
  handler: Effect.fn("Cli.artifact.list")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: ArtifactInterface = yield* resolveArtifactService(ctx)
    const artifacts = yield* svc.list({
      type: args.type,
      status: args.status,
      taskID: args.taskId,
    })
    if (artifacts.length === 0) {
      UI.println("No artifacts found.")
      return
    }
    if (args.format === "json") {
      UI.println(JSON.stringify(artifacts, null, 2))
      return
    }
    // Table format
    const maxIdWidth = Math.max(8, ...artifacts.map((a: any) => a.id.length))
    const maxTypeWidth = Math.max(4, ...artifacts.map((a: any) => a.type.length))
    const maxTitleWidth = Math.max(5, ...artifacts.map((a: any) => a.title.length))
    const maxStatusWidth = Math.max(6, ...artifacts.map((a: any) => a.status.length))

    const header =
      `ID${" ".repeat(maxIdWidth - 2)}  ` +
      `Type${" ".repeat(maxTypeWidth - 4)}  ` +
      `Title${" ".repeat(maxTitleWidth - 5)}  ` +
      `Status${" ".repeat(maxStatusWidth - 6)}`
    UI.println(header)
    UI.println("─".repeat(header.length))
    for (const a of artifacts) {
      UI.println(
        `${a.id.padEnd(maxIdWidth)}  ${a.type.padEnd(maxTypeWidth)}  ${a.title.padEnd(maxTitleWidth)}  ${a.status.padEnd(maxStatusWidth)}`,
      )
    }
  }),
})

export const ArtifactGetCommand = effectCmd({
  command: "get <id>",
  describe: "get an artifact by ID",
  builder: (yargs) =>
    yargs.positional("id", {
      describe: "artifact ID (art_...)",
      type: "string",
      demandOption: true,
    }),
  handler: Effect.fn("Cli.artifact.get")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: ArtifactInterface = yield* resolveArtifactService(ctx)
    const id = ArtifactV2.ID.ascending(args.id) as ArtifactV2.ID
    const artifact = yield* svc.get(id)
    if (!artifact) return yield* fail(`Artifact not found: ${args.id}`)
    UI.println(UI.Style.TEXT_HIGHLIGHT_BOLD + `Artifact: ${artifact.id}` + UI.Style.TEXT_NORMAL)
    UI.println(`  Type:      ${artifact.type}`)
    UI.println(`  Title:     ${artifact.title}`)
    UI.println(`  Status:    ${artifact.status}`)
    UI.println(`  Path:      ${artifact.path}`)
    if (artifact.assignee) UI.println(`  Assignee:  ${artifact.assignee}`)
    if (artifact.parentID) UI.println(`  Parent:    ${artifact.parentID}`)
    if (artifact.taskID) UI.println(`  Task:      ${artifact.taskID}`)
    UI.println(`  Created:   ${new Date(artifact.timeCreated).toISOString()}`)
    UI.println(`  Updated:   ${new Date(artifact.timeUpdated).toISOString()}`)
    if (artifact.frontmatter && Object.keys(artifact.frontmatter).length > 0) {
      UI.println(`  Metadata:  ${JSON.stringify(artifact.frontmatter)}`)
    }
    if (artifact.bodyHash) UI.println(`  Body hash: ${artifact.bodyHash}`)
  }),
})

export const ArtifactUpdateCommand = effectCmd({
  command: "update <id>",
  describe: "update an artifact",
  builder: (yargs) =>
    yargs
      .positional("id", {
        describe: "artifact ID (art_...)",
        type: "string",
        demandOption: true,
      })
      .option("title", {
        describe: "new title",
        type: "string",
      })
      .option("status", {
        describe: "new status",
        type: "string",
        choices: ARTIFACT_STATUS_CHOICES,
      })
      .option("assignee", {
        describe: "new assignee",
        type: "string",
      })
      .option("body", {
        describe: "new body content",
        type: "string",
      }),
  handler: Effect.fn("Cli.artifact.update")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: ArtifactInterface = yield* resolveArtifactService(ctx)
    const id = ArtifactV2.ID.ascending(args.id) as ArtifactV2.ID
    const updates: Record<string, unknown> = {}
    if (args.title !== undefined) updates.title = args.title
    if (args.status !== undefined) updates.status = args.status
    if (args.assignee !== undefined) updates.assignee = args.assignee
    if (args.body !== undefined) updates.body = args.body
    if (Object.keys(updates).length === 0) return yield* fail("No updates provided. Use --title, --status, --assignee, or --body.")
    yield* svc.update(id, updates as any)
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + `Artifact ${args.id} updated.` + UI.Style.TEXT_NORMAL)
  }),
})

export const ArtifactRemoveCommand = effectCmd({
  command: "remove <id>",
  describe: "remove an artifact and its descendants",
  builder: (yargs) =>
    yargs.positional("id", {
      describe: "artifact ID (art_...)",
      type: "string",
      demandOption: true,
    }),
  handler: Effect.fn("Cli.artifact.remove")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: ArtifactInterface = yield* resolveArtifactService(ctx)
    const id = ArtifactV2.ID.ascending(args.id) as ArtifactV2.ID
    yield* svc.remove(id)
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + `Artifact ${args.id} and its descendants removed.` + UI.Style.TEXT_NORMAL)
  }),
})
