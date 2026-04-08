# Outside-of-diff from Comment 3

**File:** `internal/skills/registry.go`
**Date:** 2026-04-08 12:09:56 America/Sao_Paulo
**Status:** - [x] RESOLVED

## Triage

- Disposition: `VALID`
- Notes: the current registry behavior logged marketplace hash mismatches but still loaded clean-looking tampered skills, which left a real integrity gap. The fix changes marketplace verification to fail closed: hash mismatches are still logged, content warnings are still recorded for observability, but the tampered marketplace skill is not overlaid into the active registry.

## Details

<details>
> <summary>internal/skills/registry.go (1)</summary><blockquote>
> 
> `317-337`: _⚠️ Potential issue_ | _🔴 Critical_
> 
> **Fail closed when marketplace provenance verification fails.**
> 
> Once a skill is reclassified as `SourceMarketplace`, a hash mismatch only produces a warning and the skill is still overlaid into the active registry. That leaves a tampered marketplace install able to contribute prompt content, hooks, and MCP declarations.
> 
> <details>
> <summary>🛡 Proposed fix</summary>
> 
> ```diff
>  func (r *Registry) processSkill(dst map[string]*Skill, skill *Skill) bool {
>  	r.applyDisabled(skill)
>  
> -	hashMismatch := r.verifyMarketplaceSkill(skill)
> -	// Content warnings are derived from the current file body; marketplace hash
> -	// mismatches only add provenance-specific logging above.
> +	if err := r.verifyMarketplaceSkill(skill); err != nil {
> +		return false
> +	}
> +
> +	// Content warnings are derived from the current file body.
>  	warnings := VerifyContent(skill.Content)
> -	if hashMismatch {
> -		r.logger.Debug(
> -			"skills: reusing content verification warnings after marketplace hash mismatch",
> -			"skill_name", skill.Meta.Name,
> -			"path", skill.FilePath,
> -		)
> -	}
>  	r.logVerificationWarnings(skill, warnings)
>  	if hasCriticalWarning(warnings) {
>  		return false
>  	}
> @@
> -func (r *Registry) verifyMarketplaceSkill(skill *Skill) bool {
> +func (r *Registry) verifyMarketplaceSkill(skill *Skill) error {
>  	if skill == nil || skill.Source != SourceMarketplace || skill.Provenance == nil {
> -		return false
> +		return nil
>  	}
>  
>  	err := VerifyHash(skill.Dir, skill.Provenance)
>  	if err == nil {
> -		return false
> +		return nil
>  	}
> @@
> -		return true
> +		return err
>  	}
> @@
> -	return false
> +	return err
>  }
> ```
> </details>
> 
> 
> 
> Also applies to: 370-399
> 
> <details>
> <summary>🤖 Prompt for AI Agents</summary>
> 
> ```
> Verify each finding against the current code and only fix it if needed.
> 
> In `@internal/skills/registry.go` around lines 317 - 337, processSkill currently
> treats a marketplace provenance hash mismatch as only a warning and still
> overlays the skill; change this to fail-closed: after calling
> verifyMarketplaceSkill (the hashMismatch variable) if hashMismatch is true, log
> a provenance verification error (use r.logger.* with skill.Meta.Name and
> skill.FilePath) and return false before calling r.overlaySkill so tampered
> marketplace installs are not applied. Apply the same change to the corresponding
> logic block referenced at the other occurrence (lines 370-399) where
> verifyMarketplaceSkill is used, ensuring both paths abort overlay when
> marketplace verification fails and update any tests that assumed the old
> behavior.
> ```
> 
> </details>
> 
> </blockquote></details>
