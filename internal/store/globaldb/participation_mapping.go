package globaldb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/network/participation"
)

type participationSnapshotFields struct {
	JSON    string
	Mode    string
	Channel sql.NullString
	Source  string
}

func encodeParticipationSnapshot(spec participation.Spec) (participationSnapshotFields, error) {
	if spec == (participation.Spec{}) {
		spec = participation.LocalSpec()
	}
	if err := participation.ValidateSpec(spec); err != nil {
		return participationSnapshotFields{}, fmt.Errorf("store: validate network participation snapshot: %w", err)
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return participationSnapshotFields{}, fmt.Errorf("store: encode network participation snapshot: %w", err)
	}
	return participationSnapshotFields{
		JSON:    string(raw),
		Mode:    string(spec.Mode),
		Channel: nullableParticipationChannel(spec.ChannelID),
		Source:  string(spec.Source),
	}, nil
}

func nullableParticipationChannel(channel string) sql.NullString {
	channel = strings.TrimSpace(channel)
	return sql.NullString{String: channel, Valid: channel != ""}
}

func decodeParticipationSnapshot(
	raw string,
	mode string,
	channel sql.NullString,
	source string,
) (participation.Spec, error) {
	var spec participation.Spec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return participation.Spec{}, fmt.Errorf("store: decode network participation snapshot: %w", err)
	}
	if err := participation.ValidateSpec(spec); err != nil {
		return participation.Spec{}, fmt.Errorf("store: validate network participation snapshot: %w", err)
	}
	projectedChannel := ""
	if channel.Valid {
		projectedChannel = strings.TrimSpace(channel.String)
	}
	if string(spec.Mode) != strings.TrimSpace(mode) ||
		spec.ChannelID != projectedChannel ||
		string(spec.Source) != strings.TrimSpace(source) {
		return participation.Spec{}, fmt.Errorf(
			"store: network participation snapshot projections do not match snapshot",
		)
	}
	return spec, nil
}
