package network

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	sessionpkg "github.com/pedronauck/agh/internal/session"
)

// LocalPeer is one daemon-local peer joined to one runtime channel.
type LocalPeer struct {
	SessionID         string
	PeerID            string
	WorkspaceID       string
	Channel           string
	PeerCard          PeerCard
	CapabilityCatalog []sessionpkg.NetworkPeerCapability
	JoinedAt          time.Time
}

// RemotePeerEntry is one cached remote peer advertisement.
type RemotePeerEntry struct {
	PeerID                 string
	PeerCard               PeerCard
	WorkspaceID            string
	Channel                string
	CapabilityCatalog      []sessionpkg.NetworkPeerCapability
	CapabilityCatalogKnown bool
	LastSeen               time.Time
	ExpiresAt              time.Time
}

// PeerInfo is the API-facing snapshot for one visible peer.
type PeerInfo struct {
	SessionID              *string
	PeerID                 string
	WorkspaceID            string
	Channel                string
	Local                  bool
	PeerCard               PeerCard
	CapabilityCatalog      []sessionpkg.NetworkPeerCapability
	CapabilityCatalogKnown bool
	JoinedAt               *time.Time
	LastSeen               *time.Time
	ExpiresAt              *time.Time
}

// ChannelInfo summarizes one active runtime channel.
type ChannelInfo struct {
	WorkspaceID string
	Channel     string
	PeerCount   int
}

// PeerRegistry tracks local session peers plus the remote peer cache.
type PeerRegistry struct {
	mu               sync.RWMutex
	greetInterval    time.Duration
	now              func() time.Time
	localsByID       map[string]LocalPeer
	localsByChannel  map[string]map[string]string
	remotesByChannel map[string]map[string]RemotePeerEntry
}

// PeerRegistryOption customizes the registry runtime.
type PeerRegistryOption func(*PeerRegistry)

// WithPeerRegistryClock overrides the time source used by the registry.
func WithPeerRegistryClock(now func() time.Time) PeerRegistryOption {
	return func(registry *PeerRegistry) {
		registry.now = now
	}
}

// NewPeerRegistry constructs the in-memory presence registry.
func NewPeerRegistry(greetInterval time.Duration, opts ...PeerRegistryOption) (*PeerRegistry, error) {
	if greetInterval <= 0 {
		return nil, fmt.Errorf("%w: greet interval must be positive", ErrInvalidField)
	}

	registry := &PeerRegistry{
		greetInterval:    greetInterval,
		now:              func() time.Time { return time.Now().UTC() },
		localsByID:       make(map[string]LocalPeer),
		localsByChannel:  make(map[string]map[string]string),
		remotesByChannel: make(map[string]map[string]RemotePeerEntry),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(registry)
		}
	}
	if registry.now == nil {
		registry.now = func() time.Time { return time.Now().UTC() }
	}

	return registry, nil
}

// GreetInterval reports the configured presence heartbeat interval.
func (r *PeerRegistry) GreetInterval() time.Duration {
	if r == nil {
		return 0
	}
	return r.greetInterval
}

// DefaultPeerCard returns the minimal protocol peer card for one peer identifier.
func DefaultPeerCard(peerID string) (PeerCard, error) {
	card := PeerCard{
		PeerID:              strings.TrimSpace(peerID),
		ProfilesSupported:   []string{ProtocolV0},
		Capabilities:        []string{},
		ArtifactsSupported:  []string{},
		TrustModesSupported: []string{},
	}
	normalized, err := normalizePeerCard(card)
	if err != nil {
		return PeerCard{}, err
	}
	return normalized, nil
}

// RegisterLocal upserts one local peer membership keyed by session ID.
func (r *PeerRegistry) RegisterLocal(
	sessionID string,
	workspaceID string,
	channel string,
	card PeerCard,
	joinedAt time.Time,
) (LocalPeer, error) {
	return r.RegisterLocalWithCapabilityCatalog(sessionID, workspaceID, channel, card, nil, joinedAt)
}

