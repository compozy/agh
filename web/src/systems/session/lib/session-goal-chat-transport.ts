import type { SessionGoalCommandResult } from "../types";

interface GoalAwareFetchOptions {
  fetch?: typeof globalThis.fetch;
  onRequest?: () => void;
  onResult: (result: SessionGoalCommandResult, requestText: string | null) => void;
}

const GOAL_COMMAND_FAILURE_GUIDANCE = {
  goal_judge_unavailable:
    "Goal judge is not configured. Set loops.defaults.delivery.model_defaults.judge or use a session created with an explicit model, then retry.",
  goal_objective_required: "Add an objective after /goal, then try again.",
  goal_objective_too_large: "Shorten the Goal objective, then try again.",
} as const;

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

export function describeGoalCommandFailure(
  reasonCode: SessionGoalCommandResult["reason_code"]
): string | null {
  if (reasonCode && reasonCode in GOAL_COMMAND_FAILURE_GUIDANCE) {
    return GOAL_COMMAND_FAILURE_GUIDANCE[reasonCode as keyof typeof GOAL_COMMAND_FAILURE_GUIDANCE];
  }
  return null;
}

export function isGoalCommandFailureGuidance(message: string): boolean {
  return Object.values(GOAL_COMMAND_FAILURE_GUIDANCE).some(guidance => guidance === message);
}

function goalCommandFailureMessage(reasonCode: SessionGoalCommandResult["reason_code"]): string {
  return describeGoalCommandFailure(reasonCode) ?? reasonCode ?? "Goal command failed";
}

export function createGoalAwareFetch({
  fetch = globalThis.fetch.bind(globalThis),
  onRequest,
  onResult,
}: GoalAwareFetchOptions): typeof globalThis.fetch {
  let sequence = 0;
  return async (input, init) => {
    onRequest?.();
    const submittedText = requestText(init);
    const response = await fetch(input, init);
    if (!isJsonContentType(response.headers.get("content-type"))) {
      return response;
    }

    let payload: unknown;
    try {
      payload = await response.clone().json();
    } catch {
      return response;
    }
    if (!isGoalCommandResult(payload)) {
      return response;
    }

    onResult(payload, submittedText);
    if (!response.ok) {
      return new Response(goalCommandFailureMessage(payload.reason_code), {
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
