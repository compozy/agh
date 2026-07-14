import { Link } from "@tanstack/react-router";

/**
 * Informative Network mention for onboarding (UT-059).
 * Visit/complete/dismiss must not mutate network settings.
 */
export function OnboardingNetworkMention() {
  return (
    <p className="text-sm text-muted" data-testid="onboarding-network-mention">
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
