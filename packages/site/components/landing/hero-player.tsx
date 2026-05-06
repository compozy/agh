"use client";

import { Player, Thumbnail } from "@remotion/player";
import { cn } from "@agh/ui/lib/utils";
import { HeroChatComposition, STATIC_FALLBACK_FRAME } from "@/remotion/hero/composition";
import {
  COMPOSITION_HEIGHT,
  COMPOSITION_WIDTH,
  DURATION_IN_FRAMES,
  FPS,
} from "@/remotion/hero/data";
import { useReducedMotion } from "./primitives/use-reduced-motion";

interface HeroPlayerProps {
  className?: string;
}

export function HeroPlayer({ className }: HeroPlayerProps) {
  const reduced = useReducedMotion();

  const containerClass = cn("relative mx-auto aspect-square w-full max-w-[560px]", className);
  const containerStyle = {
    background:
      "radial-gradient(circle at 50% 50%, color-mix(in srgb, var(--color-accent) 10%, transparent) 0%, transparent 58%)",
  } as const;

  if (reduced) {
    return (
      <div className={containerClass} style={containerStyle}>
        <Thumbnail
          component={HeroChatComposition}
          inputProps={{ staticMode: true }}
          compositionWidth={COMPOSITION_WIDTH}
          compositionHeight={COMPOSITION_HEIGHT}
          durationInFrames={DURATION_IN_FRAMES}
          fps={FPS}
          frameToDisplay={STATIC_FALLBACK_FRAME}
          style={{ width: "100%", height: "100%", background: "transparent" }}
        />
      </div>
    );
  }

  return (
    <div className={containerClass} style={containerStyle}>
      <Player
        component={HeroChatComposition}
        compositionWidth={COMPOSITION_WIDTH}
        compositionHeight={COMPOSITION_HEIGHT}
        durationInFrames={DURATION_IN_FRAMES}
        fps={FPS}
        loop
        autoPlay
        controls={false}
        clickToPlay={false}
        doubleClickToFullscreen={false}
        acknowledgeRemotionLicense
        style={{ width: "100%", height: "100%", background: "transparent" }}
      />
    </div>
  );
}
