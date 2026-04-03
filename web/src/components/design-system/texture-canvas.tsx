import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

function TextureCanvas({ className, ...props }: ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "ds-texture-canvas relative isolate min-h-dvh overflow-hidden bg-background text-foreground",
        className
      )}
      {...props}
    />
  );
}

export { TextureCanvas };
