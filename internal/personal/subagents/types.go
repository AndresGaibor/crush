package subagents

import "strings"

// Visibility indica cómo debe tratarse un subagente.
type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

// Subagent describe una definición de subagente especializada.
type Subagent struct {
	Name         string     `yaml:"name" json:"name"`
	Description  string     `yaml:"description" json:"description"`
	Model        string     `yaml:"model,omitempty" json:"model,omitempty"`
	Tools        []string   `yaml:"tools,omitempty" json:"tools,omitempty"`
	AutoDelegate bool       `yaml:"auto_delegate,omitempty" json:"auto_delegate,omitempty"`
	Visibility   Visibility `yaml:"visibility,omitempty" json:"visibility,omitempty"`
	Instructions string     `yaml:"-" json:"instructions"`
	Path         string     `yaml:"-" json:"path"`
	FilePath     string     `yaml:"-" json:"file_path"`
}

// ToolSet returns a normalized set of tool names.
func (s *Subagent) ToolSet() map[string]struct{} {
	set := make(map[string]struct{}, len(s.Tools))
	for _, name := range s.Tools {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}
