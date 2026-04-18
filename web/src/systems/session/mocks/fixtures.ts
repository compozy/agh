import type {
  PermissionRequest,
  SessionApprovalResponse,
  SessionEventPayload,
  SessionPayload,
  TranscriptMessage,
  TurnHistoryPayload,
  UIMessage,
} from "@/systems/session/types";

export const sessionFixtures: SessionPayload[] = [
  {
    id: "sess-storybook",
    name: "Storybook rollout",
    agent_name: "codex-agent",
    workspace_id: "ws_storybook",
    workspace_path: "/workspaces/agh2",
    state: "active",
    channel: "storybook",
    created_at: "2026-04-17T16:00:00Z",
    updated_at: "2026-04-17T18:10:00Z",
    acp_caps: {
      supports_load_session: true,
      supported_models: ["gpt-5.4"],
      supported_modes: ["chat"],
    },
  },
  {
    id: "sess-reviewer",
    name: "Review lane",
    agent_name: "claude-agent",
    workspace_id: "ws_storybook",
    workspace_path: "/workspaces/agh2",
    state: "stopped",
    created_at: "2026-04-17T15:40:00Z",
    updated_at: "2026-04-17T17:10:00Z",
  },
];

export const primarySessionFixture: SessionPayload = sessionFixtures[0];

export const sessionEventsFixture: SessionEventPayload[] = [
  {
    id: "event_001",
    agent_name: primarySessionFixture.agent_name,
    content: {
      text: "Planning Storybook rollout.",
    },
    sequence: 1,
    session_id: primarySessionFixture.id,
    timestamp: "2026-04-17T16:01:00Z",
    turn_id: "turn_001",
    type: "message.created",
    workspace_id: primarySessionFixture.workspace_id,
    workspace_path: primarySessionFixture.workspace_path,
  },
  {
    id: "event_002",
    agent_name: primarySessionFixture.agent_name,
    content: {
      tool_name: "Read",
      file_path: ".compozy/tasks/storybook-stories/_techspec.md",
    },
    sequence: 2,
    session_id: primarySessionFixture.id,
    timestamp: "2026-04-17T16:02:00Z",
    turn_id: "turn_001",
    type: "tool.called",
    workspace_id: primarySessionFixture.workspace_id,
    workspace_path: primarySessionFixture.workspace_path,
  },
];

export const sessionHistoryFixture: TurnHistoryPayload[] = [
  {
    turn_id: "turn_001",
    events: sessionEventsFixture,
  },
];

export const bashToolMessageFixture: UIMessage = {
  id: "tool_bash",
  role: "tool_call",
  content: "",
  toolName: "Bash",
  toolInput: {
    command: "bun run --cwd web build-storybook",
  },
  toolResult: {
    stdout: "Build started\nBuild finished successfully\n",
  },
  timestamp: Date.parse("2026-04-17T16:04:00Z"),
};

export const runningBashToolMessageFixture: UIMessage = {
  ...bashToolMessageFixture,
  id: "tool_bash_running",
  toolResult: undefined,
};

export const longBashToolMessageFixture: UIMessage = {
  ...bashToolMessageFixture,
  id: "tool_bash_long",
  toolResult: {
    stdout: Array.from({ length: 240 }, (_, index) => `stdout line ${index + 1}`).join("\n"),
  },
};

export const errorToolMessageFixture: UIMessage = {
  ...bashToolMessageFixture,
  id: "tool_bash_error",
  toolError: true,
  toolResult: {
    stderr: "storybook build failed\nexit status 1\n",
    error: "Command failed with exit status 1",
  },
};

export const editToolMessageFixture: UIMessage = {
  id: "tool_edit",
  role: "tool_call",
  content: "",
  toolName: "Edit",
  toolInput: {
    file_path: "web/.storybook/preview.ts",
    old_string: "loaders: storybookLoaders,",
    new_string: "loaders: storybookLoaders,\nparameters: { msw: { handlers: [] } },",
  },
  toolResult: {
    content: "Applied patch successfully.",
  },
  timestamp: Date.parse("2026-04-17T16:05:00Z"),
};

export const multiHunkEditToolMessageFixture: UIMessage = {
  ...editToolMessageFixture,
  id: "tool_edit_multi_hunk",
  toolInput: {
    file_path: "web/src/systems/session/components/stories/chat-view.stories.tsx",
    old_string: [
      "@@ -12,7 +12,7 @@",
      "-export const Default = {};",
      "",
      "@@ -28,4 +28,6 @@",
      "-export const Streaming = {};",
    ].join("\n"),
    new_string: [
      "@@ -12,7 +12,7 @@",
      "+export const Default = { args: { state: 'default' } };",
      "",
      "@@ -28,4 +28,6 @@",
      "+export const Streaming = { args: { state: 'streaming' } };",
    ].join("\n"),
  },
};

export const readToolMessageFixture: UIMessage = {
  id: "tool_read",
  role: "tool_call",
  content: "",
  toolName: "Read",
  toolInput: {
    file_path: ".compozy/tasks/storybook-stories/_techspec.md",
  },
  toolResult: {
    stdout: "# TechSpec\n\nStorybook rollout details...\n",
  },
  timestamp: Date.parse("2026-04-17T16:06:00Z"),
};

