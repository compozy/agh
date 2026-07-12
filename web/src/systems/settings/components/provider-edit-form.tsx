import type { ProviderDraft } from "../types";
import { ProviderAuthFields } from "./provider-edit-form-auth-fields";
import { ProviderGeneralFields } from "./provider-edit-form-general-fields";

export type ProviderDraftChange = (updater: (draft: ProviderDraft) => ProviderDraft) => void;

interface ProviderEditFormProps {
  mode: "create" | "edit";
  draft: ProviderDraft;
  onChange: ProviderDraftChange;
}

export function ProviderEditForm({ mode, draft, onChange }: ProviderEditFormProps) {
  return (
    <div className="flex flex-col gap-3">
      <ProviderGeneralFields draft={draft} mode={mode} onChange={onChange} />
      <ProviderAuthFields draft={draft} onChange={onChange} />
    </div>
  );
}
