package marketplace

import "io"

// SkillListing is the summary entry returned by marketplace search endpoints.
type SkillListing struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Downloads   int    `json:"downloads"`
}

// SkillArchive is a downloadable skill package stream.
type SkillArchive struct {
	Slug    string
	Version string
	Data    io.ReadCloser
}

// SkillDetail is the full skill metadata returned by marketplace info endpoints.
type SkillDetail struct {
	SkillListing
	Readme     string   `json:"readme"`
	MCPServers []string `json:"mcp_servers,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// SearchOpts controls marketplace search pagination.
type SearchOpts struct {
	Limit  int
	Offset int
}
