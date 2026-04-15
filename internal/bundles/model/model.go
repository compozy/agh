package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrActivationNotFound = errors.New("bundles: activation not found")
)

type Scope string

const (
	ScopeGlobal    Scope = "global"
	ScopeWorkspace Scope = "workspace"
)

func (s Scope) Normalize() Scope {
	return Scope(strings.ToLower(strings.TrimSpace(string(s))))
}

func (s Scope) Validate(workspaceID string) error {
	switch s.Normalize() {
	case ScopeGlobal:
		if strings.TrimSpace(workspaceID) != "" {
			return errors.New("bundles: global activation cannot include workspace id")
		}
		return nil
	case ScopeWorkspace:
		if strings.TrimSpace(workspaceID) == "" {
			return errors.New("bundles: workspace activation requires workspace id")
		}
		return nil
	default:
		return fmt.Errorf("bundles: unsupported scope %q", s)
	}
}

type Activation struct {
	ID                          string
	ExtensionName               string
	BundleName                  string
	ProfileName                 string
	Scope                       Scope
	WorkspaceID                 string
	SpecContentHash             string
	BindPrimaryChannelAsDefault bool
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

func (a Activation) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return errors.New("bundles: activation id is required")
	}
	if strings.TrimSpace(a.ExtensionName) == "" {
		return errors.New("bundles: activation extension name is required")
	}
	if strings.TrimSpace(a.BundleName) == "" {
		return errors.New("bundles: activation bundle name is required")
	}
	if strings.TrimSpace(a.ProfileName) == "" {
		return errors.New("bundles: activation profile name is required")
	}
	return a.Scope.Validate(a.WorkspaceID)
}

type InventoryItem struct {
	ActivationID  string
	ResourceKind  string
	ResourceID    string
	ResourceName  string
	RecordedAtUTC time.Time
}

func (i InventoryItem) Validate() error {
	if strings.TrimSpace(i.ActivationID) == "" {
		return errors.New("bundles: inventory activation id is required")
	}
	if strings.TrimSpace(i.ResourceKind) == "" {
		return errors.New("bundles: inventory resource kind is required")
	}
	if strings.TrimSpace(i.ResourceID) == "" {
		return errors.New("bundles: inventory resource id is required")
	}
	if strings.TrimSpace(i.ResourceName) == "" {
		return errors.New("bundles: inventory resource name is required")
	}
	return nil
}
