export type ModelRef = {
  providerID: string
  modelID: string
}

const SIMPLE_CHAT_PATTERNS = [
  /^(?:hi|hey|hello|yo|howdy|sup)(?:\s+there)?[!.?]*$/i,
  /^good\s+(?:morning|afternoon|evening)[!.?]*$/i,
  /^how\s+are\s+you[?.!]*$/i,
  /^(?:what\s+time\s+is\s+it|what(?:'s| is)\s+the\s+time)(?:\s+now|\s+in\s+.+)?[?.!]*$/i,
  /^what(?:'s| is)\s+the\s+weather(?:\s+like)?(?:\s+(?:today|right\s+now|in\s+.+))?[?.!]*$/i,
  /^(?:thanks|thank\s+you)[!.?]*$/i,
]

const CODE_OR_WORK_PATTERN =
  /\b(?:file|code|repo|repository|implement|fix|debug|bug|test|build|compile|function|class|module|api|endpoint|stacktrace|error)\b/i

export function shouldUseLightweightTurn(input: {
  step: number
  text: string
  hasMediaAttachments: boolean
  hasStructuredOutput: boolean
  hasSubtasks: boolean
}): boolean {
  if (input.step !== 1) return false
  if (input.hasMediaAttachments || input.hasStructuredOutput || input.hasSubtasks) return false

  const text = input.text.trim()
  if (!text) return false
  if (text.length > 120) return false
  if (text.includes("\n")) return false
  if (CODE_OR_WORK_PATTERN.test(text)) return false

  return SIMPLE_CHAT_PATTERNS.some((pattern) => pattern.test(text))
}

export function shouldBypassVisionPreanalysis(input: {
  codingModel: ModelRef | undefined
  visionModel: ModelRef | undefined
}): boolean {
  if (!input.codingModel || !input.visionModel) return false
  return (
    input.codingModel.providerID === input.visionModel.providerID &&
    input.codingModel.modelID === input.visionModel.modelID
  )
}
