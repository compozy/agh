package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

var networkTools = []toolspkg.Descriptor{
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

const networkSendInputSchema = `{
	"type":"object",
	"required":["channel","kind","body"],
	"properties":{
		"session_id":{"type":"string"},
		"channel":{"type":"string"},
		"kind":{"type":"string"},
		"to":{"type":"string"},
		"body":{"type":"object"},
		"interaction_id":{"type":"string"},
		"reply_to":{"type":"string"},
		"trace_id":{"type":"string"},
		"causation_id":{"type":"string"},
		"expires_at":{"type":"integer"},
		"id":{"type":"string"},
		"ext":{"type":"object"}
	},
	"additionalProperties":false
}`
