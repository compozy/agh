import type { ComponentProps } from "react";

import { cn } from "@/lib/utils";

function PageContent({ className, ...props }: ComponentProps<"main">) {
  return (
    <main
      className={cn(
        "relative z-10 mx-auto flex min-h-dvh w-full max-w-6xl flex-col gap-6 px-4 py-6 sm:gap-8 sm:px-6 sm:py-10 lg:px-8 lg:py-12",
        className
      )}
      {...props}
    />
  );
}

export { PageContent };
