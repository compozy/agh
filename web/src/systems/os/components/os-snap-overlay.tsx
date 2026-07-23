import { useWindowManagerGesturePreview } from "../hooks/use-window-manager-store";

function previewLabel(kind: string): string {
  if (kind === "zoom") return "Zoom";
  if (kind === "stack") return "Add to stack";
  if (kind === "insert") return "Insert";
  if (kind === "split") return "Split";
  return "Tile";
}

/** One ephemeral structural preview; it never mutates the authoritative snapshot. */
export function OsSnapOverlay() {
  const preview = useWindowManagerGesturePreview();
  if (preview === null) return null;
  return (
    <div
      aria-hidden="true"
      data-slot="window-manager-command-preview"
      className="pointer-events-none absolute z-50 rounded-md border border-accent bg-accent-tint shadow-inset-accent"
      style={{
        left: preview.rect.x,
        top: preview.rect.y,
        width: preview.rect.w,
        height: preview.rect.h,
      }}
    >
      <span className="absolute top-2 left-2 rounded-sm bg-elevated px-2 py-1 text-form-hint font-medium text-fg shadow-elevated">
        {previewLabel(preview.kind)}
      </span>
    </div>
  );
}
