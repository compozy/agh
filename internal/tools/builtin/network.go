package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

const (
	networkDirectsKey = "directs"
	networkNetworkKey = "network"
)

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
		[]string{networkNetworkKey, "status"},
		[]string{"network status", "network diagnostics"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkChannels,
		"network_channels",
		"Network Channels",
		"List active AGH network channels through the existing network manager.",
		networkWorkspaceInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, "channels"},
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
		[]string{networkNetworkKey, "inbox"},
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
		[]string{networkNetworkKey, "peers"},
		[]string{"network peers", "channel presence"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkSend,
		"network_send",
		"Network Send",
		"Send one AGH network message into a public thread or restricted direct room. "+
			"Direct-room visibility is restricted to the two room peers plus runtime/audit access, "+
			"not cryptographic privacy.",
		networkSendInputSchema,
		toolspkg.RiskOpenWorld,
		false,
		false,
		true,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, "send"},
		[]string{"network message", "send to peer"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkThreads,
		"network_threads",
		"Network Threads",
		"List public-thread summaries for one AGH network channel.",
		networkThreadsInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, "threads"},
		[]string{"network threads", "public thread summaries"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkThreadMessages,
		"network_thread_messages",
		"Network Thread Messages",
		"Read messages isolated to one public AGH network thread.",
		networkThreadMessagesInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, "threads", "messages"},
		[]string{"thread messages", "public thread timeline"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkDirects,
		"network_directs",
		"Network Direct Rooms",
		"List direct-room summaries for one AGH network channel. "+
			"Direct-room visibility is restricted to the two room peers plus runtime/audit access, "+
			"not cryptographic privacy.",
		networkDirectsInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, networkDirectsKey},
		[]string{"direct rooms", "restricted network rooms"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkDirectResolve,
		"network_direct_resolve",
		"Network Direct Resolve",
		"Create or return the deterministic direct room for the caller session and one peer. "+
			"Direct-room visibility is restricted to the two room peers plus runtime/audit access, "+
			"not cryptographic privacy.",
		networkDirectResolveInputSchema,
		toolspkg.RiskMutating,
		false,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, networkDirectsKey, "resolve"},
		[]string{"resolve direct room", "open restricted room"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkDirectMessages,
		"network_direct_messages",
		"Network Direct Messages",
		"Read messages isolated to one restricted direct room. "+
			"Direct-room visibility is restricted to the two room peers plus runtime/audit access, "+
			"not cryptographic privacy.",
		networkDirectMessagesInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, networkDirectsKey, "messages"},
		[]string{"direct messages", "restricted room timeline"},
	),
	nativeDescriptor(
		toolspkg.ToolIDNetworkWork,
		"network_work",
		"Network Work",
		"Read lifecycle metadata for one AGH network work_id.",
		networkWorkInputSchema,
		toolspkg.RiskRead,
		true,
		false,
		false,
		[]toolspkg.ToolsetID{toolspkg.ToolsetIDCoordination},
		[]string{networkNetworkKey, "work"},
		[]string{"network work", "work lifecycle"},
	),
}

func networkDescriptors() []toolspkg.Descriptor {
	return networkTools
}

const networkWorkspaceInputSchema = `{
	"type":"object",
	"required":["workspace_id"],
	"properties":{
		"workspace_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkPeersInputSchema = `{
	"type":"object",
	"required":["workspace_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"channel":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkInboxInputSchema = `{
	"type":"object",
	"required":["workspace_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"session_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkSendInputSchema = `{
	"type":"object",
	"required":["workspace_id","channel","kind","body"],
	"properties":{
		"workspace_id":{"type":"string"},
		"session_id":{"type":"string"},
		"channel":{"type":"string"},
		"kind":{"type":"string","enum":["greet","whois","say","capability","receipt","trace"]},
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

const networkThreadsInputSchema = `{
	"type":"object",
	"required":["workspace_id","channel"],
	"properties":{
		"workspace_id":{"type":"string"},
		"channel":{"type":"string"},
		"limit":{"type":"integer"},
		"after":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkThreadMessagesInputSchema = `{
	"type":"object",
	"required":["workspace_id","channel","thread_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"channel":{"type":"string"},
		"thread_id":{"type":"string"},
		"before":{"type":"string"},
		"after":{"type":"string"},
		"kind":{"type":"string","enum":["greet","whois","say","capability","receipt","trace"]},
		"work_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const networkDirectsInputSchema = `{
	"type":"object",
	"required":["workspace_id","channel"],
	"properties":{
		"workspace_id":{"type":"string"},
		"channel":{"type":"string"},
		"peer_id":{"type":"string"},
		"limit":{"type":"integer"},
		"after":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkDirectResolveInputSchema = `{
	"type":"object",
	"required":["workspace_id","channel","peer_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"session_id":{"type":"string"},
		"channel":{"type":"string"},
		"peer_id":{"type":"string"}
	},
	"additionalProperties":false
}`

const networkDirectMessagesInputSchema = `{
	"type":"object",
	"required":["workspace_id","channel","direct_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"channel":{"type":"string"},
		"direct_id":{"type":"string"},
		"before":{"type":"string"},
		"after":{"type":"string"},
		"kind":{"type":"string","enum":["greet","whois","say","capability","receipt","trace"]},
		"work_id":{"type":"string"},
		"limit":{"type":"integer"}
	},
	"additionalProperties":false
}`

const networkWorkInputSchema = `{
	"type":"object",
	"required":["workspace_id","work_id"],
	"properties":{
		"workspace_id":{"type":"string"},
		"work_id":{"type":"string"}
	},
	"additionalProperties":false
}`
