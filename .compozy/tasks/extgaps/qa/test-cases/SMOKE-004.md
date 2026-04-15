# SMOKE-004: Bundle with all resource types materialized

**Priority:** P0 (Critical)
**Type:** Smoke
**Component:** Resource materialization

## Test Steps

1. Activate bundle with profile containing jobs, triggers, and bridges
   **Expected:** HTTP 201

2. Verify activation response contains jobs[] with stable IDs
   **Expected:** Each job has id, name, agent_name, enabled

3. Verify activation response contains triggers[] with stable IDs
   **Expected:** Each trigger has id, name, agent_name, event, enabled

4. Verify activation response contains bridges[] with stable IDs
   **Expected:** Each bridge has id, name, extension_name, platform, display_name

5. Verify inventory[] matches all materialized resources
   **Expected:** inventory count = len(jobs) + len(triggers) + len(bridges)

6. Verify inventory items have correct resource_kind values
   **Expected:** "automation_job", "automation_trigger", "bridge_instance"
