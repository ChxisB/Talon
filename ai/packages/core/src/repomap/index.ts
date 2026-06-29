/**
 * Repo Map — codebase-aware context injection for the LLM.
 *
 * Builds a ranked map of the workspace: finds source files, extracts symbols,
 * builds an import dependency graph, and ranks files by importance.
 *
 * The ranked map is injected into the system prompt so the LLM always has
 * a prioritized view of the codebase without needing to `read` every file.
 *
 * Inspired by Aider's repomap.py (tree-sitter + PageRank on import graph).
 * This initial implementation uses regex-based extraction for speed and
 * simplicity; tree-sitter-based deep parsing can be layered on later.
 */

export { scanWorkspace, type ScannedFile } from "./scanner"
export { extractSymbols, extractSymbolsSg, type FileSymbols, type Symbol } from "./parser"
export { rankFiles } from "./graph"
export { buildRepoMap, buildRepoMapSg, formatRepoMap, type RepoMapResult } from "./map"
