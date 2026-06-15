import { NextRequest, NextResponse } from "next/server";
import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";

// Strip chain-of-thought reasoning from models
function stripThinkingTrace(text: string): string {
  if (!text) return text;
  
  // Remove <think>...</think> blocks (MiniMax, some Qwen)
  let cleaned = text.replace(/<think>[\s\S]*?<\/think>/gi, "").trim();
  
  // Remove leading "Thinking." or "Thinking:" blocks (DeepSeek)
  cleaned = cleaned.replace(/^Thinking\.?\s*\n*/i, "").trim();
  
  // Split into paragraphs
  const paragraphs = cleaned.split(/\n\s*\n/);
  
  // Filter out reasoning paragraphs: numbered lists, analysis steps, etc.
  const reasoningPattern = /^\s*(?:\d+[\.\)]|[-*]\s*(?:Analyze|Determine|Formulate|Draft|Refine|Self-Correction|Let's|Key|Goal|Approach))/i;
  const result = paragraphs.filter((p) => !reasoningPattern.test(p.trim()));
  
  // If all paragraphs were filtered, return the last paragraph
  if (result.length === 0 && paragraphs.length > 0) {
    return paragraphs[paragraphs.length - 1].trim();
  }
  
  // If most content was filtered, return just the final result paragraphs
  if (result.length < paragraphs.length / 2 && result.length > 0) {
    return result.join("\n\n").trim();
  }
  
  return cleaned;
}

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const AGENT_URL = process.env.AGENT_INTERNAL_URL || "http://localhost:8082";

// Read model from .env config file (reflects dashboard settings)
function getConfiguredModel(): string {
  try {
    const envPath = path.join(os.homedir(), ".talon", ".env");
    if (!existsSync(envPath)) return "";
    const content = readFileSync(envPath, "utf8");
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (trimmed.startsWith("MODEL=")) {
        return trimmed.slice(6).replace(/^["']|["']$/g, "");
      }
    }
  } catch {}
  return "";
}

export async function POST(req: NextRequest) {
  const { message, history } = await req.json();
  const configuredModel = getConfiguredModel();

  // Build messages array in Anthropic format
  const messages: { role: string; content: { type: string; text: string }[] }[] = [];

  if (history && Array.isArray(history)) {
    for (const msg of history) {
      messages.push({
        role: msg.role,
        content: [{ type: "text", text: msg.text }],
      });
    }
  }

  messages.push({
    role: "user",
    content: [{ type: "text", text: message }],
  });

  try {
    const res = await fetch(`${AGENT_URL}/v1/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        model: configuredModel || "",
        messages,
        max_tokens: 4096,
        stream: true,
      }),
      signal: AbortSignal.timeout(60000),
    });

    if (!res.ok) {
      const err = await res.text().catch(() => "unknown error");
      return NextResponse.json({ error: `Agent error (${res.status}): ${err}` }, { status: 502 });
    }

    // Parse SSE stream
    const reader = res.body?.getReader();
    if (!reader) {
      return NextResponse.json({ error: "No response body" }, { status: 502 });
    }

    const decoder = new TextDecoder();
    let buffer = "";
    let fullText = "";
    let model = "";
    let inputTokens = 0;
    let outputTokens = 0;

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          try {
            const data = JSON.parse(line.slice(6));
            if (data.type === "content_block_delta" && data.delta?.type === "text_delta") {
              fullText += data.delta.text;
            }
            if (data.type === "message_start" && data.message) {
              model = data.message.model || "";
            }
            if (data.type === "message_delta" && data.usage) {
              outputTokens = data.usage.output_tokens || 0;
            }
          } catch {
            // skip malformed JSON
          }
        }
      }
    }

    // Strip thinking/reasoning traces from DeepSeek and similar models
    const cleanText = stripThinkingTrace(fullText);

    return NextResponse.json({
      content: cleanText || "(empty response — check your model configuration)",
      model: model || configuredModel,
      usage: { input_tokens: inputTokens, output_tokens: outputTokens },
    });
  } catch (err) {
    return NextResponse.json(
      { error: err instanceof Error ? err.message : "Request failed" },
      { status: 502 }
    );
  }
}
