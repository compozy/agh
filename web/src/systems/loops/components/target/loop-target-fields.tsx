import { Field, FieldLabel, NativeSelect, NativeSelectOption, Spinner } from "@agh/ui";

import { useLoops } from "../../hooks/use-loops";
import {
  setLoopTargetInput,
  setLoopTargetLoop,
  setLoopTargetMapping,
  type LoopTargetDraft,
} from "../../lib/loop-target";
import { LoopInputControl } from "./loop-input-control";
import { LoopInputMapping } from "./loop-input-mapping";

interface LoopTargetFieldsProps {
  workspaceId: string;
  value: LoopTargetDraft;
  onChange: (value: LoopTargetDraft) => void;
  /** Show the event-payload mapping table (triggers/webhooks only). */
  showMapping?: boolean;
}

/**
 * The Run-loop target editor (TechSpec §9.14): a loop picker, an auto-generated
 * typed-input form from the chosen Loop's declared inputs, and — for
 * triggers/webhooks — an event-payload -> input mapping table. Rendered from the
 * daemon's declared-input schema, so it always matches what the Loop accepts.
 */
export function LoopTargetFields({
  workspaceId,
  value,
  onChange,
  showMapping = false,
}: LoopTargetFieldsProps) {
  const loopsQuery = useLoops(workspaceId, workspaceId !== "");
  const loops = loopsQuery.data ?? [];
  const selected = loops.find(loop => loop.name === value.loop_name) ?? null;
  const inputs = selected?.inputs ?? {};
  const inputNames = Object.keys(inputs);

  return (
    <div className="space-y-4" data-testid="loop-target-fields">
      <Field>
        <FieldLabel htmlFor="loop-target-loop">Loop</FieldLabel>
        {loopsQuery.isLoading ? (
          <div className="flex h-9 items-center gap-2 text-[11.5px] text-subtle">
            <Spinner aria-hidden="true" className="size-3.5 text-subtle" />
            Loading loops…
          </div>
        ) : loops.length === 0 ? (
          <p className="text-[11.5px] text-subtle">No Loops are available in this workspace.</p>
        ) : (
          <NativeSelect
            id="loop-target-loop"
            data-testid="loop-target-select"
            value={value.loop_name}
            onChange={event => onChange(setLoopTargetLoop(value, event.target.value))}
          >
            <NativeSelectOption value="">Select a loop</NativeSelectOption>
            {loops.map(loop => (
              <NativeSelectOption key={loop.name} value={loop.name}>
                {loop.name}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        )}
      </Field>

      {selected && inputNames.length > 0 ? (
        <div className="space-y-3" data-testid="loop-target-inputs">
          {inputNames.map(name => (
            <LoopInputControl
              key={name}
              name={name}
              field={inputs[name]}
              value={value.inputs?.[name]}
              onChange={next => onChange(setLoopTargetInput(value, name, next))}
            />
          ))}
        </div>
      ) : selected ? (
        <p className="text-[11.5px] text-subtle">This Loop declares no inputs.</p>
      ) : null}

      {selected && showMapping ? (
        <LoopInputMapping
          inputs={inputs}
          mapping={value.input_mapping ?? {}}
          onChange={(key, path) => onChange(setLoopTargetMapping(value, key, path))}
        />
      ) : null}
    </div>
  );
}
