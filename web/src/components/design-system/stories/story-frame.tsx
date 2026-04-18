import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

function StoryFrame({ children, className, ...props }: ComponentProps<"div">) {
  return (
    <div className="relative isolate min-h-[22rem] overflow-hidden bg-[color:var(--color-canvas)] text-[color:var(--color-text-primary)]">
      <div className={cn("relative z-10 w-[min(100vw-2rem,72rem)] p-6", className)} {...props}>
        {children}
      </div>
    </div>
  );
}

function TexturedStoryFrame({ children, className, ...props }: ComponentProps<"div">) {
  return (
    <div className="relative isolate min-h-dvh overflow-hidden bg-[color:var(--color-canvas)] text-[color:var(--color-text-primary)]">
      <div
        className={cn("relative z-10 min-h-[24rem] w-[min(100vw,84rem)] px-6 py-8", className)}
        {...props}
      >
        {children}
      </div>
    </div>
  );
}

export { StoryFrame, TexturedStoryFrame };
