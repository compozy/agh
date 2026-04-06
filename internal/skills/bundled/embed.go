package bundled

import (
	"embed"
	"io/fs"
)

// embeddedSkills stores the bundled starter skills compiled into the binary.
//
//go:embed skills/**/SKILL.md
var embeddedSkills embed.FS

// FS returns the bundled skills filesystem compiled into the AGH binary.
func FS() fs.FS {
	return embeddedSkills
}
