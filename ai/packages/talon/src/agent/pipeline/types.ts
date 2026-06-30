// ── Pipeline Types ──────────────────────────────────────────────────────

/**
 * The pipeline mode determines which stages are executed.
 * - fix:    explore → implement → test → review → signoff
 * - feature: research → plan → implement → test → review → signoff
 */
export type PipelineMode = "fix" | "feature"

/**
 * Individual stage within a pipeline run.
 */
export type StageType =
  | "explore"
  | "research"
  | "plan"
  | "implement"
  | "fix"
  | "test"
  | "review"
  | "signoff"

/**
 * A single stage execution in a pipeline.
 */
export interface Stage {
  /** Unique stage name within the pipeline */
  readonly name: StageType
  /** The subagent type to use for this stage */
  readonly agent: string
  /** The task description/prompt for this stage */
  readonly task: string
  /** Whether this stage depends on the previous one completing */
  readonly sequential: boolean
}

/**
 * Result of a single stage execution.
 */
export interface StageResult {
  readonly stage: StageType
  readonly agent: string
  readonly output: string
  readonly error?: string
  readonly duration: number
  readonly passed: boolean
}

/**
 * A quality gate that runs after a stage.
 */
export interface Gate {
  /** Name of the gate (e.g., "test", "review") */
  readonly name: string
  /** The subagent type to run the gate */
  readonly agent: string
  /** The task for the gate */
  readonly task: string
}

/**
 * Result of a quality gate check.
 */
export interface GateResult {
  readonly gate: string
  readonly passed: boolean
  readonly output: string
  readonly error?: string
  readonly details?: string
}

/**
 * Full pipeline execution result.
 */
export interface PipelineResult {
  readonly goal: string
  readonly mode: PipelineMode
  readonly stages: StageResult[]
  readonly gates: GateResult[]
  readonly loopbacks: number
  readonly signedOff: boolean
  readonly passed: boolean
  readonly summary: string
}

/**
 * Configuration for a pipeline run.
 */
export interface PipelineConfig {
  readonly maxLoopbacks: number
  readonly autoTest: boolean
  readonly autoReview: boolean
  readonly requireSignoff: boolean
}

export const defaultPipelineConfig: PipelineConfig = {
  maxLoopbacks: 3,
  autoTest: true,
  autoReview: true,
  requireSignoff: true,
}

/**
 * Intent classification result from the router.
 */
export interface IntentClassification {
  readonly mode: PipelineMode
  readonly confidence: number
  readonly summary: string
  readonly keyFindings: string[]
}
