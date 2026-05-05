package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var networkTools = []toolspkg.Descriptor{
	nativeDescriptor(
		toolspkg.ToolIDNetworkStatus,
		"network_status",
		"Network Status",
		"Read daemon-owned AGH network runtime status.",
		emptyInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{"network", "status"},
		[]string{"network status", "network diagnostics"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkChannels,
		"network_channels",
		"Network Channels",
		"List active AGH network channels through the existing network manager.",
		emptyInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{"network", "channels"},
		[]string{"network channels", "coordination channels"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkInbox,
		"network_inbox",
		"Network Inbox",
		"Read queued inbound AGH network messages for one local session.",
		networkInboxInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{"network", "inbox"},
		[]string{"network inbox", "queued network messages"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkPeers,
		"network_peers",
		"Network Peers",
		"List visible AGH network peers through the existing network manager.",
		networkPeersInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{"network", "peers"},
		[]string{"network peers", "channel presence"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkSend,
		"network_send",
		"Network Send",
		"Send one AGH network message through the existing network manager.",
		networkSendInputSchema,
		toolspkg.RiskOpenWorld,
		false,
		false,
		true,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{"network", "send"},
		[]string{"network message", "send to peer"},
	),
}

func networkDescriptors() []toolspkg.Descriptor {
	return networkTools
}

const networkPeersInputSchema = `{
	"type":"object",
	"properties":{
		"channel":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkInboxInputSchema = `{
	"type":"object",
	"properties":{
		"session_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkSendInputSchema = `{
	"type":"object",
	"required":["channel","kind","body"],
	"properties":{
		"session_id":{"type":"string"},
		"channel":{"type":"string"},
		"kind":{"type":"string"},
		"surface":{"type":"string","enum":["thread","direct"]},
		"thread_id":{"type":"string"},
		"direct_id":{"type":"string"},
		"work_id":{"type":"string"},
		"to":{"type":"string"},
		"body":{"type":"object"},
		"reply_to":{"type":"string"},
		"trace_id":{"type":"string"},
		"causation_id":{"type":"string"},
		"expires_at":{"type":"integer"},
		"id":{"type":"string"},
		"ext":{"type":"object"}
	},
	"additionalProperties":false
}`
