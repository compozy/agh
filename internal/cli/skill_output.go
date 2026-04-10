package cli

import (
	"strconv"

	"github.com/pedronauck/agh/internal/skills/marketplace"
)

func skillSearchBundle(items []marketplace.SkillListing) outputBundle {
	return listBundle(
		items,
		items,
		"Marketplace Skills",
		[]string{"Slug", "Name", "Description", "Author", "Version", "Downloads"},
		"skills",
		[]string{"slug", "name", "description", "author", "version", "downloads"},
		func(item marketplace.SkillListing) []string {
			return []string{
				stringOrDash(item.Slug),
				stringOrDash(item.Name),
				stringOrDash(item.Description),
				stringOrDash(item.Author),
				stringOrDash(item.Version),
				strconv.Itoa(item.Downloads),
			}
		},
		func(item marketplace.SkillListing) []string {
			return []string{
				item.Slug,
				item.Name,
				item.Description,
				item.Author,
				item.Version,
				strconv.Itoa(item.Downloads),
			}
		},
	)
}

func skillListBundle(items []skillListItem) outputBundle {
	return listBundle(
		items,
		items,
		"Skills",
		[]string{"Name", "Description", "Source", "Enabled"},
		"skills",
		[]string{"name", "description", "source", "enabled"},
		func(item skillListItem) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Description),
				stringOrDash(item.Source),
				strconv.FormatBool(item.Enabled),
			}
		},
		func(item skillListItem) []string {
			return []string{
				item.Name,
				item.Description,
				item.Source,
				strconv.FormatBool(item.Enabled),
			}
		},
	)
}

func skillViewBundle(item skillViewItem, rendered string) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return rendered, nil
		},
		toon: func() (string, error) {
			return rendered, nil
		},
	}
}

func skillInfoBundle(item skillInfoItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			base := renderHumanSection("Skill", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Description", Value: stringOrDash(item.Description)},
				{Label: "Version", Value: stringOrDash(item.Version)},
				{Label: "Source", Value: stringOrDash(item.Source)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Enabled", Value: strconv.FormatBool(item.Enabled)},
			})

			metadataRows := make([][]string, 0, len(item.Metadata))
			for _, entry := range sortedSkillMetadataEntries(item.Metadata) {
				metadataRows = append(metadataRows, []string{entry.Label, entry.Value})
			}
			metadata := renderHumanTable("Metadata", []string{"Key", "Value"}, metadataRows)

			resourceRows := make([][]string, 0, len(item.Resources))
			for _, resource := range item.Resources {
				resourceRows = append(resourceRows, []string{resource})
			}
			resources := renderHumanTable("Resources", []string{"Path"}, resourceRows)

			return renderHumanBlocks(base, metadata, resources), nil
		},
		toon: func() (string, error) {
			metadataRows := make([][]string, 0, len(item.Metadata))
			for _, entry := range sortedSkillMetadataEntries(item.Metadata) {
				metadataRows = append(metadataRows, []string{entry.Label, entry.Value})
			}

			resourceRows := make([][]string, 0, len(item.Resources))
			for _, resource := range item.Resources {
				resourceRows = append(resourceRows, []string{resource})
			}

			return renderHumanBlocks(
				renderToonObject("skill", []string{"name", "description", "version", "source", "path", "enabled"}, []string{
					item.Name,
					item.Description,
					item.Version,
					item.Source,
					item.Path,
					strconv.FormatBool(item.Enabled),
				}),
				renderToonArray("metadata", []string{"key", "value"}, metadataRows),
				renderToonArray("resources", []string{"path"}, resourceRows),
			), nil
		},
	}
}

func skillCreateBundle(item skillCreateItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Skill", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Source", Value: stringOrDash(item.Source)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "File", Value: stringOrDash(item.File)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("skill", []string{"name", "source", "path", "file", "status"}, []string{
				item.Name,
				item.Source,
				item.Path,
				item.File,
				item.Status,
			}), nil
		},
	}
}

func skillInstallBundle(item skillInstallItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Skill Install", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Slug", Value: stringOrDash(item.Slug)},
				{Label: "Version", Value: stringOrDash(item.Version)},
				{Label: "Registry", Value: stringOrDash(item.Registry)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Hash", Value: stringOrDash(item.Hash)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("skill_install", []string{"name", "slug", "version", "registry", "path", "hash", "status"}, []string{
				item.Name,
				item.Slug,
				item.Version,
				item.Registry,
				item.Path,
				item.Hash,
				item.Status,
			}), nil
		},
	}
}

func skillRemoveBundle(item skillRemoveItem) outputBundle {
	return outputBundle{
		jsonValue: item,
		human: func() (string, error) {
			return renderHumanSection("Skill Remove", []keyValue{
				{Label: "Name", Value: stringOrDash(item.Name)},
				{Label: "Slug", Value: stringOrDash(item.Slug)},
				{Label: "Path", Value: stringOrDash(item.Path)},
				{Label: "Status", Value: stringOrDash(item.Status)},
			}), nil
		},
		toon: func() (string, error) {
			return renderToonObject("skill_remove", []string{"name", "slug", "path", "status"}, []string{
				item.Name,
				item.Slug,
				item.Path,
				item.Status,
			}), nil
		},
	}
}

func skillUpdateBundle(items []skillUpdateItem) outputBundle {
	return listBundle(
		items,
		items,
		"Skill Updates",
		[]string{"Name", "Slug", "Current", "Latest", "Path", "Status"},
		"skill_updates",
		[]string{"name", "slug", "current_version", "latest_version", "path", "status"},
		func(item skillUpdateItem) []string {
			return []string{
				stringOrDash(item.Name),
				stringOrDash(item.Slug),
				stringOrDash(item.CurrentVersion),
				stringOrDash(item.LatestVersion),
				stringOrDash(item.Path),
				stringOrDash(item.Status),
			}
		},
		func(item skillUpdateItem) []string {
			return []string{
				item.Name,
				item.Slug,
				item.CurrentVersion,
				item.LatestVersion,
				item.Path,
				item.Status,
			}
		},
	)
}
