import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { Button, OwnerAvatar, Pill, Section, Surface } from "@agh/ui";

import { useSessionCreate } from "@/systems/session";

import { useElapsedNowSeconds } from "../hooks/use-elapsed-ticker";
import type { HomeRunCardModel } from "../hooks/use-home-working-now";

export interface HomeWorkingNowProps {
  cards: HomeRunCardModel[];
  total: number;
}

function formatElapsed(totalSeconds: number): string {
  if (totalSeconds >= 3600) {
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    return `${hours}h ${String(minutes).padStart(2, "0")}m`;
  }
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}m ${String(seconds).padStart(2, "0")}s`;
}

function HomeRunCard({ card, nowSeconds }: { card: HomeRunCardModel; nowSeconds: number }) {
  const ticked =
    card.elapsedBaseSeconds + Math.max(0, nowSeconds - Math.floor(card.baseAtMs / 1000));

  const body = (
    <Surface
      className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 transition-colors duration-base hover:bg-canvas-tint"
      data-slot="home-run-card"
    >
      <div className="min-w-0">
        <div className="flex items-center gap-2 text-small-body font-medium text-fg-strong">
          <Pill.Dot pulse tone="accent" />
          <OwnerAvatar name={card.agentName} ownerId={card.agentName} ownerKind="agent" size="sm" />
          <span className="truncate">{card.title}</span>
          {card.kind === "task_run" ? (
            <Pill size="xs" tone="neutral">
              task run
            </Pill>
          ) : null}
        </div>
        <p className="mt-1 truncate text-small-body text-muted">{card.subtitle}</p>
      </div>
      <span className="font-mono text-mono-id tabular-nums text-muted">
        {formatElapsed(ticked)}
      </span>
    </Surface>
  );

  if (card.sessionLink) {
    return (
      <Link
        className="block min-w-0 rounded-lg focus-visible:shadow-focus-ring focus-visible:outline-none"
        params={{ name: card.sessionLink.agentName, id: card.sessionLink.sessionId }}
        to="/agents/$name/sessions/$id"
      >
        {body}
      </Link>
    );
  }
  if (card.runLink) {
    return (
      <Link
        className="block min-w-0 rounded-lg focus-visible:shadow-focus-ring focus-visible:outline-none"
        params={{ id: card.runLink.taskId, runId: card.runLink.runId }}
        to="/tasks/$id/runs/$runId"
      >
        {body}
      </Link>
    );
  }
  return body;
}

function HomeWorkingNowEmpty() {
  const sessionCreate = useSessionCreate();
  return (
    <Surface className="flex items-center justify-between gap-3" data-slot="home-working-now-empty">
      <p className="text-small-body text-subtle">No agents working right now.</p>
      <Button onClick={() => sessionCreate.openForAgent("")} size="sm" variant="neutral">
        Start a session
      </Button>
    </Surface>
  );
}

/**
 * Zone 3a — live run cards for active sessions and task runs; a single shared
 * ticker keeps every elapsed timer in lockstep.
 */
export function HomeWorkingNow({ cards, total }: HomeWorkingNowProps) {
  const nowSeconds = useElapsedNowSeconds();

  return (
    <Section
      className="flex min-h-full flex-col"
      bodyClassName="flex flex-1 flex-col gap-2.5"
      count={total}
      label="Working now"
      right={
        <Button render={<Link to="/agents" />} size="sm" variant="ghost">
          View agents
          <ChevronRight aria-hidden="true" />
        </Button>
      }
    >
      {cards.length === 0 ? (
        <HomeWorkingNowEmpty />
      ) : (
        cards.map(card => <HomeRunCard card={card} key={card.key} nowSeconds={nowSeconds} />)
      )}
    </Section>
  );
}
