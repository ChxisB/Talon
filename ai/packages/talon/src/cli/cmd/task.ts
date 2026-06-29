import type { Argv } from "yargs"
import { Effect } from "effect"
import { cmd } from "./cmd"
import { effectCmd, fail } from "../effect-cmd"
import { UI } from "../ui"
import type { Interface as TaskInterface } from "@talon-ai/core/control-plane/task/index"
import { resolveTaskService } from "@/tool/task-helper"
import { InstanceState } from "@/effect/instance-state"

const FORMAT_CHOICES = ["table", "json"] as const
const TASK_STATUS_CHOICES = ["todo", "in_progress", "done", "blocked", "archived"] as const

export const TaskCommand = cmd({
  command: "task",
  describe: "manage tasks",
  builder: (yargs: Argv) =>
    yargs
      .command(TaskCreateCommand)
      .command(TaskListCommand)
      .command(TaskGetCommand)
      .command(TaskUpdateCommand)
      .command(TaskRemoveCommand)
      .command(TaskFocusCommand)
      .demandCommand(),
  async handler() {},
})

export const TaskCreateCommand = effectCmd({
  command: "create <title>",
  describe: "create a new task",
  builder: (yargs) =>
    yargs
      .positional("title", { describe: "task title", type: "string", demandOption: true })
      .option("status", { describe: "task status", type: "string", choices: TASK_STATUS_CHOICES, default: "todo" as const })
      .option("assignee", { describe: "assignee", type: "string" })
      .option("parent-id", { describe: "parent task ID", type: "string" })
      .option("summary", { describe: "task summary", type: "string" }),
  handler: Effect.fn("Cli.task.create")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)
    const task = yield* svc.create({
      title: args.title,
      status: args.status,
      assignee: args.assignee,
      parentID: args.parentId,
      summary: args.summary,
    })
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + `Task created: ${task.id}` + UI.Style.TEXT_NORMAL)
    UI.println(`  Title: ${task.title}`)
    UI.println(`  Status: ${task.status}`)
    if (task.assignee) UI.println(`  Assignee: ${task.assignee}`)
  }),
})

export const TaskListCommand = effectCmd({
  command: "list",
  describe: "list tasks",
  builder: (yargs) =>
    yargs
      .option("status", { describe: "filter by status", type: "string", choices: TASK_STATUS_CHOICES })
      .option("format", { describe: "output format", type: "string", choices: FORMAT_CHOICES, default: "table" as const }),
  handler: Effect.fn("Cli.task.list")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)
    const tasks = yield* svc.list({ status: args.status })

    if (tasks.length === 0) {
      UI.println("No tasks found.")
      return
    }

    if (args.format === "json") {
      UI.println(JSON.stringify(tasks, null, 2))
      return
    }

    const header = `${"ID".padEnd(30)} ${"Status".padEnd(14)} ${"Assignee".padEnd(12)} Title`
    UI.println(header)
    UI.println("-".repeat(header.length))
    for (const t of tasks) {
      UI.println(`${t.id.padEnd(30)} ${t.status.padEnd(14)} ${(t.assignee ?? "-").padEnd(12)} ${t.title}`)
    }
  }),
})

export const TaskGetCommand = effectCmd({
  command: "get <id>",
  describe: "get task details",
  builder: (yargs) =>
    yargs.positional("id", { describe: "task ID (tsk_*)", type: "string", demandOption: true }),
  handler: Effect.fn("Cli.task.get")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)
    const task = yield* svc.get(args.id)
    if (!task) return yield* fail(`Task not found: ${args.id}`)
    UI.println(`ID: ${task.id}`)
    UI.println(`Title: ${task.title}`)
    UI.println(`Status: ${task.status}`)
    if (task.assignee) UI.println(`Assignee: ${task.assignee}`)
    if (task.summary) UI.println(`Summary: ${task.summary}`)
  }),
})

export const TaskUpdateCommand = effectCmd({
  command: "update <id>",
  describe: "update a task",
  builder: (yargs) =>
    yargs
      .positional("id", { describe: "task ID (tsk_*)", type: "string", demandOption: true })
      .option("title", { describe: "new title", type: "string" })
      .option("status", { describe: "new status", type: "string", choices: TASK_STATUS_CHOICES })
      .option("assignee", { describe: "new assignee", type: "string" })
      .option("summary", { describe: "new summary", type: "string" }),
  handler: Effect.fn("Cli.task.update")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)
    const existing = yield* svc.get(args.id)
    if (!existing) return yield* fail(`Task not found: ${args.id}`)
    const updates: Record<string, unknown> = {}
    if (args.title !== undefined) updates.title = args.title
    if (args.status !== undefined) updates.status = args.status
    if (args.assignee !== undefined) updates.assignee = args.assignee
    if (args.summary !== undefined) updates.summary = args.summary
    yield* svc.update(args.id, updates)
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + `Task updated: ${args.id}` + UI.Style.TEXT_NORMAL)
  }),
})

export const TaskRemoveCommand = effectCmd({
  command: "remove <id>",
  describe: "remove a task and its descendants",
  builder: (yargs) =>
    yargs.positional("id", { describe: "task ID (tsk_*)", type: "string", demandOption: true }),
  handler: Effect.fn("Cli.task.remove")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)
    const existing = yield* svc.get(args.id)
    if (!existing) return yield* fail(`Task not found: ${args.id}`)
    yield* svc.remove(args.id)
    UI.println(UI.Style.TEXT_SUCCESS_BOLD + `Task removed: ${args.id}` + UI.Style.TEXT_NORMAL)
  }),
})

export const TaskFocusCommand = effectCmd({
  command: "focus [id]",
  describe: "set or show the focused task",
  builder: (yargs) =>
    yargs.positional("id", { describe: "task ID to focus on (omit to show current)", type: "string" }),
  handler: Effect.fn("Cli.task.focus")(function* (args: any) {
    const ctx = yield* InstanceState.context
    const svc: TaskInterface = yield* resolveTaskService(ctx)
    if (args.id) {
      const existing = yield* svc.get(args.id)
      if (!existing) return yield* fail(`Task not found: ${args.id}`)
      yield* svc.focus(args.id)
      UI.println(UI.Style.TEXT_SUCCESS_BOLD + `Focused task: ${args.id}` + UI.Style.TEXT_NORMAL)
    } else {
      const focusedId = yield* svc.focused()
      if (!focusedId) {
        UI.println("No focused task.")
        return
      }
      const task = yield* svc.get(focusedId)
      if (task) {
        UI.println(`Focused task: ${task.id}`)
        UI.println(`  Title: ${task.title}`)
        UI.println(`  Status: ${task.status}`)
      } else {
        UI.println(`Focused task ID: ${focusedId} (not found)`)
      }
    }
  }),
})
