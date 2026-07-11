import { ListFilter } from "lucide-react";
import { useMemo, type RefObject } from "react";

import { Button, Filters, ListingToolbar, SearchInput, type Filter } from "@agh/ui";

import {
  agentFleetFiltersToChips,
  agentFleetChipsToFilters,
  buildAgentFleetFilterFields,
} from "../lib/agent-fleet-filters";
import type { AgentsFleetSearch } from "../lib/agent-fleet-search";

export interface AgentFleetToolbarProps {
  search: AgentsFleetSearch;
  draftQuery: string;
  categoryOptions: readonly string[];
  searchInputRef: RefObject<HTMLInputElement | null>;
  onDraftQueryChange: (next: string) => void;
  onFiltersChange: (next: Pick<AgentsFleetSearch, "category" | "status">) => void;
}

function AgentFleetToolbar({
  search,
  draftQuery,
  categoryOptions,
  searchInputRef,
  onDraftQueryChange,
  onFiltersChange,
}: AgentFleetToolbarProps) {
  const fields = useMemo(() => buildAgentFleetFilterFields(categoryOptions), [categoryOptions]);
  const chips = useMemo(() => agentFleetFiltersToChips(search), [search]);

  const handleFiltersChange = (next: Filter<string>[]) => {
    onFiltersChange(agentFleetChipsToFilters(next));
  };

  return (
    <ListingToolbar data-testid="agent-fleet-toolbar">
      <ListingToolbar.Leading>
        <SearchInput
          ref={searchInputRef}
          aria-label="Search agents"
          data-testid="agent-fleet-search"
          kbd="/"
          onChange={onDraftQueryChange}
          placeholder="Search agents"
          value={draftQuery}
        />
        <ListingToolbar.Filters>
          <Filters<string>
            allowMultiple={false}
            data-testid="agent-fleet-filters"
            fields={fields}
            filters={chips}
            onChange={handleFiltersChange}
            size="sm"
            trigger={
              <Button
                aria-label="Add filter"
                data-testid="agent-fleet-filters-add"
                size="sm"
                type="button"
                variant="ghost"
              >
                <ListFilter aria-hidden="true" className="size-3" />
                Filter
              </Button>
            }
          />
        </ListingToolbar.Filters>
      </ListingToolbar.Leading>
    </ListingToolbar>
  );
}

export { AgentFleetToolbar };
