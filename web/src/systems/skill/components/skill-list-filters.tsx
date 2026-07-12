import { ListFilter } from "lucide-react";
import { useMemo } from "react";

import { Button } from "@agh/ui";
import { Filters, type Filter } from "@agh/ui";

import {
  applySkillFilterChips,
  buildSkillFilterFields,
  skillFiltersToChips,
  type SkillEnabledFilter,
  type SkillSourceFilter,
} from "../lib/skill-list-filters";

export interface SkillListFiltersProps {
  sourceFilter: SkillSourceFilter | null;
  enabledFilter: SkillEnabledFilter | null;
  onSourceFilterChange: (next: SkillSourceFilter | null) => void;
  onEnabledFilterChange: (next: SkillEnabledFilter | null) => void;
}

/**
 * Installed-skills filter chip bar for composition inside ListingToolbar.Filters.
 */
function SkillListFilters({
  sourceFilter,
  enabledFilter,
  onSourceFilterChange,
  onEnabledFilterChange,
}: SkillListFiltersProps) {
  const fields = useMemo(() => buildSkillFilterFields(), []);
  const chips = useMemo(
    () => skillFiltersToChips({ source: sourceFilter, enabled: enabledFilter }),
    [enabledFilter, sourceFilter]
  );

  const handleFiltersChange = (next: Filter<string>[]) => {
    applySkillFilterChips(next, {
      onSourceChange: onSourceFilterChange,
      onEnabledChange: onEnabledFilterChange,
    });
  };

  return (
    <Filters<string>
      allowMultiple={false}
      data-testid="skill-list-filters"
      fields={fields}
      filters={chips}
      onChange={handleFiltersChange}
      size="sm"
      trigger={
        <Button
          aria-label="Add filter"
          data-testid="skill-list-filters-add"
          size="sm"
          type="button"
          variant="ghost"
        >
          <ListFilter aria-hidden="true" className="size-3" />
          Filter
        </Button>
      }
    />
  );
}

export { SkillListFilters };
