import { Effect } from "effect"
import { scanWorkspace, type ScannedFile } from "./scanner"
import { extractSymbols, extractSymbolsSg, type FileSymbols } from "./parser"
import { rankFiles } from "./graph"

// ---------------------------------------------------------------------------
// Repo Map builder
// ---------------------------------------------------------------------------

export interface RepoMapConfig {
  /** Workspace directory to scan */
  directory: string
  /** Maximum number of source files to scan */
  maxFiles?: number
  /** Maximum entries in the ranked output */
  maxEntries?: number
}

export interface RepoMapResult {
  /** Ranked files with scores */
  ranked: { path: string; score: number; exports: number }[]
  /** All extracted symbols (for reference) */
  symbols: FileSymbols[]
  /** Total files scanned */
  totalFiles: number
  /** Total unique symbols found */
  totalSymbols: number
}

/**
 * Build a ranked repo map for the given workspace directory.
 * Scans source files, extracts symbols/imports, builds a dependency graph,
 * and ranks files by importance.
 */
export function buildRepoMap(config: RepoMapConfig): RepoMapResult {
  const { directory, maxFiles = 200, maxEntries = 20 } = config

  // 1. Scan workspace
  const files = scanWorkspace(directory, maxFiles)

  // 2. Extract symbols from each file
  const symbols: FileSymbols[] = []
  for (const file of files) {
    const extracted = extractSymbols(file)
    if (extracted) {
      symbols.push(extracted)
    }
  }

  // 3. Rank files by importance
  const ranked = rankFiles(symbols)

  // 4. Compute stats
  const totalSymbols = symbols.reduce((sum, s) => sum + s.exports.length, 0)

  return {
    ranked: ranked.slice(0, maxEntries),
    symbols,
    totalFiles: files.length,
    totalSymbols,
  }
}

/**
 * Build a ranked repo map using ast-grep's `sg outline` for AST-level symbol extraction.
 * Falls back to regex extraction per-file when sg is unavailable.
 */
export async function buildRepoMapSg(config: RepoMapConfig): Promise<RepoMapResult> {
  const { directory, maxFiles = 200, maxEntries = 20 } = config
  const files = scanWorkspace(directory, maxFiles)
  const symbols: FileSymbols[] = []
  for (const file of files) {
    try {
      const extracted = await extractSymbolsSg(file)
      if (extracted) {
        symbols.push(extracted)
      }
    } catch {
      const extracted = extractSymbols(file)
      if (extracted) {
        symbols.push(extracted)
      }
    }
  }
  const ranked = rankFiles(symbols)
  const totalSymbols = symbols.reduce((sum, s) => sum + s.exports.length, 0)
  return {
    ranked: ranked.slice(0, maxEntries),
    symbols,
    totalFiles: files.length,
    totalSymbols,
  }
}

/**
 * Format a repo map as a markdown block suitable for system prompt injection.
 */
export function formatRepoMap(result: RepoMapResult): string {
  if (result.ranked.length === 0) return ""

  const lines: string[] = [
    "<repo-map>",
    `This is a ranked map of the ${result.totalFiles} source files in the project.`,
    `Files are ordered by relevance (most referenced/central first). Key symbols are listed.`,
    "",
  ]

  for (const entry of result.ranked) {
    // Shorten path to relative
    const relPath = shortenPath(entry.path)
    const symbolEntry = result.symbols.find((s) => s.file === entry.path)
    const symbolList = symbolEntry
      ? symbolEntry.exports.slice(0, 5).map((s) => `${s.kind} ${s.name}`).join(", ")
      : ""

    lines.push(`- ${relPath}${symbolList ? `: ${symbolList}` : ""}`)
  }

  lines.push("</repo-map>")
  return lines.join("\n")
}

/**
 * Shorten an absolute path to a workspace-relative path.
 * Keeps the last 3 path components for context.
 */
function shortenPath(absPath: string): string {
  const parts = absPath.split("/")
  if (parts.length <= 4) return parts.slice(-3).join("/")
  return "..." + parts.slice(-3).join("/")
}
