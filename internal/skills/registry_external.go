package skills

import (
	"errors"
	"slices"
	"strings"
)

// RegisterExternal replaces the external skill set for one owner key. The
// owner should be stable for the lifetime of the external source, such as an
// extension name.
func (r *Registry) RegisterExternal(owner string, skills []*Skill) error {
	if r == nil {
		return errors.New("skills: registry is required")
	}

	trimmedOwner := strings.TrimSpace(owner)
	if trimmedOwner == "" {
		return errors.New("skills: external owner is required")
	}

	registered := make(map[string]*Skill)
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		name := strings.TrimSpace(skill.Meta.Name)
		if name == "" {
			continue
		}
		registered[name] = cloneSkill(skill)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.externalSkills == nil {
		r.externalSkills = make(map[string]map[string]*Skill)
	}

	if len(registered) == 0 {
		delete(r.externalSkills, trimmedOwner)
	} else {
		r.externalSkills[trimmedOwner] = registered
	}
	r.globalVersion.Add(1)

	return nil
}

// RemoveExternal removes all externally registered skills for one owner key.
func (r *Registry) RemoveExternal(owner string) {
	if r == nil {
		return
	}

	trimmedOwner := strings.TrimSpace(owner)
	if trimmedOwner == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.externalSkills == nil {
		return
	}
	delete(r.externalSkills, trimmedOwner)
	r.globalVersion.Add(1)
}

func (r *Registry) lookupSkillLocked(name string) (*Skill, bool) {
	if r == nil {
		return nil, false
	}

	if external := r.externalSkillSetLocked(); external != nil {
		if skill := external[name]; skill != nil {
			return skill, true
		}
	}

	skill, ok := r.globalSkills[name]
	return skill, ok
}

func (r *Registry) externalSkillSetLocked() map[string]*Skill {
	if r == nil || len(r.externalSkills) == 0 {
		return nil
	}

	owners := make([]string, 0, len(r.externalSkills))
	for owner := range r.externalSkills {
		owners = append(owners, owner)
	}
	slices.Sort(owners)

	merged := make(map[string]*Skill)
	for _, owner := range owners {
		for name, skill := range r.externalSkills[owner] {
			merged[name] = skill
		}
	}
	return merged
}

func mergeSkillMaps(base map[string]*Skill, overlay map[string]*Skill) map[string]*Skill {
	switch {
	case len(base) == 0 && len(overlay) == 0:
		return nil
	case len(overlay) == 0:
		return base
	case len(base) == 0:
		return overlay
	}

	merged := make(map[string]*Skill, len(base)+len(overlay))
	for name, skill := range base {
		merged[name] = skill
	}
	for name, skill := range overlay {
		merged[name] = skill
	}
	return merged
}
