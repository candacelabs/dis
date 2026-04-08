package container

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseLabelsFromDockerfile reads a Dockerfile and extracts dis.* LABEL values.
// Supports both `LABEL key=value` and `LABEL key="value"` formats.
// Supports multi-label: `LABEL key1=val1 key2=val2`
// Supports continuation lines with trailing backslash.
func ParseLabelsFromDockerfile(dockerfilePath string) (map[string]string, error) {
	f, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer f.Close()

	labels := make(map[string]string)
	scanner := bufio.NewScanner(f)

	var continuation strings.Builder
	inContinuation := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if inContinuation {
			// Strip trailing backslash if present
			if strings.HasSuffix(trimmed, "\\") {
				trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, "\\"))
				continuation.WriteString(" ")
				continuation.WriteString(trimmed)
				continue
			}
			// Final line of continuation
			continuation.WriteString(" ")
			continuation.WriteString(trimmed)
			parseLabelLine(continuation.String(), labels)
			continuation.Reset()
			inContinuation = false
			continue
		}

		// Check if line starts with LABEL (case-insensitive)
		if !isLabelLine(trimmed) {
			continue
		}

		// Remove the LABEL prefix
		rest := strings.TrimSpace(trimmed[5:]) // len("LABEL") == 5

		// Check for continuation
		if strings.HasSuffix(rest, "\\") {
			rest = strings.TrimSpace(strings.TrimSuffix(rest, "\\"))
			continuation.Reset()
			continuation.WriteString(rest)
			inContinuation = true
			continue
		}

		parseLabelLine(rest, labels)
	}

	// Handle case where file ends mid-continuation
	if inContinuation {
		parseLabelLine(continuation.String(), labels)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Dockerfile: %w", err)
	}

	return labels, nil
}

// isLabelLine checks if a trimmed line starts with LABEL (case-insensitive).
func isLabelLine(trimmed string) bool {
	if len(trimmed) < 5 {
		return false
	}
	prefix := strings.ToUpper(trimmed[:5])
	if prefix != "LABEL" {
		return false
	}
	// Must be followed by whitespace or end of string
	if len(trimmed) == 5 {
		return true
	}
	return trimmed[5] == ' ' || trimmed[5] == '\t'
}

// parseLabelLine parses "key1=val1 key2=val2" and adds dis.* entries to labels.
func parseLabelLine(line string, labels map[string]string) {
	line = strings.TrimSpace(line)
	for len(line) > 0 {
		key, value, rest := parseOneKeyValue(line)
		if key == "" {
			break
		}
		if strings.HasPrefix(key, "dis.") {
			labels[key] = value
		}
		line = strings.TrimSpace(rest)
	}
}

// parseOneKeyValue extracts the first key=value pair from the line.
// Returns the key, unquoted value, and the remainder of the line.
func parseOneKeyValue(line string) (string, string, string) {
	// Find the '=' that separates key from value
	eqIdx := strings.Index(line, "=")
	if eqIdx == -1 {
		return "", "", ""
	}

	key := strings.TrimSpace(line[:eqIdx])
	rest := line[eqIdx+1:]

	if len(rest) == 0 {
		return key, "", ""
	}

	// Check if value is quoted
	if rest[0] == '"' {
		// Find closing double quote
		closeIdx := strings.Index(rest[1:], "\"")
		if closeIdx == -1 {
			// No closing quote — take the rest
			return key, rest[1:], ""
		}
		value := rest[1 : closeIdx+1]
		remainder := rest[closeIdx+2:]
		return key, value, remainder
	}
	if rest[0] == '\'' {
		// Find closing single quote
		closeIdx := strings.Index(rest[1:], "'")
		if closeIdx == -1 {
			return key, rest[1:], ""
		}
		value := rest[1 : closeIdx+1]
		remainder := rest[closeIdx+2:]
		return key, value, remainder
	}

	// Unquoted value — read until next whitespace
	spaceIdx := strings.IndexAny(rest, " \t")
	if spaceIdx == -1 {
		return key, rest, ""
	}
	return key, rest[:spaceIdx], rest[spaceIdx:]
}
