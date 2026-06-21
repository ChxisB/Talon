import { Context } from "effect"
import type { InstanceContext } from "@/project/instance-context"
import type { WorkspaceV2 } from "@talon-ai/core/workspace"

export const InstanceRef = Context.Reference<InstanceContext | undefined>("~talon/InstanceRef", {
  defaultValue: () => undefined,
})

export const WorkspaceRef = Context.Reference<WorkspaceV2.ID | undefined>("~talon/WorkspaceRef", {
  defaultValue: () => undefined,
})
