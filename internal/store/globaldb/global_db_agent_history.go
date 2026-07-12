package globaldb

import (
	"context"
	"fmt"
	"strings"
)

// DeleteHeartbeatAgentHistory removes revisions owned by one effective authored definition.
func (g *GlobalDB) DeleteHeartbeatAgentHistory(
	ctx context.Context,
	workspaceID string,
	agentName string,
	sourcePath string,
) error {
	if err := g.checkReady(ctx, "delete heartbeat agent history"); err != nil {
		return err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	agentName = strings.TrimSpace(agentName)
	sourcePath = strings.TrimSpace(sourcePath)
	if workspaceID == "" || agentName == "" || sourcePath == "" {
		return fmt.Errorf("store: delete heartbeat agent history requires workspace id, agent name, and source path")
	}
	if _, err := g.db.ExecContext(
		ctx,
		`DELETE FROM agent_heartbeat_revisions WHERE workspace_id = ? AND agent_name = ? AND source_path = ?`,
		workspaceID,
		agentName,
		sourcePath,
	); err != nil {
		return fmt.Errorf("store: delete heartbeat agent history: %w", err)
	}
	return nil
}

// DeleteSoulAgentHistory removes revisions owned by one effective authored definition.
func (g *GlobalDB) DeleteSoulAgentHistory(
	ctx context.Context,
	workspaceID string,
	agentName string,
	sourcePath string,
) error {
	if err := g.checkReady(ctx, "delete soul agent history"); err != nil {
		return err
	}
	workspaceID = strings.TrimSpace(workspaceID)
	agentName = strings.TrimSpace(agentName)
	sourcePath = strings.TrimSpace(sourcePath)
	if workspaceID == "" || agentName == "" || sourcePath == "" {
		return fmt.Errorf("store: delete soul agent history requires workspace id, agent name, and source path")
	}
	if _, err := g.db.ExecContext(
		ctx,
		`DELETE FROM agent_soul_revisions WHERE workspace_id = ? AND agent_name = ? AND source_path = ?`,
		workspaceID,
		agentName,
		sourcePath,
	); err != nil {
		return fmt.Errorf("store: delete soul agent history: %w", err)
	}
	return nil
}
