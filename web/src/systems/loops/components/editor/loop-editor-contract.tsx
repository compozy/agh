import { Field, FieldDescription, FieldLabel, Textarea } from "@agh/ui";

import type { EditableLoopContractField } from "../../lib/loop-editor-definition";
import type { LoopContract } from "../../types";

interface LoopEditorContractProps {
  contract: LoopContract;
  disabled: boolean;
  onChange: (field: EditableLoopContractField, value: string) => void;
}

/** Authoring controls for the Loop-level outcome contract, separate from graph-node fields. */
export function LoopEditorContract({ contract, disabled, onChange }: LoopEditorContractProps) {
  return (
    <section className="flex-none border-b border-line bg-canvas px-4 py-3.5">
      <div className="mb-3">
        <h2 className="text-item-title font-medium text-fg-strong">Contract</h2>
        <p className="mt-1 text-[11px] leading-relaxed text-subtle">
          Define the outcome this Loop should reach.
        </p>
      </div>
      <div className="flex flex-col gap-3.5">
        <Field>
          <FieldLabel htmlFor="loop-contract-goal">Goal (optional)</FieldLabel>
          <Textarea
            id="loop-contract-goal"
            rows={2}
            value={contract.goal ?? ""}
            disabled={disabled}
            onChange={event => onChange("goal", event.target.value)}
            className="min-h-16 resize-y"
          />
          <FieldDescription>Clear this field to publish a goal-less Loop.</FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor="loop-contract-definition-of-done">Definition of done</FieldLabel>
          <Textarea
            id="loop-contract-definition-of-done"
            rows={3}
            value={contract.definition_of_done}
            disabled={disabled}
            onChange={event => onChange("definition_of_done", event.target.value)}
            className="min-h-20 resize-y"
          />
        </Field>
      </div>
    </section>
  );
}
