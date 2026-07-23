import { z } from "zod";

import {
  windowManagerClientViewSchema,
  windowManagerErrorSchema,
  windowManagerEventSchema,
  windowManagerSnapshotSchema,
} from "./window-manager-schemas";
import type { WindowManagerStreamFrame } from "./window-manager-types";

const safeRevisionSchema = z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const identifierSchema = z.string().trim().min(1);

const rawWindowManagerStreamFrameSchema = z.discriminatedUnion("type", [
  z.strictObject({
    type: z.literal("snapshot"),
    workspace_id: identifierSchema,
    revision: safeRevisionSchema,
    snapshot: windowManagerSnapshotSchema,
    client: windowManagerClientViewSchema.optional(),
  }),
  z.strictObject({
    type: z.literal("event"),
    workspace_id: identifierSchema,
    revision: safeRevisionSchema,
    event: windowManagerEventSchema,
  }),
  z
    .strictObject({
      type: z.literal("client"),
      workspace_id: identifierSchema,
      revision: safeRevisionSchema,
      client: windowManagerClientViewSchema,
    })
    .refine(frame => frame.revision === frame.client.presentationRevision, {
      message: "Client frame revision must match presentation_revision.",
      path: ["revision"],
    }),
  z.strictObject({
    type: z.literal("error"),
    error: windowManagerErrorSchema,
  }),
]);

export const windowManagerStreamFrameSchema = rawWindowManagerStreamFrameSchema.transform(
  (frame): WindowManagerStreamFrame => {
    if (frame.type === "error") return { type: frame.type, error: frame.error };
    if (frame.type === "snapshot") {
      return {
        type: frame.type,
        workspaceId: frame.workspace_id,
        revision: frame.revision,
        snapshot: frame.snapshot,
        client: frame.client ?? null,
      };
    }
    if (frame.type === "client") {
      return {
        type: frame.type,
        workspaceId: frame.workspace_id,
        revision: frame.revision,
        client: frame.client,
      };
    }
    return {
      type: frame.type,
      workspaceId: frame.workspace_id,
      revision: frame.revision,
      event: frame.event,
    };
  }
);

export function parseWindowManagerStreamFrame(value: unknown): WindowManagerStreamFrame {
  return windowManagerStreamFrameSchema.parse(value);
}
