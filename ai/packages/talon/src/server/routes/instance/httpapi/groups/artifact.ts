import { Schema } from "effect"
import { HttpApi, HttpApiEndpoint, HttpApiError, HttpApiGroup, OpenApi } from "effect/unstable/httpapi"
import { ArtifactV2 } from "@talon-ai/core/artifact"
import { Authorization } from "../middleware/authorization"
import { InstanceContextMiddleware } from "../middleware/instance-context"
import { WorkspaceRoutingMiddleware, WorkspaceRoutingQuery } from "../middleware/workspace-routing"
import { described } from "./metadata"

const root = "/artifact"

/**
 * Schema for an artifact response. Mirrors the Artifact interface.
 * Uses Schema.NullOr for fields that can be null, and Schema.optional for fields that can be absent.
 */
const NullString = Schema.NullOr(Schema.String)
const NullNumber = Schema.NullOr(Schema.Number)
const NullArtifactID = Schema.NullOr(ArtifactV2.ID)
const NullRecord = Schema.NullOr(Schema.Record(Schema.String, Schema.Unknown))

export const ArtifactSchema = Schema.Struct({
  id: ArtifactV2.ID,
  type: Schema.String,
  title: Schema.String,
  status: Schema.String,
  assignee: NullString,
  taskID: NullString,
  workspaceID: NullString,
  projectID: Schema.String,
  parentID: NullArtifactID,
  orderKey: NullString,
  path: Schema.String,
  bodyHash: NullString,
  frontmatter: NullRecord,
  timeCreated: Schema.Number,
  timeUpdated: Schema.Number,
  timeArchived: NullNumber,
}).annotate({ identifier: "Artifact" })

export type ArtifactSchemaType = Schema.Schema.Type<typeof ArtifactSchema>

/**
 * Payload for creating an artifact.
 */
export const CreateArtifactPayload = Schema.Struct({
  type: Schema.Literals(["spec", "ticket", "story", "review", "plan", "note"]),
  title: Schema.String,
  body: Schema.optional(Schema.String),
  status: Schema.optional(Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"])),
  parentID: Schema.optional(ArtifactV2.ID),
  taskID: Schema.optional(Schema.String),
  assignee: Schema.optional(Schema.String),
})

export type CreateArtifactPayloadType = Schema.Schema.Type<typeof CreateArtifactPayload>

/**
 * Payload for updating an artifact. All fields optional.
 */
export const UpdateArtifactPayload = Schema.Struct({
  title: Schema.optional(Schema.String),
  status: Schema.optional(Schema.Literals(["todo", "in_progress", "done", "blocked", "archived"])),
  assignee: Schema.optional(Schema.String),
  orderKey: Schema.optional(Schema.String),
  taskID: Schema.optional(Schema.String),
  body: Schema.optional(Schema.String),
})

export type UpdateArtifactPayloadType = Schema.Schema.Type<typeof UpdateArtifactPayload>

/**
 * Query parameters for listing artifacts.
 */
export const ListArtifactQuery = Schema.Struct({
  ...WorkspaceRoutingQuery.fields,
  type: Schema.optional(Schema.String),
  status: Schema.optional(Schema.String),
  taskID: Schema.optional(Schema.String),
})

export type ListArtifactQueryType = Schema.Schema.Type<typeof ListArtifactQuery>

/**
 * Path parameters for artifact operations.
 */
export const ArtifactIDParam = Schema.Struct({
  id: ArtifactV2.ID,
})

export const ArtifactPaths = {
  list: root,
  get: `${root}/:id`,
  create: root,
  update: `${root}/:id`,
  remove: `${root}/:id`,
} as const

export const ArtifactApi = HttpApi.make("artifact")
  .add(
    HttpApiGroup.make("artifact")
      .add(
        HttpApiEndpoint.get("list", ArtifactPaths.list, {
          query: ListArtifactQuery,
          success: described(Schema.Array(ArtifactSchema), "List of artifacts"),
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "artifact.list",
            summary: "List artifacts",
            description: "Get a list of all artifacts for the current project, optionally filtered.",
          }),
        ),
        HttpApiEndpoint.get("get", ArtifactPaths.get, {
          params: ArtifactIDParam,
          query: WorkspaceRoutingQuery,
          success: described(ArtifactSchema, "Get artifact"),
          error: HttpApiError.BadRequest,
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "artifact.get",
            summary: "Get artifact",
            description: "Retrieve detailed information about a specific artifact.",
          }),
        ),
        HttpApiEndpoint.post("create", ArtifactPaths.create, {
          query: WorkspaceRoutingQuery,
          payload: CreateArtifactPayload,
          success: described(ArtifactSchema, "Created artifact"),
          error: HttpApiError.BadRequest,
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "artifact.create",
            summary: "Create artifact",
            description: "Create a new artifact in the current project.",
          }),
        ),
        HttpApiEndpoint.patch("update", ArtifactPaths.update, {
          params: ArtifactIDParam,
          query: WorkspaceRoutingQuery,
          payload: UpdateArtifactPayload,
          success: described(ArtifactSchema, "Updated artifact"),
          error: [HttpApiError.BadRequest, HttpApiError.NotFound],
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "artifact.update",
            summary: "Update artifact",
            description: "Update an existing artifact's title, status, assignee, body, or other fields.",
          }),
        ),
        HttpApiEndpoint.delete("remove", ArtifactPaths.remove, {
          params: ArtifactIDParam,
          query: WorkspaceRoutingQuery,
          success: described(Schema.Boolean, "Successfully deleted artifact"),
          error: [HttpApiError.BadRequest, HttpApiError.NotFound],
        }).annotateMerge(
          OpenApi.annotations({
            identifier: "artifact.remove",
            summary: "Remove artifact",
            description: "Delete an artifact and all of its descendants (recursive cascade).",
          }),
        ),
      )
      .annotateMerge(
        OpenApi.annotations({
          title: "artifact",
          description: "Artifact management routes.",
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