// RegisterLocalWithCapabilityCatalog upserts one local peer membership keyed by
// session ID, optionally retaining the runtime-owned rich capability catalog for
// explicit whois discovery.
func (r *PeerRegistry) RegisterLocalWithCapabilityCatalog(
	sessionID string,
	workspaceID string,
	channel string,
	card PeerCard,
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
	joinedAt time.Time,
) (LocalPeer, error) {
	if r == nil {
		return LocalPeer{}, fmt.Errorf("%w: peer registry is required", ErrInvalidField)
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return LocalPeer{}, fmt.Errorf("%w: session id is required", ErrMissingField)
	}
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := ValidateWorkspaceID(trimmedWorkspaceID); err != nil {
		return LocalPeer{}, err
	}
	trimmedChannel := strings.TrimSpace(channel)
	if err := ValidateChannel(trimmedChannel); err != nil {
		return LocalPeer{}, err
	}
	normalizedCard, err := normalizePeerCard(card)
	if err != nil {
		return LocalPeer{}, err
	}
	if joinedAt.IsZero() {
		joinedAt = r.now()
	}
	joinedAt = joinedAt.UTC()

	local := LocalPeer{
		SessionID:         trimmedSessionID,
		PeerID:            normalizedCard.PeerID,
		WorkspaceID:       trimmedWorkspaceID,
		Channel:           trimmedChannel,
		PeerCard:          normalizedCard,
		CapabilityCatalog: cloneNetworkPeerCapabilityCatalog(capabilityCatalog),
		JoinedAt:          joinedAt,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := peerChannelKey(trimmedWorkspaceID, trimmedChannel)
	if _, ok := r.localsByChannel[key]; !ok {
		r.localsByChannel[key] = make(map[string]string)
	}
	if owner, ok := r.localsByChannel[key][local.PeerID]; ok && owner != trimmedSessionID {
		return LocalPeer{}, fmt.Errorf(
			"%w: local peer_id already registered in channel: %s",
			ErrInvalidField,
			local.PeerID,
		)
	}
	if current, ok := r.localsByID[trimmedSessionID]; ok {
		r.removeLocalIndexesLocked(current)
	}
	r.localsByID[trimmedSessionID] = local
	r.localsByChannel[key][local.PeerID] = trimmedSessionID
	r.deleteRemoteLocked(trimmedWorkspaceID, trimmedChannel, local.PeerID)

	return cloneLocalPeer(local), nil
}

// LeaveLocal removes one local session peer from the registry.
func (r *PeerRegistry) LeaveLocal(sessionID string) (LocalPeer, bool) {
	if r == nil {
		return LocalPeer{}, false
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	r.mu.Lock()
	defer r.mu.Unlock()

	local, ok := r.localsByID[trimmedSessionID]
	if !ok {
		return LocalPeer{}, false
	}

	delete(r.localsByID, trimmedSessionID)
	r.removeLocalIndexesLocked(local)
	r.deleteRemoteLocked(local.WorkspaceID, local.Channel, local.PeerID)

	return cloneLocalPeer(local), true
}

// LocalBySession resolves one local peer by session ID.
func (r *PeerRegistry) LocalBySession(sessionID string) (LocalPeer, bool) {
	if r == nil {
		return LocalPeer{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	local, ok := r.localsByID[strings.TrimSpace(sessionID)]
	if !ok {
		return LocalPeer{}, false
	}
	return cloneLocalPeer(local), true
}

// LocalByPeer resolves one local peer by workspace, channel, and peer ID.
func (r *PeerRegistry) LocalByPeer(workspaceID string, channel string, peerID string) (LocalPeer, bool) {
	if r == nil {
		return LocalPeer{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	local, ok := r.lookupLocalLocked(
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(channel),
		strings.TrimSpace(peerID),
	)
	if !ok {
		return LocalPeer{}, false
	}
	return cloneLocalPeer(local), true
}

// LocalPeers returns the local peers currently joined to one workspace channel.
func (r *PeerRegistry) LocalPeers(workspaceID string, channel string) []LocalPeer {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	trimmedChannel := strings.TrimSpace(channel)
	sessionIDs := r.localsByChannel[peerChannelKey(trimmedWorkspaceID, trimmedChannel)]
	if len(sessionIDs) == 0 {
		return nil
	}

	peers := make([]LocalPeer, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		peers = append(peers, cloneLocalPeer(r.localsByID[sessionID]))
	}
	sort.Slice(peers, func(i int, j int) bool {
		return peers[i].SessionID < peers[j].SessionID
	})
	return peers
}

// MatchLocalPeers returns local peers matching one whois query.
func (r *PeerRegistry) MatchLocalPeers(workspaceID string, channel string, query string) []LocalPeer {
	peers := r.LocalPeers(workspaceID, channel)
	if len(peers) == 0 {
		return nil
	}

	matches := make([]LocalPeer, 0, len(peers))
	for _, peer := range peers {
		if matchesWhoisQuery(peer.PeerCard, query) {
			matches = append(matches, peer)
		}
	}
	return matches
}

// RefreshRemote stores or refreshes one remote peer advertisement.
func (r *PeerRegistry) RefreshRemote(
	workspaceID string,
	channel string,
	card PeerCard,
	seenAt time.Time,
) (RemotePeerEntry, bool, error) {
	return r.RefreshRemoteWithCapabilityCatalog(workspaceID, channel, card, nil, false, seenAt)
}

// RefreshRemoteWithCapabilityCatalog stores or refreshes one remote peer
// advertisement plus optional rich capability discovery state learned via
// explicit whois responses.
func (r *PeerRegistry) RefreshRemoteWithCapabilityCatalog(
	workspaceID string,
	channel string,
	card PeerCard,
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
	capabilityCatalogKnown bool,
	seenAt time.Time,
) (RemotePeerEntry, bool, error) {
	if r == nil {
		return RemotePeerEntry{}, false, fmt.Errorf("%w: peer registry is required", ErrInvalidField)
	}

	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	if err := ValidateWorkspaceID(trimmedWorkspaceID); err != nil {
		return RemotePeerEntry{}, false, err
	}
	trimmedChannel := strings.TrimSpace(channel)
	if err := ValidateChannel(trimmedChannel); err != nil {
		return RemotePeerEntry{}, false, err
	}
	normalizedCard, err := normalizePeerCard(card)
	if err != nil {
		return RemotePeerEntry{}, false, err
	}
	if seenAt.IsZero() {
		seenAt = r.now()
	}
	seenAt = seenAt.UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.expireRemotesLocked(seenAt)
	if _, ok := r.lookupLocalLocked(trimmedWorkspaceID, trimmedChannel, normalizedCard.PeerID); ok {
		r.deleteRemoteLocked(trimmedWorkspaceID, trimmedChannel, normalizedCard.PeerID)
		return RemotePeerEntry{}, false, nil
	}

	key := peerChannelKey(trimmedWorkspaceID, trimmedChannel)
	if _, ok := r.remotesByChannel[key]; !ok {
		r.remotesByChannel[key] = make(map[string]RemotePeerEntry)
	}
	existing, hasExisting := r.remotesByChannel[key][normalizedCard.PeerID]
	storedCatalog, storedCatalogKnown := nextRemoteCapabilityCatalog(
		existing,
		hasExisting,
		normalizedCard.Capabilities,
		capabilityCatalog,
		capabilityCatalogKnown,
	)

	entry := RemotePeerEntry{
		PeerID:                 normalizedCard.PeerID,
		PeerCard:               normalizedCard,
		WorkspaceID:            trimmedWorkspaceID,
		Channel:                trimmedChannel,
		CapabilityCatalog:      storedCatalog,
		CapabilityCatalogKnown: storedCatalogKnown,
		LastSeen:               seenAt,
		ExpiresAt:              seenAt.Add(2 * r.greetInterval),
	}
	r.remotesByChannel[key][entry.PeerID] = entry

	return cloneRemotePeerEntry(entry), true, nil
}

// RemoteByPeer resolves one active remote peer entry.
func (r *PeerRegistry) RemoteByPeer(
	workspaceID string,
	channel string,
	peerID string,
	at time.Time,
) (RemotePeerEntry, bool) {
	if r == nil {
		return RemotePeerEntry{}, false
	}

	if at.IsZero() {
		at = r.now()
	}
	at = at.UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.expireRemotesLocked(at)
	key := peerChannelKey(strings.TrimSpace(workspaceID), strings.TrimSpace(channel))
	channelEntries := r.remotesByChannel[key]
	entry, ok := channelEntries[strings.TrimSpace(peerID)]
	if !ok {
		return RemotePeerEntry{}, false
	}
	return cloneRemotePeerEntry(entry), true
}

// LookupPresence resolves one peer from the local registry first, then the remote cache.
func (r *PeerRegistry) LookupPresence(
	workspaceID string,
	channel string,
	peerID string,
	at time.Time,
) (PeerInfo, bool) {
	if r == nil {
		return PeerInfo{}, false
	}

	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	trimmedChannel := strings.TrimSpace(channel)
	trimmedPeerID := strings.TrimSpace(peerID)
	if at.IsZero() {
		at = r.now()
	}
	at = at.UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	if local, ok := r.lookupLocalLocked(trimmedWorkspaceID, trimmedChannel, trimmedPeerID); ok {
		return peerInfoFromLocal(local), true
	}
	r.expireRemotesLocked(at)
	if entry, ok := r.remotesByChannel[peerChannelKey(trimmedWorkspaceID, trimmedChannel)][trimmedPeerID]; ok {
		return peerInfoFromRemote(entry), true
	}
	return PeerInfo{}, false
}

// HasPresence reports whether the peer is visible and unexpired in the given channel.
func (r *PeerRegistry) HasPresence(workspaceID string, channel string, peerID string, at time.Time) bool {
	_, ok := r.LookupPresence(workspaceID, channel, peerID, at)
	return ok
}

// ListPeers returns visible peers, optionally filtered to one workspace channel.
func (r *PeerRegistry) ListPeers(workspaceID string, channel string, at time.Time) []PeerInfo {
	if r == nil {
		return nil
	}

	if at.IsZero() {
		at = r.now()
	}
	at = at.UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.expireRemotesLocked(at)
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	trimmedChannel := strings.TrimSpace(channel)
	if trimmedWorkspaceID != "" && trimmedChannel != "" {
		return listPeersForChannelLocked(r, trimmedWorkspaceID, trimmedChannel)
	}

	total := 0
	for _, local := range r.localsByID {
		if trimmedWorkspaceID != "" && local.WorkspaceID != trimmedWorkspaceID {
			continue
		}
		total++
	}
	for key, entries := range r.remotesByChannel {
		workspace, _ := splitPeerChannelKey(key)
		if trimmedWorkspaceID != "" && workspace != trimmedWorkspaceID {
			continue
		}
		total += len(entries)
	}

	peers := make([]PeerInfo, 0, total)
	for _, local := range r.localsByID {
		if trimmedWorkspaceID != "" && local.WorkspaceID != trimmedWorkspaceID {
			continue
		}
		peers = append(peers, peerInfoFromLocal(local))
	}
	for key, entries := range r.remotesByChannel {
		workspace, _ := splitPeerChannelKey(key)
		if trimmedWorkspaceID != "" && workspace != trimmedWorkspaceID {
			continue
		}
		for _, entry := range entries {
			peers = append(peers, peerInfoFromRemote(entry))
		}
	}
	sortPeerInfos(peers)
	return peers
}

// ListChannels returns active runtime channels plus current peer counts.
func (r *PeerRegistry) ListChannels(workspaceID string, at time.Time) []ChannelInfo {
	if r == nil {
		return nil
	}

	if at.IsZero() {
		at = r.now()
	}
	at = at.UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	r.expireRemotesLocked(at)
	counts := make(map[string]int)
	for _, local := range r.localsByID {
		if trimmedWorkspaceID != "" && local.WorkspaceID != trimmedWorkspaceID {
			continue
		}
		counts[peerChannelKey(local.WorkspaceID, local.Channel)]++
	}
	for key, entries := range r.remotesByChannel {
		workspace, _ := splitPeerChannelKey(key)
		if trimmedWorkspaceID != "" && workspace != trimmedWorkspaceID {
			continue
		}
		counts[key] += len(entries)
	}

	channels := make([]ChannelInfo, 0, len(counts))
	for key, count := range counts {
		workspace, channel := splitPeerChannelKey(key)
		channels = append(channels, ChannelInfo{WorkspaceID: workspace, Channel: channel, PeerCount: count})
	}
	sort.Slice(channels, func(i int, j int) bool {
		if channels[i].WorkspaceID != channels[j].WorkspaceID {
			return channels[i].WorkspaceID < channels[j].WorkspaceID
		}
		return channels[i].Channel < channels[j].Channel
	})
	return channels
}

func (r *PeerRegistry) lookupLocalLocked(workspaceID string, channel string, peerID string) (LocalPeer, bool) {
	sessionIDs := r.localsByChannel[peerChannelKey(workspaceID, channel)]
	sessionID, ok := sessionIDs[peerID]
	if !ok {
		return LocalPeer{}, false
	}
	local, ok := r.localsByID[sessionID]
	if !ok {
		return LocalPeer{}, false
	}
	return local, true
}

func (r *PeerRegistry) removeLocalIndexesLocked(local LocalPeer) {
	key := peerChannelKey(local.WorkspaceID, local.Channel)
	channelEntries := r.localsByChannel[key]
	if len(channelEntries) == 0 {
		return
	}
	delete(channelEntries, local.PeerID)
	if len(channelEntries) == 0 {
		delete(r.localsByChannel, key)
	}
}

func (r *PeerRegistry) deleteRemoteLocked(workspaceID string, channel string, peerID string) {
	key := peerChannelKey(workspaceID, channel)
	entries := r.remotesByChannel[key]
	if len(entries) == 0 {
		return
	}
	delete(entries, peerID)
	if len(entries) == 0 {
		delete(r.remotesByChannel, key)
	}
}

func (r *PeerRegistry) expireRemotesLocked(at time.Time) {
	for channel, entries := range r.remotesByChannel {
		for peerID, entry := range entries {
			if !entry.ExpiresAt.After(at) {
				delete(entries, peerID)
			}
		}
		if len(entries) == 0 {
			delete(r.remotesByChannel, channel)
		}
	}
}

func normalizePeerCard(card PeerCard) (PeerCard, error) {
	normalized := clonePeerCard(card)
	if err := normalizeAndValidatePeerCard(&normalized); err != nil {
		return PeerCard{}, err
	}
	return normalized, nil
}

func clonePeerCard(card PeerCard) PeerCard {
	cloned := PeerCard{
		PeerID:              strings.TrimSpace(card.PeerID),
		ProfilesSupported:   cloneStringList(card.ProfilesSupported),
		Capabilities:        cloneStringList(card.Capabilities),
		ArtifactsSupported:  cloneStringList(card.ArtifactsSupported),
		TrustModesSupported: cloneStringList(card.TrustModesSupported),
		Ext:                 cloneExtensionMap(card.Ext),
	}
	if card.DisplayName != nil {
		displayName := strings.TrimSpace(*card.DisplayName)
		cloned.DisplayName = &displayName
	}
	return cloned
}

func cloneStringList(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneLocalPeer(local LocalPeer) LocalPeer {
	return LocalPeer{
		SessionID:         strings.TrimSpace(local.SessionID),
		PeerID:            strings.TrimSpace(local.PeerID),
		WorkspaceID:       strings.TrimSpace(local.WorkspaceID),
		Channel:           strings.TrimSpace(local.Channel),
		PeerCard:          clonePeerCard(local.PeerCard),
		CapabilityCatalog: cloneNetworkPeerCapabilityCatalog(local.CapabilityCatalog),
		JoinedAt:          local.JoinedAt.UTC(),
	}
}

func cloneRemotePeerEntry(entry RemotePeerEntry) RemotePeerEntry {
	return RemotePeerEntry{
		PeerID:                 strings.TrimSpace(entry.PeerID),
		PeerCard:               clonePeerCard(entry.PeerCard),
		WorkspaceID:            strings.TrimSpace(entry.WorkspaceID),
		Channel:                strings.TrimSpace(entry.Channel),
		CapabilityCatalog:      cloneNetworkPeerCapabilityCatalog(entry.CapabilityCatalog),
		CapabilityCatalogKnown: entry.CapabilityCatalogKnown,
		LastSeen:               entry.LastSeen.UTC(),
		ExpiresAt:              entry.ExpiresAt.UTC(),
	}
}

func peerInfoFromLocal(local LocalPeer) PeerInfo {
	sessionID := strings.TrimSpace(local.SessionID)
	joinedAt := local.JoinedAt.UTC()
	return PeerInfo{
		SessionID:              &sessionID,
		PeerID:                 local.PeerID,
		WorkspaceID:            local.WorkspaceID,
		Channel:                local.Channel,
		Local:                  true,
		PeerCard:               clonePeerCard(local.PeerCard),
		CapabilityCatalog:      cloneNetworkPeerCapabilityCatalog(local.CapabilityCatalog),
		CapabilityCatalogKnown: true,
		JoinedAt:               &joinedAt,
	}
}

func peerInfoFromRemote(entry RemotePeerEntry) PeerInfo {
	lastSeen := entry.LastSeen.UTC()
	expiresAt := entry.ExpiresAt.UTC()
	return PeerInfo{
		PeerID:                 entry.PeerID,
		WorkspaceID:            entry.WorkspaceID,
		Channel:                entry.Channel,
		Local:                  false,
		PeerCard:               clonePeerCard(entry.PeerCard),
		CapabilityCatalog:      cloneNetworkPeerCapabilityCatalog(entry.CapabilityCatalog),
		CapabilityCatalogKnown: entry.CapabilityCatalogKnown,
		LastSeen:               &lastSeen,
		ExpiresAt:              &expiresAt,
	}
}

func nextRemoteCapabilityCatalog(
	existing RemotePeerEntry,
	hasExisting bool,
	capabilityIDs []string,
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
	capabilityCatalogKnown bool,
) ([]sessionpkg.NetworkPeerCapability, bool) {
	if capabilityCatalogKnown {
		if !capabilityCatalogAlignsWithCapabilityIDs(capabilityIDs, capabilityCatalog) {
			return nil, false
		}
		return cloneNetworkPeerCapabilityCatalog(capabilityCatalog), true
	}
	if !hasExisting || !existing.CapabilityCatalogKnown {
		return nil, false
	}
	if !sameCapabilityIDSequence(capabilityIDs, existing.PeerCard.Capabilities) {
		return nil, false
	}
	return cloneNetworkPeerCapabilityCatalog(existing.CapabilityCatalog), true
}

func sameCapabilityIDSequence(left []string, right []string) bool {
	normalizedLeft := normalizeCapabilityIDList(left)
	normalizedRight := normalizeCapabilityIDList(right)
	if len(normalizedLeft) != len(normalizedRight) {
		return false
	}
	for idx := range normalizedLeft {
		if normalizedLeft[idx] != normalizedRight[idx] {
			return false
		}
	}
	return true
}

func matchesWhoisQuery(card PeerCard, query string) bool {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return true
	}
	if card.PeerID == trimmedQuery {
		return true
	}
	if card.DisplayName != nil && strings.TrimSpace(*card.DisplayName) == trimmedQuery {
		return true
	}
	return containsString(card.Capabilities, trimmedQuery) ||
		containsString(card.ProfilesSupported, trimmedQuery) ||
		containsString(card.ArtifactsSupported, trimmedQuery) ||
		containsString(card.TrustModesSupported, trimmedQuery)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func listPeersForChannelLocked(r *PeerRegistry, workspaceID string, channel string) []PeerInfo {
	key := peerChannelKey(workspaceID, channel)
	sessionIDs := r.localsByChannel[key]
	remoteEntries := r.remotesByChannel[key]
	if len(sessionIDs) == 0 && len(remoteEntries) == 0 {
		return nil
	}

	peers := make([]PeerInfo, 0, len(sessionIDs)+len(remoteEntries))
	for _, sessionID := range sessionIDs {
		local, ok := r.localsByID[sessionID]
		if !ok {
			continue
		}
		peers = append(peers, peerInfoFromLocal(local))
	}
	for _, entry := range remoteEntries {
		peers = append(peers, peerInfoFromRemote(entry))
	}
	sortPeerInfos(peers)
	return peers
}

func sortPeerInfos(peers []PeerInfo) {
	sort.Slice(peers, func(i int, j int) bool {
		if peers[i].WorkspaceID != peers[j].WorkspaceID {
			return peers[i].WorkspaceID < peers[j].WorkspaceID
		}
		if peers[i].Channel != peers[j].Channel {
			return peers[i].Channel < peers[j].Channel
		}
		if peers[i].Local != peers[j].Local {
			return peers[i].Local
		}
		return peers[i].PeerID < peers[j].PeerID
	})
}

func peerChannelKey(workspaceID string, channel string) string {
	return strings.TrimSpace(workspaceID) + "\x00" + strings.TrimSpace(channel)
}

func splitPeerChannelKey(key string) (string, string) {
	workspaceID, channel, ok := strings.Cut(key, "\x00")
	if !ok {
		return "", strings.TrimSpace(key)
	}
	return strings.TrimSpace(workspaceID), strings.TrimSpace(channel)
}
