import { Schema } from "effect"
import { HttpApi, HttpApiEndpoint, HttpApiError, HttpApiGroup, OpenApi } from "effect/unstable/httpapi"
import { TaskV2 } from "@talon-ai/core/task"
import { Authorization } from "../middleware/authorization"
import { InstanceContextMiddleware } from "../middleware/instance-context"
import { WorkspaceRoutingMiddleware, WorkspaceRoutingQuery } from "../middleware/workspace-routing"
import { described } from "./metadata"

const root = "/task"

const NullString = Schema.NullOr(Schema.String)

export const TaskSchema = Schema.Struct({
  id: TaskV2.ID,
  title: Schema.String,
  status: Schema.String,
  assignee: NullString,
  workspaceID: NullString,
  projectID: Schema.String,
  parentID: NullString,
  orderKey: NullString,
  summary: NullString,
  metadata: Schema.NullOr(Schema.Record(Schema.String, Schema.Unknown)),
  timeCreated: Schema.Number,
  timeUpdated: Schema.Number,
  timeArchived: Schema.NullOr(Schema.Number),
}).annotate({ identifier: "Task" })

export type TaskSchemaType = Schema.Schema.Type<typeof TaskSchema>

export const TaskAggregateSchema = Schema.Struct({
  task: TaskSchema,
  sessions: Schema.Array(
    Schema.Struct({
      id: Schema.String,
      title: Schema.String,
      timeCreated: Schema.Number,
    }),
  ),
  artifacts: Schema.Array(
    Schema.Struct({
      id: Schema.String,
      type: Schema.String,
      title: Schema.String,
      status: Schema.String,
      parentID: NullString,
    }),
  ),
  diff: NullString,
}).annotate({ identifier: "TaskAggregate" })

export type TaskAggregateSchemaType = Schema.Schema.Type<typeof TaskAggregateSchema>

export const CreateTaskPayload = Schema.Struct({
  title: Schema.String,
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ),
  assignee: Schema.optional(Schema.String),
  parentID: Schema.optional(TaskV2.ID),
  summary: Schema.optional(Schema.String),
})

export type CreateTaskPayloadType = Schema.Schema.Type<typeof CreateTaskPayload>

export const UpdateTaskPayload = Schema.Struct({
  title: Schema.optional(Schema.String),
  status: Schema.optional(
    Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"]),
  ),
  assignee: Schema.optional(Schema.String),
  summary: Schema.optional(Schema.String),
})

export type UpdateTaskPayloadType = Schema.Schema.Type<typeof UpdateTaskPayload>

export const ListTaskQuery = Schema.Struct({
  ...WorkspaceRoutingQuery.fields,
  status: Schema.optional(Schema.String),
})

export type ListTaskQueryType = Schema.Schema.Type<typeof ListTaskQuery>

export const TaskIDParam = Schema.Struct({
  id: TaskV2.ID,
})

export const TaskPaths = {
  list: root,
  get: `${root}/:id`,
  create: root,
  update: `${root}/:id`,
  remove: `${root}/:id`,
  aggregate: `${root}/:id/aggregate`,
} as const

export const TaskApi = HttpApi.make("task")
  .add(
    HttpApiGroup.make("task")
      .add(
        HttpApiEndpoint.get("list", TaskPaths.list, {
          query: ListTaskQuery,
          success: described(Schema.Array(TaskSchema), "List of tasks"),
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "task.list",
            summary: "List tasks",
            description: "Get a list of all tasks for the current project, optionally filtered.",
          }),
        ),
        HttpApiEndpoint.get("get", TaskPaths.get, {
          params: TaskIDParam,
          query: WorkspaceRoutingQuery,
          success: described(TaskSchema, "Get task"),
          error: HttpApiError.BadRequest,
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "task.get",
            summary: "Get task",
            description: "Retrieve detailed information about a specific task.",
          }),
        ),
        HttpApiEndpoint.post("create", TaskPaths.create, {
          query: WorkspaceRoutingQuery,
          payload: CreateTaskPayload,
          success: described(TaskSchema, "Created task"),
          error: HttpApiError.BadRequest,
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "task.create",
            summary: "Create task",
            description: "Create a new task in the current project.",
          }),
        ),
        HttpApiEndpoint.patch("update", TaskPaths.update, {
          params: TaskIDParam,
          query: WorkspaceRoutingQuery,
          payload: UpdateTaskPayload,
          success: described(TaskSchema, "Updated task"),
          error: [HttpApiError.BadRequest, HttpApiError.NotFound],
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "task.update",
            summary: "Update task",
            description: "Update an existing task's title, status, or assignee.",
          }),
        ),
        HttpApiEndpoint.delete("remove", TaskPaths.remove, {
          params: TaskIDParam,
          query: WorkspaceRoutingQuery,
          success: described(Schema.Boolean, "Successfully deleted task"),
          error: [HttpApiError.BadRequest, HttpApiError.NotFound],
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "task.remove",
            summary: "Remove task",
            description: "Delete a task and all of its descendants (recursive cascade).",
          }),
        ),
      )
      .annotateMerge(
        OpenApi.annotations({
          title: "task",
          description: "Task management routes.",
        }),
      )
      .middleware(InstanceContextMiddleware)
      .middleware(WorkspaceRoutingMiddleware)
      .middleware(Authorization),
  )
  .annotateMerge(
    OpenApi.annotations({
      title: "talon experimental HttpApi",
      version: "0.0.1",
      description: "Experimental HttpApi surface for selected instance routes.",
    }),
  )
