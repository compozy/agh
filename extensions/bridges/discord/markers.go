package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	bridgepkg "github.com/compozy/agh/internal/bridges"
	extensioncontract "github.com/compozy/agh/internal/extension/contract"
	"github.com/compozy/agh/internal/subprocess"
)

const (
	adapterHandshakeEnv = "AGH_BRIDGE_ADAPTER_HANDSHAKE_PATH"
	adapterOwnershipEnv = "AGH_BRIDGE_ADAPTER_OWNERSHIP_PATH"
	adapterStateEnv     = "AGH_BRIDGE_ADAPTER_STATE_PATH"
	adapterDeliveryEnv  = "AGH_BRIDGE_ADAPTER_DELIVERY_PATH"
	adapterIngestEnv    = "AGH_BRIDGE_ADAPTER_INGEST_PATH"
	adapterStartsEnv    = "AGH_BRIDGE_ADAPTER_STARTS_PATH"
	adapterShutdownEnv  = "AGH_BRIDGE_ADAPTER_SHUTDOWN_PATH"
	adapterCrashOnceEnv = "AGH_BRIDGE_ADAPTER_CRASH_ONCE_PATH"
)

type markerEnv struct {
	handshakePath string
	ownershipPath string
	statePath     string
	deliveryPath  string
	ingestPath    string
	startsPath    string
	shutdownPath  string
	crashOncePath string
}

type initializeMarker struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
}

type ownershipMarker struct {
	Listed  []bridgepkg.BridgeInstance `json:"listed,omitempty"`
	Fetched []bridgepkg.BridgeInstance `json:"fetched,omitempty"`
	Error   string                     `json:"error,omitempty"`
}

type deliveryMarker struct {
	PID     int                       `json:"pid"`
	Request bridgepkg.DeliveryRequest `json:"request"`
	Ack     *bridgepkg.DeliveryAck    `json:"ack,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

type stateMarker struct {
	BridgeInstanceID string                   `json:"bridge_instance_id,omitempty"`
	Status           bridgepkg.BridgeStatus   `json:"status"`
	Instance         bridgepkg.BridgeInstance `json:"instance"`
	Error            string                   `json:"error,omitempty"`
}

type ingestMarker struct {
	Envelope bridgepkg.InboundMessageEnvelope              `json:"envelope"`
	Result   extensioncontract.BridgesMessagesIngestResult `json:"result"`
	Error    string                                        `json:"error,omitempty"`
}

func markerEnvFromProcess() markerEnv {
	return markerEnv{
		handshakePath: strings.TrimSpace(os.Getenv(adapterHandshakeEnv)),
		ownershipPath: strings.TrimSpace(os.Getenv(adapterOwnershipEnv)),
		statePath:     strings.TrimSpace(os.Getenv(adapterStateEnv)),
		deliveryPath:  strings.TrimSpace(os.Getenv(adapterDeliveryEnv)),
		ingestPath:    strings.TrimSpace(os.Getenv(adapterIngestEnv)),
		startsPath:    strings.TrimSpace(os.Getenv(adapterStartsEnv)),
		shutdownPath:  strings.TrimSpace(os.Getenv(adapterShutdownEnv)),
		crashOncePath: strings.TrimSpace(os.Getenv(adapterCrashOnceEnv)),
	}
}

func appendMarkerLine(path string, line string) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	_, err = fmt.Fprintln(file, strings.TrimSpace(line))
	return err
}

func appendJSONLine(path string, value any) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func writeJSONFile(path string, value any) error {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(target, payload, 0o600)
}

func reportSideEffectError(writer io.Writer, action string, err error) {
	if err == nil || writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, "discord: %s: %v\n", strings.TrimSpace(action), err)
}

func shouldCrashOnce(path string) bool {
	target := strings.TrimSpace(path)
	if target == "" {
		return false
	}
	_, err := os.Stat(target)
	return os.IsNotExist(err)
}
