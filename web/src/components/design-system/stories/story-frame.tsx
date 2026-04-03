import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

import { TextureCanvas } from "../texture-canvas";

function StoryFrame({ children, className, ...props }: ComponentProps<"div">) {
  return (
    <TextureCanvas className="min-h-[22rem]">
      <div
        className={cn("relative z-10 w-[min(100vw-2rem,72rem)] p-6 text-foreground", className)}
        {...props}
      >
        {children}
      </div>
    </TextureCanvas>
  );
}

function TexturedStoryFrame({ children, className, ...props }: ComponentProps<"div">) {
  return (
    <TextureCanvas>
      <div
        className={cn(
          "relative z-10 min-h-[24rem] w-[min(100vw,84rem)] px-6 py-8 text-foreground",
          className
        )}
        {...props}
      >
        {children}
      </div>
    </TextureCanvas>
  );
}

export { StoryFrame, TexturedStoryFrame };
