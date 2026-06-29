import { Effect } from "effect"
import type { ScannedFile } from "./scanner"

// ---------------------------------------------------------------------------
// Symbol types
// ---------------------------------------------------------------------------

export interface Symbol {
  /** Symbol name */
  name: string
  /** Kind: function, class, interface, type, enum, const, method, import */
  kind: string
  /** Line number (1-based) */
  line: number
}

export interface FileSymbols {
  /** File path */
  file: string
  /** Symbols exported/defined in this file */
  exports: Symbol[]
  /** Import targets (module names or relative paths this file imports from) */
  imports: string[]
  /** Whether this is likely a test file */
  isTest: boolean
}

// ---------------------------------------------------------------------------
// Language-specific extractors
// ---------------------------------------------------------------------------

interface Extractor {
  /** Language name */
  name: string
  /** File extensions this extractor handles */
  exts: string[]
  /** Extract exports and imports from file content */
  extract(content: string): { exports: Symbol[]; imports: string[] }
}

const EXTRACTORS: Extractor[] = [
  // TypeScript / JavaScript
  {
    name: "typescript",
    exts: [".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"],
    extract(content: string) {
      const exports: Symbol[] = []
      const imports: string[] = []

      const lines = content.split("\n")

      for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        const trimmed = line.trim()
        const lineNum = i + 1

        // Import statements → dependency edges
        const importMatch = trimmed.match(
          /^import\s+(?:\{[^}]*\}\s+from\s+)?['"]([^'"]+)['"]|^import\s+(?:\w+\s+from\s+)?['"]([^'"]+)['"]/,
        )
        if (importMatch) {
          const module = (importMatch[1] || importMatch[2] || "").replace(/^\.\.?\/+/, "")
          if (module) imports.push(module)
          continue
        }

        // `require(...)` calls
        const requireMatch = trimmed.match(/require\s*\(\s*['"]([^'"]+)['"]\s*\)/)
        if (requireMatch) {
          const module = requireMatch[1].replace(/^\.\.?\/+/, "")
          if (module) imports.push(module)
          continue
        }

        // Export declarations
        const exportMatch = trimmed.match(
          /^export\s+(?:default\s+)?(?:function|class|interface|type|enum|const|let|var|async\s+function)\s+(\w+)/,
        )
        if (exportMatch) {
          exports.push({ name: exportMatch[1], kind: exportMatch[0].includes("function") ? "function" : "class", line: lineNum })
          continue
        }

        // Function/class declarations (non-exported, top-level)
        const declMatch = trimmed.match(/^(?:function|class|interface|type|enum)\s+(\w+)/)
        if (declMatch && !trimmed.startsWith("//") && !trimmed.startsWith("/*")) {
          const kind = declMatch[0].startsWith("function") ? "function"
            : declMatch[0].startsWith("class") ? "class"
            : declMatch[0].startsWith("interface") ? "interface"
            : declMatch[0].startsWith("type") ? "type"
            : "enum"
          exports.push({ name: declMatch[1], kind, line: lineNum })
        }
      }

      return { exports, imports }
    },
  },

  // Python
  {
    name: "python",
    exts: [".py"],
    extract(content: string) {
      const exports: Symbol[] = []
      const imports: string[] = []
      const lines = content.split("\n")

      for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        const trimmed = line.trim()
        const lineNum = i + 1

        // Import statements
        const importMatch = trimmed.match(/^(?:from\s+(\S+)\s+)?import\s+(\S+)/)
        if (importMatch) {
          imports.push(importMatch[1] || importMatch[2])
          continue
        }

        // Function/class definitions
        const defMatch = trimmed.match(/^(?:async\s+)?(?:def|class)\s+(\w+)/)
        if (defMatch && !trimmed.startsWith("#")) {
          const kind = defMatch[0].includes("class") ? "class" : "function"
          exports.push({ name: defMatch[1], kind, line: lineNum })
        }
      }

      return { exports, imports }
    },
  },

  // Rust
  {
    name: "rust",
    exts: [".rs"],
    extract(content: string) {
      const exports: Symbol[] = []
      const imports: string[] = []
      const lines = content.split("\n")

      for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        const trimmed = line.trim()
        const lineNum = i + 1

        // `use` statements
        const useMatch = trimmed.match(/^use\s+(?:\S+(?:::))?(\S+)/)
        if (useMatch) {
          imports.push(useMatch[1])
          continue
        }

        // `mod` declarations
        const modMatch = trimmed.match(/^pub\s+(?:mod|fn|struct|enum|trait|type|const|unsafe\s+fn|async\s+fn)\s+(\w+)/)
        if (modMatch) {
          exports.push({ name: modMatch[1], kind: "pub", line: lineNum })
          continue
        }

        // `fn`, `struct`, `enum`, `trait` 
        const fnMatch = trimmed.match(/^(?:fn|struct|enum|trait|type)\s+(\w+)/)
        if (fnMatch && !trimmed.startsWith("//")) {
          exports.push({ name: fnMatch[1], kind: fnMatch[0].startsWith("fn") ? "function" : "struct", line: lineNum })
        }
      }

      return { exports, imports }
    },
  },

  // Go
  {
    name: "go",
    exts: [".go"],
    extract(content: string) {
      const exports: Symbol[] = []
      const imports: string[] = []
      const lines = content.split("\n")

      let inImportBlock = false

      for (let i = 0; i < lines.length; i++) {
        const line = lines[i]
        const trimmed = line.trim()
        const lineNum = i + 1

        // Import statements
        if (trimmed.startsWith('import "')) {
          const modMatch = trimmed.match(/^import\s+"([^"]+)"/)
          if (modMatch) imports.push(modMatch[1])
          continue
        }
        if (trimmed === "import (") { inImportBlock = true; continue }
        if (inImportBlock) {
          if (trimmed === ")") { inImportBlock = false; continue }
          const modMatch = trimmed.match(/^\s*"([^"]+)"/)
          if (modMatch) imports.push(modMatch[1])
          continue
        }

        // Exported functions/types (capitalized)
        const fnMatch = trimmed.match(/^func\s+(?:\([^)]*\)\s+)?(\w[A-Za-z0-9]*)\s*\(/)
        if (fnMatch && fnMatch[1][0] >= "A" && fnMatch[1][0] <= "Z") {
          exports.push({ name: fnMatch[1], kind: "function", line: lineNum })
          continue
        }

        const typeMatch = trimmed.match(/^type\s+(\w[A-Za-z0-9]*)\s+(?:struct|interface)/)
        if (typeMatch && typeMatch[1][0] >= "A" && typeMatch[1][0] <= "Z") {
          exports.push({ name: typeMatch[1], kind: "type", line: lineNum })
        }
      }

      return { exports, imports }
    },
  },
]

