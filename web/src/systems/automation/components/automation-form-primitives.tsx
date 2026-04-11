import type {
  InputHTMLAttributes,
  ReactNode,
  SelectHTMLAttributes,
  TextareaHTMLAttributes,
} from "react";

import { cn } from "@/lib/utils";

interface AutomationFieldProps {
  label: string;
  hint?: string;
  children: ReactNode;
}

export function AutomationField({ label, hint, children }: AutomationFieldProps) {
  return (
    <label className="flex flex-col gap-2">
      <span className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-[color:var(--color-text-primary)]">{label}</span>
        {hint ? (
          <span className="text-xs text-[color:var(--color-text-tertiary)]">{hint}</span>
        ) : null}
      </span>
      {children}
    </label>
  );
}

function inputBaseClassName() {
  return cn(
    "w-full rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]",
    "px-3 py-2 text-sm text-[color:var(--color-text-primary)] outline-none transition-colors",
    "placeholder:text-[color:var(--color-text-tertiary)] focus:border-[color:var(--color-accent)]"
  );
}

export function AutomationInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={cn(inputBaseClassName(), props.className)} />;
}

export function AutomationTextarea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={cn(inputBaseClassName(), "min-h-28 resize-y py-2.5", props.className)}
    />
  );
}

export function AutomationSelect(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select {...props} className={cn(inputBaseClassName(), props.className)} />;
}

interface AutomationCheckboxProps {
  checked: boolean;
  description: string;
  label: string;
  onCheckedChange: (checked: boolean) => void;
  testId?: string;
}

export function AutomationCheckbox({
  checked,
  description,
  label,
  onCheckedChange,
  testId,
}: AutomationCheckboxProps) {
  return (
    <label
      className="flex items-start gap-3 rounded-lg border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-3"
      data-testid={testId}
    >
      <input
        checked={checked}
        className="mt-0.5 size-4 accent-[color:var(--color-accent)]"
        onChange={event => onCheckedChange(event.target.checked)}
        type="checkbox"
      />
      <span className="flex flex-col gap-1">
        <span className="text-sm font-medium text-[color:var(--color-text-primary)]">{label}</span>
        <span className="text-xs leading-relaxed text-[color:var(--color-text-secondary)]">
          {description}
        </span>
      </span>
    </label>
  );
}

interface AutomationFormSectionProps {
  children: ReactNode;
  description?: string;
  title: string;
}

export function AutomationFormSection({
  children,
  description,
  title,
}: AutomationFormSectionProps) {
  return (
    <section className="space-y-4 rounded-xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
      <div className="space-y-1">
        <h3 className="font-mono text-[10px] font-semibold uppercase tracking-[0.1em] text-[color:var(--color-text-label)]">
          {title}
        </h3>
        {description ? (
          <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
            {description}
          </p>
        ) : null}
      </div>
      {children}
    </section>
  );
}
