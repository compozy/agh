"use client";

import { Button } from "@agh/ui";
import { useCopyButton } from "fumadocs-ui/utils/use-copy-button";
import { Check, Copy } from "lucide-react";
import { useState } from "react";

const cache = new Map<string, string>();

export interface LLMCopyButtonProps {
  markdownUrl: string;
}

export function LLMCopyButton({ markdownUrl }: LLMCopyButtonProps) {
  const [copyPending, setCopyPending] = useState(false);
  const [checked, onClick] = useCopyButton(async () => {
    const cached = cache.get(markdownUrl);
    if (cached) {
      await navigator.clipboard.writeText(cached);
      return;
    }
    setCopyPending(true);
    try {
      const response = await fetch(markdownUrl);
      const content = await response.text();
      cache.set(markdownUrl, content);
      await navigator.clipboard.writeText(content);
    } finally {
      setCopyPending(false);
    }
  });

  return (
    <Button disabled={copyPending} size="sm" variant="outline" onClick={onClick}>
      {checked ? <Check aria-hidden /> : <Copy aria-hidden />}
      Copy as Markdown
    </Button>
  );
}
