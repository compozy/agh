package skills

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/filesnap"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type wsCache struct {
	skills     map[string]*Skill
	snapshots  map[string]filesnap.Snapshot
	lastAccess time.Time
}

type workspaceLoad struct {
	paths     []workspaceSkillPath
	snapshots map[string]filesnap.Snapshot
}

type workspaceSkillPath struct {
	filePath string
	source   SkillSource
}

func (r *Registry) workspaceDisabledSkillsSnapshot(cacheKey string, configured []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	disabledSkills := mergeDisabledSkills(r.cfg.DisabledSkills, configured)
	if cacheKey == "" {
		return disabledSkills
	}
	return mergeDisabledSkills(disabledSkills, r.workspaceDisabled[cacheKey])
}

func (r *Registry) workspaceSkillTargetLocked(name string, resolved *workspacepkg.ResolvedWorkspace) (string, *Skill) {
	if resolved == nil {
		return "", nil
	}

	cacheKey := workspaceCacheKey(*resolved, nil)
	if cacheKey == "" {
		return "", nil
	}

	cached := r.wsCache[cacheKey]
	if cached == nil {
		return cacheKey, nil
	}

	return cacheKey, cached.skills[name]
}

func (r *Registry) workspaceLoadFromResolved(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) (workspaceLoad, error) {
	load := workspaceLoad{
		paths:     make([]workspaceSkillPath, 0, len(resolved.Skills)),
		snapshots: make(map[string]filesnap.Snapshot, len(resolved.Skills)),
	}

	for _, skillPath := range resolved.Skills {
		if err := checkRegistryContext(ctx); err != nil {
			return workspaceLoad{}, err
		}

		source, include, err := skillSourceFromWorkspacePath(skillPath.Source)
		if err != nil {
			return workspaceLoad{}, err
		}
		if !include {
			continue
		}

		skillDir := strings.TrimSpace(skillPath.Dir)
		if skillDir == "" {
			continue
		}

		skillFile := filepath.Join(skillDir, skillFileName)
		snapshot, err := filesnap.FromPath(skillFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return workspaceLoad{}, fmt.Errorf("skills: snapshot workspace skill %q: %w", skillFile, err)
		}

		load.snapshots[skillFile] = snapshot
		load.paths = append(load.paths, workspaceSkillPath{
			filePath: skillFile,
			source:   source,
		})
	}

	return load, nil
}

func (r *Registry) evictExpiredWorkspaceLocked(now time.Time) {
	cutoff := now.Add(-workspaceCacheTTL)
	for workspace, entry := range r.wsCache {
		if entry.lastAccess.Before(cutoff) {
			delete(r.wsCache, workspace)
		}
	}
}

func workspaceCacheKey(resolved workspacepkg.ResolvedWorkspace, paths []workspaceSkillPath) string {
	if id := strings.TrimSpace(resolved.ID); id != "" {
		return "id:" + id
	}
	if root := strings.TrimSpace(resolved.RootDir); root != "" {
		return "root:" + root
	}
	if len(paths) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, path := range paths {
		if builder.Len() > 0 {
			builder.WriteByte('|')
		}
		builder.WriteString(skillSourceName(path.source))
		builder.WriteByte(':')
		builder.WriteString(path.filePath)
	}

	return builder.String()
}
