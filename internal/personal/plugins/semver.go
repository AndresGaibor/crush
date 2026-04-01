package plugins

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type semverVersion struct {
	major      int
	minor      int
	patch      int
	prerelease string
	hasPre     bool
}

func parseSemverVersion(input string) (semverVersion, error) {
	input = strings.TrimSpace(strings.TrimPrefix(input, "v"))
	if input == "" {
		return semverVersion{}, fmt.Errorf("empty version")
	}

	main, prerelease, _ := strings.Cut(input, "-")
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return semverVersion{}, fmt.Errorf("version %q must be major.minor.patch", input)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semverVersion{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semverVersion{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semverVersion{}, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	return semverVersion{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: prerelease,
		hasPre:     prerelease != "",
	}, nil
}

func compareSemverVersions(a, b semverVersion) int {
	switch {
	case a.major != b.major:
		if a.major < b.major {
			return -1
		}
		return 1
	case a.minor != b.minor:
		if a.minor < b.minor {
			return -1
		}
		return 1
	case a.patch != b.patch:
		if a.patch < b.patch {
			return -1
		}
		return 1
	}

	switch {
	case a.hasPre && !b.hasPre:
		return -1
	case !a.hasPre && b.hasPre:
		return 1
	case a.prerelease == b.prerelease:
		return 0
	case a.prerelease < b.prerelease:
		return -1
	default:
		return 1
	}
}

func matchVersionRequirement(requirement, version string) bool {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" || version == "" {
		return false
	}

	if strings.Contains(requirement, "||") {
		for _, part := range strings.Split(requirement, "||") {
			if matchVersionRequirement(part, version) {
				return true
			}
		}
		return false
	}

	for _, part := range splitVersionRequirementParts(requirement) {
		if !matchSingleVersionRequirement(part, version) {
			return false
		}
	}
	return true
}

func matchSingleVersionRequirement(requirement, version string) bool {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return false
	}

	current, err := parseSemverVersion(version)
	if err != nil {
		return false
	}

	switch {
	case strings.HasPrefix(requirement, ">="):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, ">=")))
		return err == nil && compareSemverVersions(current, target) >= 0
	case strings.HasPrefix(requirement, "<="):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "<=")))
		return err == nil && compareSemverVersions(current, target) <= 0
	case strings.HasPrefix(requirement, ">"):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, ">")))
		return err == nil && compareSemverVersions(current, target) > 0
	case strings.HasPrefix(requirement, "<"):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "<")))
		return err == nil && compareSemverVersions(current, target) < 0
	case strings.HasPrefix(requirement, "="):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "=")))
		return err == nil && compareSemverVersions(current, target) == 0
	case strings.HasPrefix(requirement, "^"):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "^")))
		return err == nil && caretMatches(current, target)
	case strings.HasPrefix(requirement, "~"):
		target, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "~")))
		return err == nil && tildeMatches(current, target)
	default:
		target, err := parseSemverVersion(requirement)
		return err == nil && compareSemverVersions(current, target) == 0
	}
}

func splitVersionRequirementParts(requirement string) []string {
	parts := strings.FieldsFunc(requirement, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func caretMatches(current, target semverVersion) bool {
	if compareSemverVersions(current, target) < 0 {
		return false
	}

	switch {
	case target.major > 0:
		return current.major == target.major
	case target.minor > 0:
		return current.major == 0 && current.minor == target.minor
	default:
		return current.major == 0 && current.minor == 0 && current.patch == target.patch
	}
}

func tildeMatches(current, target semverVersion) bool {
	if compareSemverVersions(current, target) < 0 {
		return false
	}
	return current.major == target.major && current.minor == target.minor
}

func validateVersionRequirement(requirement string) error {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return fmt.Errorf("empty version requirement")
	}

	if strings.Contains(requirement, "||") {
		parts := strings.Split(requirement, "||")
		if len(parts) == 0 {
			return fmt.Errorf("invalid version requirement %q", requirement)
		}
		for _, part := range parts {
			if err := validateVersionRequirement(part); err != nil {
				return err
			}
		}
		return nil
	}

	parts := splitVersionRequirementParts(requirement)
	if len(parts) == 0 {
		return fmt.Errorf("invalid version requirement %q", requirement)
	}

	for _, part := range parts {
		if err := validateSingleVersionRequirement(part); err != nil {
			return err
		}
	}
	return nil
}

func validateSingleVersionRequirement(requirement string) error {
	requirement = strings.TrimSpace(requirement)
	if requirement == "" {
		return fmt.Errorf("empty version requirement")
	}

	switch {
	case strings.HasPrefix(requirement, ">="):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, ">=")))
		return err
	case strings.HasPrefix(requirement, "<="):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "<=")))
		return err
	case strings.HasPrefix(requirement, ">"):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, ">")))
		return err
	case strings.HasPrefix(requirement, "<"):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "<")))
		return err
	case strings.HasPrefix(requirement, "="):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "=")))
		return err
	case strings.HasPrefix(requirement, "^"):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "^")))
		return err
	case strings.HasPrefix(requirement, "~"):
		_, err := parseSemverVersion(strings.TrimSpace(strings.TrimPrefix(requirement, "~")))
		return err
	default:
		_, err := parseSemverVersion(requirement)
		return err
	}
}