// ---------------------------------------------------------------------------
// sg outline integration
// ---------------------------------------------------------------------------

/**
 * Run `sg outline` on a file and parse the JSON output.
 * Returns null on any error (sg not found, non-zero exit, parse failure).
 */
function runSgOutline(filePath: string) {
  return Effect.promise(() =>
    (async () => {
      // @ts-ignore - Bun API available at runtime
      const process = Bun.spawn(["sg", "outline", filePath], { stdout: "pipe", stderr: "pipe" })
      const output = await new Response(process.stdout).text()
      const exitCode = await process.exited
      if (exitCode !== 0) {
        throw new Error(`sg outline exited with code ${exitCode}`)
      }
      const items = JSON.parse(output) as Array<{
        role: string
        name: string
        kind: string
        line_range: { start: number }
      }>
      const exports: Array<{ name: string; kind: string; line: number }> = []
      const imports: string[] = []
      for (const item of items) {
        if (item.role === "import") {
          imports.push(item.name)
        } else if (item.role === "export" || item.role === "definition") {
          exports.push({ name: item.name, kind: item.kind, line: item.line_range.start })
        }
      }
      return { exports, imports }
    })(),
  ).pipe(Effect.catch(() => Effect.succeed(null)))
}

/**
 * Extract symbols from a scanned file using ast-grep's `sg outline`.
 * Falls back to regex-based extraction if sg is unavailable.
 */
export async function extractSymbolsSg(file: ScannedFile): Promise<FileSymbols | null> {
  try {
    const sgResult = await Effect.runPromise(runSgOutline(file.path))
    if (sgResult) {
      const isTest = file.path.includes(".test.") ||
        file.path.includes(".spec.") ||
        file.path.includes("__tests__") ||
        file.path.includes("test_") ||
        file.path.endsWith("_test.go")
      return {
        file: file.path,
        exports: sgResult.exports,
        imports: sgResult.imports,
        isTest,
      }
    }
  } catch {
    // sg failed, fall through to regex
  }
  return extractSymbols(file)
}
// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Extract symbols and imports from a scanned file.
 * Returns null if no extractor matches the file extension.
 */
export function extractSymbols(file: ScannedFile): FileSymbols | null {
  const extractor = EXTRACTORS.find((e) => e.exts.includes(file.ext))
  if (!extractor) return null

  const { exports, imports } = extractor.extract(file.content)

  const isTest = file.path.includes(".test.") ||
    file.path.includes(".spec.") ||
    file.path.includes("__tests__") ||
    file.path.includes("test_") ||
    file.path.endsWith("_test.go")

  return {
    file: file.path,
    exports,
    imports,
    isTest,
  }
}
