import type { SessionGoalCommandResult } from "../types";

interface GoalAwareFetchOptions {
  fetch?: typeof globalThis.fetch;
  onResult: (result: SessionGoalCommandResult, requestText: string | null) => void;
}

function requestText(init?: RequestInit): string | null {
  if (typeof init?.body !== "string") return null;
  try {
    const body: unknown = JSON.parse(init.body);
    if (typeof body !== "object" || body === null) return null;
    if ("message" in body && typeof body.message === "string") return body.message;
    if (!("messages" in body) || !Array.isArray(body.messages)) return null;
    const last = body.messages.at(-1);
    if (
      typeof last !== "object" ||
      last === null ||
      !("parts" in last) ||
      !Array.isArray(last.parts)
    ) {
      return null;
    }
    return (
      (last.parts as unknown[])
        .filter(
          (part): part is Record<string, unknown> =>
            typeof part === "object" && part !== null && "type" in part && part.type === "text"
        )
        .map(part => (typeof part.text === "string" ? part.text : ""))
        .join("\n")
        .trim() || null
    );
  } catch {
    return null;
  }
}

function isJsonContentType(value: string | null): boolean {
  return value?.toLowerCase().includes("json") ?? false;
}

function isGoalCommandResult(value: unknown): value is SessionGoalCommandResult {
  return (
    typeof value === "object" &&
    value !== null &&
    "outcome" in value &&
    typeof value.outcome === "string"
  );
}

function completionBody(messageId: string): string {
  return [
    `data: ${JSON.stringify({ type: "start", messageId })}\n\n`,
    `data: ${JSON.stringify({ type: "finish", finishReason: "stop" })}\n\n`,
  ].join("");
}

export function createGoalAwareFetch({
  fetch = globalThis.fetch.bind(globalThis),
  onResult,
}: GoalAwareFetchOptions): typeof globalThis.fetch {
  let sequence = 0;
  return async (input, init) => {
    const submittedText = requestText(init);
    const response = await fetch(input, init);
    if (!isJsonContentType(response.headers.get("content-type"))) {
      return response;
    }

    const payload: unknown = await response.clone().json();
    if (!isGoalCommandResult(payload)) {
      return response;
    }

    onResult(payload, submittedText);
    if (!response.ok) {
      return new Response(payload.reason_code ?? "Goal command failed", {
        status: response.status,
        statusText: response.statusText,
        headers: { "content-type": "text/plain; charset=utf-8" },
      });
    }

    sequence += 1;
    return new Response(completionBody(`goal-command-${sequence}`), {
      status: response.status,
      statusText: response.statusText,
      headers: { "content-type": "text/event-stream" },
    });
  };
}
