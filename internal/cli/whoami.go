package cli

import (
	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/spf13/cobra"
)

const (
	envSessionID = agentidentity.EnvSessionID
	envAgentID   = agentidentity.EnvAgent
	envAgentName = "AGH_AGENT_NAME"
)

func newWhoamiCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the current AGH agent identity from environment variables",
		RunE: func(cmd *cobra.Command, _ []string) error {
			identity := IdentityRecord{
				SessionID: deps.getenv(envSessionID),
				Agent:     deps.getenv(envAgentID),
				AgentName: deps.getenv(envAgentName),
			}
			return writeCommandOutput(cmd, whoamiBundle(identity))
		},
	}
}

func whoamiBundle(identity IdentityRecord) outputBundle {
	return outputBundle{
		jsonValue: identity,
		human: func() (string, error) {
			return renderHumanSection("Identity", []keyValue{
				{Label: "Session ID", Value: stringOrDash(identity.SessionID)},
				{Label: "Agent", Value: stringOrDash(identity.Agent)},
				{Label: "Agent Name", Value: stringOrDash(identity.AgentName)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("identity", []string{"session_id", "agent", "agent_name"}, []string{
				identity.SessionID,
				identity.Agent,
				identity.AgentName,
			}), nil
		},
	}
}