export const truncatedReadToolMessageFixture: UIMessage = {
  ...readToolMessageFixture,
  id: "tool_read_large",
  toolInput: {
    file_path: "web/src/generated/agh-openapi.d.ts",
  },
  toolResult: {
    stdout: Array.from({ length: 180 }, (_, index) => `type Line${index + 1} = string;`).join("\n"),
  },
};

export const searchToolMessageFixture: UIMessage = {
  id: "tool_search",
  role: "tool_call",
  content: "",
  toolName: "Grep",
  toolInput: {
    pattern: "storybook",
    glob: "**/*.tsx",
  },
  toolResult: {
    stdout:
      "web/src/components/ui/stories/dialog.stories.tsx\nweb/src/systems/session/components/chat-view.tsx",
  },
  timestamp: Date.parse("2026-04-17T16:07:00Z"),
};

export const emptySearchToolMessageFixture: UIMessage = {
  ...searchToolMessageFixture,
  id: "tool_search_empty",
  toolResult: {
    stdout: "",
  },
};

export const writeToolMessageFixture: UIMessage = {
  id: "tool_write",
  role: "tool_call",
  content: "",
  toolName: "Write",
  toolInput: {
    file_path: "web/src/systems/session/components/stories/chat-view.stories.tsx",
    content: "export const Default = {};",
  },
  toolResult: {
    content: "Created story file.",
  },
  timestamp: Date.parse("2026-04-17T16:08:00Z"),
};

export const overwriteWriteToolMessageFixture: UIMessage = {
  ...writeToolMessageFixture,
  id: "tool_write_overwrite",
  toolInput: {
    file_path: "web/src/systems/session/components/stories/chat-view.stories.tsx",
    content: [
      "// WARNING: overwriting existing story module",
      "export const Default = { args: { mode: 'overwrite' } };",
    ].join("\n"),
  },
};

export const genericToolMessageFixture: UIMessage = {
  id: "tool_generic",
  role: "tool_call",
  content: "",
  toolName: "Context7",
  toolInput: {
    library: "storybook",
    topic: "stories",
  },
  toolResult: {
    content: "Fetched docs excerpt.",
  },
  timestamp: Date.parse("2026-04-17T16:09:00Z"),
};

export const markdownFixture = `# Storybook rollout

- Finish the remaining system stories.
- Verify both Storybook instances.

\`\`\`ts
const status = "green";
\`\`\`

[ADR-003](https://example.com/adr-003)
`;

export const userMessageFixture: UIMessage = {
  id: "msg_user_001",
  role: "user",
  content: "Finish the remaining Storybook tasks.",
  timestamp: Date.parse("2026-04-17T16:00:00Z"),
};

export const assistantMessageFixture: UIMessage = {
  id: "msg_assistant_001",
  role: "assistant",
  content: "I am wiring the system mocks and stories now.",
  thinking: "Need typed fixtures first so stories stay truthful.",
  thinkingComplete: true,
  timestamp: Date.parse("2026-04-17T16:01:00Z"),
};

export const streamingAssistantMessageFixture: UIMessage = {
  id: "msg_assistant_streaming",
  role: "assistant",
  content: "Streaming partial answer…",
  isStreaming: true,
  timestamp: Date.parse("2026-04-17T16:11:00Z"),
};

export const systemMessageFixture: UIMessage = {
  id: "msg_system_001",
  role: "system",
  content: "System notice: permission required to run a shell command.",
  timestamp: Date.parse("2026-04-17T16:02:00Z"),
};

export const uiMessageFixtures: UIMessage[] = [
  userMessageFixture,
  assistantMessageFixture,
  bashToolMessageFixture,
  {
    ...bashToolMessageFixture,
    id: "tool_bash_result",
    role: "tool_result",
  },
  {
    id: "msg_assistant_002",
    role: "assistant",
    content: markdownFixture,
    timestamp: Date.parse("2026-04-17T16:10:00Z"),
  },
];

export const sessionTranscriptFixture: TranscriptMessage[] = [
  {
    id: "transcript_user_001",
    role: "user",
    content: "Finish the remaining Storybook tasks.",
    thinking_complete: false,
    timestamp: "2026-04-17T16:00:00Z",
    tool_error: false,
  },
  {
    id: "transcript_assistant_001",
    role: "assistant",
    content: markdownFixture,
    thinking: "Need typed fixtures first so stories stay truthful.",
    thinking_complete: true,
    timestamp: "2026-04-17T16:10:00Z",
    tool_error: false,
  },
  {
    id: "transcript_tool_001",
    role: "tool_call",
    content: "",
    thinking_complete: false,
    timestamp: "2026-04-17T16:04:00Z",
    tool_error: false,
    tool_name: "Bash",
    tool_input: bashToolMessageFixture.toolInput,
    tool_result: {
      stdout: "Build started\nBuild finished successfully\n",
    },
  },
];

export const sessionApprovalFixture: SessionApprovalResponse = {
  status: "approved",
};

export const permissionRequestFixture: PermissionRequest = {
  requestId: "perm_storybook_001",
  toolName: "Bash",
  toolInput: {
    command: "make web-typecheck",
  },
  action: "execute",
  resource: "make web-typecheck",
};
