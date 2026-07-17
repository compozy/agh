import { cn } from "@agh/ui";
import { Link } from "@tanstack/react-router";
import type { ComponentProps } from "react";

/**
 * Informative Network mention for onboarding (UT-059).
 * Visit/complete/dismiss must not mutate network settings.
 */
export type OnboardingNetworkMentionProps = Omit<ComponentProps<"p">, "children">;

export function OnboardingNetworkMention({ className, ...props }: OnboardingNetworkMentionProps) {
  return (
    <p
      {...props}
      className={cn("text-sm text-muted", className)}
      data-testid="onboarding-network-mention"
    >
      Explore{" "}
      <Link
        className="text-action underline-offset-2 hover:underline"
        data-testid="onboarding-network-link"
        to="/network"
      >
        Network
      </Link>{" "}
      after setup. Opening it does not enable coordination or change settings.
    </p>
  );
}
