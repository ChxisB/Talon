/**
 * Minimal YAML-subset frontmatter parser/serializer for artifact markdown files.
 *
 * Supports only the shapes this project emits:
 *   - `---` delimited block at the start of the file
 *   - `key: value` scalar lines (bare or quoted with "..." or '...')
 *   - inline list values: `key: [a, b, c]`
 *   - block list values: a `key:` line followed by `- item` lines
 *
 * It deliberately does NOT support arbitrary YAML (nested maps, anchors,
 * multi-line strings, etc.) to stay dependency-free and ship in the single
 * binary build. Unknown keys are preserved round-trip as opaque scalars/lists.
 *
 * Comments are not supported: a leading `# ` would make a line a comment, but
 * inline ` # ...` is treated as part of the value (so values containing `#`,
 * such as colors or URLs, are safe). The serializer quotes values that need it.
 */

export type FrontmatterValue = string | string[]
export type Frontmatter = Record<string, FrontmatterValue>

export type ParsedDocument = {
  frontmatter: Frontmatter
  /** The document body (everything after the closing `---`), with the leading newline trimmed. */
  body: string
}

const DELIM = /^---\r?\n/
const BLOCK = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/

/** Parse a markdown document into frontmatter + body. A document without a leading `---` block yields an empty frontmatter. */
export function parseFrontmatter(content: string): ParsedDocument {
  if (!DELIM.test(content)) return { frontmatter: {}, body: content }
  const match = BLOCK.exec(content)
  if (!match) return { frontmatter: {}, body: content }
  const [, block, body] = match
  // Strip the blank separator line after the closing fence and the file's
  // trailing newline so the body round-trips exactly: a body written as
  // `serializeFrontmatter(fm, "x")` parses back to `"x"`, not `"\nx\n"`. This
  // keeps body_hash consistent between create/update and reindex.
  return { frontmatter: parseBlock(block!), body: body!.replace(/^\n+/, "").replace(/\n+$/, "") }
}

/** Serialize frontmatter + body back into a `---`-delimited markdown document. */
export function serializeFrontmatter(frontmatter: Frontmatter, body: string): string {
  const lines = ["---"]
  for (const [key, value] of Object.entries(frontmatter)) {
    if (value === undefined) continue
    if (Array.isArray(value)) {
      lines.push(value.length ? `${key}: [${value.map(quote).join(", ")}]` : `${key}: []`)
    } else if (value === "") {
      lines.push(`${key}: ""`)
    } else {
      lines.push(`${key}: ${quote(value)}`)
    }
  }
  lines.push("---")
  // One blank line between the frontmatter fence and the body, matching the
  // canonical artifact format. Trailing newline so the file is POSIX-clean.
  return `${lines.join("\n")}\n\n${body.replace(/^\n+/, "")}\n`
}

function parseBlock(block: string): Frontmatter {
  const frontmatter: Frontmatter = {}
  let lastKey: string | null = null
  for (const raw of block.split(/\r?\n/)) {
    if (raw.trim() === "") continue
    // Block list item: append to the most recent key.
    if (raw.trimStart().startsWith("- ")) {
      if (lastKey) {
        const item = unquote(raw.trim().slice(2).trim())
        const existing = frontmatter[lastKey]
        if (Array.isArray(existing)) existing.push(item)
        else if (existing !== undefined) frontmatter[lastKey] = [existing, item]
        else frontmatter[lastKey] = [item]
      }
      continue
    }
    const idx = raw.indexOf(":")
    if (idx === -1) continue
    const key = raw.slice(0, idx).trim()
    const rawValue = raw.slice(idx + 1).trim()
    lastKey = key
    if (rawValue === "") {
      // Could be the start of a block list; leave undefined for now, block items
      // will populate it. Emit nothing as a scalar.
      continue
    }
    if (rawValue.startsWith("[") && rawValue.endsWith("]")) {
      const inner = rawValue.slice(1, -1).trim()
      frontmatter[key] = inner ? inner.split(",").map((s) => unquote(s.trim())) : []
      continue
    }
    frontmatter[key] = unquote(rawValue)
  }
  return frontmatter
}

const MUST_QUOTE = /[\s:#"'\[\],]/

function quote(value: string): string {
  if (value === "") return '""'
  if (MUST_QUOTE.test(value)) return `"${value.replaceAll("\\", "\\\\").replaceAll('"', '\\"')}"`
  return value
}

function unquote(value: string): string {
  if (value.length >= 2 && value.startsWith('"') && value.endsWith('"')) {
    return value.slice(1, -1).replaceAll('\\"', '"').replaceAll("\\\\", "\\")
  }
  if (value.length >= 2 && value.startsWith("'") && value.endsWith("'")) {
    return value.slice(1, -1)
  }
  return value
}
