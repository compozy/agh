import { GitBranchPlus } from "lucide-react";

import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldLabel,
  Section,
  Spinner,
  Textarea,
} from "@agh/ui";
import { NetworkParticipationFields } from "@/systems/network";

import { useTasksFanOutRunsCard } from "../hooks/use-tasks-fan-out-runs-card";
import type { FanOutTaskRunsRequest, FanOutTaskRunsResponse } from "../types";

export interface TasksFanOutRunsCardProps {
  isPending?: boolean;
  onFanOut: (data: FanOutTaskRunsRequest) => Promise<FanOutTaskRunsResponse | void>;
}

export function TasksFanOutRunsCard({ isPending = false, onFanOut }: TasksFanOutRunsCardProps) {
  const state = useTasksFanOutRunsCard({ onFanOut });

  return (
    <Section
      aria-label="Designated fan-out"
      bodyClassName="gap-4"
      className="w-full gap-4"
      data-testid="tasks-fan-out-runs-card"
      icon={GitBranchPlus}
      label="Designated fan-out"
      note="Start several scoped runs from one task."
      right={
        <Button
          data-testid="tasks-fan-out-runs-trigger"
          disabled={isPending}
          onClick={state.openDialog}
          size="sm"
          type="button"
          variant="neutral"
        >
          <GitBranchPlus aria-hidden="true" className="size-3" />
          Fan out
        </Button>
      }
    >
      <p className="max-w-3xl text-small-body text-muted">
        Designations become assignment context in each worker prompt and status rollups return to
        the originating network thread when the task came from AGH Network.
      </p>

      <Dialog onOpenChange={state.handleOpenChange} open={state.open}>
        <DialogContent
          className="gap-0 p-0 text-fg sm:max-w-2xl"
          data-testid="tasks-fan-out-runs-dialog"
        >
          <DialogHeader className="border-b border-line px-5 py-4">
            <DialogTitle>Fan out task runs</DialogTitle>
            <DialogDescription>
              Create scoped assignments that run under one designation group.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={state.handleSubmit}>
            <div className="space-y-5 p-5">
              {state.formError ? (
                <Alert data-testid="tasks-fan-out-runs-error" variant="danger">
                  <AlertDescription>{state.formError}</AlertDescription>
                </Alert>
              ) : null}

              <Field>
                <FieldLabel htmlFor="tasks-fan-out-designations">Assignments</FieldLabel>
                <FieldDescription>
                  One line per run. Each line is injected as that worker's assignment brief.
                </FieldDescription>
                <Textarea
                  data-testid="tasks-fan-out-designations"
                  id="tasks-fan-out-designations"
                  onChange={event => state.setDesignationsText(event.target.value)}
                  rows={5}
                  value={state.designationsText}
                  variant="mono"
                />
              </Field>

              <NetworkParticipationFields
                allowedStrategies={state.networkStrategies}
                onChange={state.setNetworkParticipation}
                testIdPrefix="tasks-fan-out-network"
                value={state.networkParticipation}
              />
            </div>

            <DialogFooter className="border-t border-line bg-canvas-soft px-5 py-3">
              <Button onClick={state.closeDialog} type="button" variant="outline">
                Cancel
              </Button>
              <Button data-testid="tasks-fan-out-runs-submit" disabled={isPending} type="submit">
                {isPending ? <Spinner aria-hidden="true" className="size-4" /> : null}
                Create runs
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </Section>
  );
}
