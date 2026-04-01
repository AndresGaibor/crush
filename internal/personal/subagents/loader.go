package subagents

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/charlievieth/fastwalk"
	"gopkg.in/yaml.v3"
)

const (
	SubagentFileName      = "SUBAGENT.md"
	MaxSubagentNameLength = 64
	MaxDescriptionLength  = 1024
	MaxInstructionsLength = 32_000
)

var subagentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// Validate comprueba que el subagente cumpla los requisitos básicos.
func (s *Subagent) Validate() error {
	var errs []error

	if s.Name == "" {
		errs = append(errs, errors.New("name is required"))
	} else {
		if len(s.Name) > MaxSubagentNameLength {
			errs = append(errs, fmt.Errorf("name exceeds %d characters", MaxSubagentNameLength))
		}
		if !subagentNamePattern.MatchString(s.Name) {
			errs = append(errs, errors.New("name must be alphanumeric with hyphens, no leading/trailing/consecutive hyphens"))
		}
	}

	if s.Description == "" {
		errs = append(errs, errors.New("description is required"))
	} else if len(s.Description) > MaxDescriptionLength {
		errs = append(errs, fmt.Errorf("description exceeds %d characters", MaxDescriptionLength))
	}

	if strings.TrimSpace(s.Instructions) == "" {
		errs = append(errs, errors.New("instructions are required"))
	} else if len(s.Instructions) > MaxInstructionsLength {
		errs = append(errs, fmt.Errorf("instructions exceed %d characters", MaxInstructionsLength))
	}

	if s.Model != "" && s.Model != "large" && s.Model != "small" {
		errs = append(errs, errors.New("model must be empty, large, or small"))
	}

	switch s.Visibility {
	case "", VisibilityPrivate, VisibilityPublic:
	default:
		errs = append(errs, errors.New("visibility must be private or public"))
	}

	return errors.Join(errs...)
}

// Parse lee y valida una definición de subagente desde un archivo Markdown.
func Parse(path string) (*Subagent, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	frontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	var subagent Subagent
	if err := yaml.Unmarshal([]byte(frontmatter), &subagent); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	subagent.Instructions = strings.TrimSpace(body)
	subagent.Path = filepath.Dir(path)
	subagent.FilePath = path
	if subagent.Visibility == "" {
		subagent.Visibility = VisibilityPrivate
	}

	if err := subagent.Validate(); err != nil {
		return nil, fmt.Errorf("validating subagent %q: %w", subagent.Name, err)
	}

	return &subagent, nil
}

// Discover carga subagentes válidos desde una lista de directorios.
func Discover(paths []string) []*Subagent {
	var subagents []*Subagent
	var mu sync.Mutex
	seen := make(map[string]bool)

	for _, base := range paths {
		conf := fastwalk.Config{
			Follow:  true,
			ToSlash: fastwalk.DefaultToSlash(),
		}
		fastwalk.Walk(&conf, base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			mu.Lock()
			if seen[path] {
				mu.Unlock()
				return nil
			}
			seen[path] = true
			mu.Unlock()

			subagent, err := Parse(path)
			if err != nil {
				slog.Warn("Failed to parse subagent file", "path", path, "error", err)
				return nil
			}

			slog.Debug("Successfully loaded subagent", "name", subagent.Name, "path", path)
			mu.Lock()
			subagents = append(subagents, subagent)
			mu.Unlock()
			return nil
		})
	}

	return subagents
}

// splitFrontmatter extrae YAML frontmatter y el cuerpo Markdown.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("no YAML frontmatter found")
	}

	rest := strings.TrimPrefix(content, "---\n")
	before, after, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", "", errors.New("unclosed frontmatter")
	}

	return before, after, nil
}
