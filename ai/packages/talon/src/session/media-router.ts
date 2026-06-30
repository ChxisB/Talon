import { SessionV1 } from "@talon-ai/core/v1/session"
import { ProviderV2 } from "@talon-ai/core/provider"
import { ModelV2 } from "@talon-ai/core/model"
import type { Agent } from "@/agent/agent"
import type { Provider } from "@/provider/provider"
import { isMedia } from "@/util/media"
import { LLMEvent } from "@talon-ai/llm"
import { Effect, Schema } from "effect"
import * as Stream from "effect/Stream"
import type { UserModelMessage } from "ai"
import type { LLM } from "./llm"

const VISION_SYSTEM_PROMPT = `You are a precise image and document analyst. Your job is to examine the attached media carefully and describe everything you see in detail. Focus on:

- For images/screenshots: Describe UI layouts, code snippets, error messages, diagrams, text content, colors, and any visual elements relevant to software development.
- For PDFs/documents: Extract key information, structure, code blocks, technical specifications, and actionable content.
- Be thorough and precise. Your analysis will be passed to a coding assistant that cannot see the original media.

Output only your analysis text — no greetings, no meta-commentary.`

export class NoVisionModelConfiguredError extends Schema.TaggedErrorClass<NoVisionModelConfiguredError>()(
  "MediaRouterNoVisionModelConfiguredError",
  {},
) {}

export class VisionModelNotFoundError extends Schema.TaggedErrorClass<VisionModelNotFoundError>()(
  "MediaRouterVisionModelNotFoundError",
  {
    value: Schema.String,
  },
) {}

export class VisionModelMissingCapabilityError extends Schema.TaggedErrorClass<VisionModelMissingCapabilityError>()(
  "MediaRouterVisionModelMissingCapabilityError",
  {
    modelID: Schema.String,
    mediaType: Schema.String,
  },
) {}

/**
 * Check if any parts contain image or PDF attachments that can be routed
 * to a vision model.
 */
export function hasMediaAttachments(parts: SessionV1.Part[]): boolean {
  return parts.some(
    (part) => part.type === "file" && isMedia(part.mime),
  )
}

/**
 * Parse a "provider/model" string into structured IDs.
 * Returns undefined if the string is empty or unparsable.
 */
export function parseModelString(
  value: string | undefined,
): { providerID: ProviderV2.ID; modelID: ModelV2.ID } | undefined {
  if (!value || typeof value !== "string") return undefined
  const slash = value.lastIndexOf("/")
  if (slash === -1 || slash === 0 || slash === value.length - 1) return undefined
  return {
    providerID: ProviderV2.ID.make(value.slice(0, slash)),
    modelID: ModelV2.ID.make(value.slice(slash + 1)),
  }
}

/**
 * Validate that the selected vision model supports the media types
 * present in the message parts. Logs a warning but does not block —
 * capability metadata may be incomplete.
 */
export function validateModelCapabilities(
  model: Provider.Model,
  parts: SessionV1.Part[],
): Effect.Effect<void, VisionModelMissingCapabilityError> {
  for (const part of parts) {
    if (part.type !== "file") continue
    if (part.mime.startsWith("image/") && !model.capabilities.input.image) {
      return Effect.fail(
        new VisionModelMissingCapabilityError({ modelID: model.id, mediaType: part.mime }),
      )
    }
    if (part.mime === "application/pdf" && !model.capabilities.input.pdf) {
      return Effect.fail(
        new VisionModelMissingCapabilityError({ modelID: model.id, mediaType: part.mime }),
      )
    }
  }
  return Effect.void
}

/**
 * Build an AI SDK user message that includes media attachments as
 * ImagePart / FilePart content alongside the original user text.
 */
function buildVisionMessage(
  originalTextParts: string[],
  fileParts: SessionV1.FilePart[],
): UserModelMessage {
  const content: UserModelMessage["content"] = []

  if (originalTextParts.length > 0) {
    content.push({ type: "text", text: originalTextParts.join("\n") })
  }

  for (const file of fileParts) {
    if (file.mime.startsWith("image/")) {
      content.push({
        type: "image",
        image: file.url,
        mediaType: file.mime,
      })
    } else {
      content.push({
        type: "file",
        data: file.url,
        filename: file.filename,
        mediaType: file.mime,
      })
    }
  }

  return { role: "user", content }
}

/**
 * Send the media attachments to the configured vision model for analysis.
 * Returns the analysis text, or `{ analyzed: false }` if no media to analyze.
 *
 * Called within the session prompt's Effect context where services are available.
 */
export function analyzeAttachments(
  parts: SessionV1.Part[],
  textParts: string[],
  model: Provider.Model,
  sessionID: string,
  agent: Agent.Info,
  user: SessionV1.User,
  llmStream: (input: LLM.StreamInput) => Stream.Stream<LLMEvent, unknown>,
  systemPrompt?: string,
): Effect.Effect<{ analysis: string; analyzed: boolean }> {
  return Effect.gen(function* () {
    const fileParts: SessionV1.FilePart[] = parts.filter(
      (part): part is SessionV1.FilePart => part.type === "file" && isMedia(part.mime),
    )
    if (fileParts.length === 0) return { analysis: "", analyzed: false }

    // Warn if model metadata says it doesn't support the media types
    yield* validateModelCapabilities(model, fileParts).pipe(
      Effect.catch((err) =>
        Effect.logWarning("vision model capability mismatch (proceeding anyway)", {
          modelID: err.modelID,
          mediaType: err.mediaType,
        }),
      ),
    )

    const visionMessage = buildVisionMessage(textParts, fileParts)

    const analysisText = yield* llmStream({
      user,
      agent,
      model,
      sessionID,
      system: [systemPrompt ?? VISION_SYSTEM_PROMPT],
      small: false,
      tools: {},
      messages: [visionMessage],
      retries: 1,
    }).pipe(
      Stream.filter(LLMEvent.is.textDelta),
      Stream.map((e) => e.text),
      Stream.mkString,
      Effect.orDie,
    )

    return { analysis: analysisText, analyzed: analysisText.length > 0 }
  })
}

export * as MediaRouter from "./media-router"
