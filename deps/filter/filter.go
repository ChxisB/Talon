// Package filter executes shell commands and compresses their output to
// minimize token consumption when passed to LLMs. Supports git, npm,
// cargo, pytest, docker, and many more.
//
// Reference: rtk (Rust Token Killer) - MIT License
package filter

import (
	"os/exec"
	"strings"
)

// Level defines how aggressively output is filtered.
type Level int

const (
	// LevelNone passes output through unchanged.
	LevelNone Level = iota
	// LevelMinimal removes obvious noise (comments, blank lines).
	LevelMinimal
	// LevelAggressive groups, truncates, and deduplicates.
	LevelAggressive
)

// Result holds the filtered output and metadata.
type Result struct {
	Stdout   string // Filtered stdout
	Stderr   string // Filtered stderr
	ExitCode int
	Filtered bool   // Whether filtering was applied
	Command  string // Original command
}

// Filter runs a command and applies output filtering based on the command type.
func Filter(args []string, level Level) (*Result, error) {
	if len(args) == 0 {
		return nil, &FilterError{Msg: "no command provided"}
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	cmd := exec.Command(cmdName, cmdArgs...)
	stdout, err := cmd.CombinedOutput()

	result := &Result{
		Stdout:   string(stdout),
		ExitCode: 0,
		Command:  strings.Join(args, " "),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	}

	if level == LevelNone {
		return result, nil
	}

	// Route to command-specific filter
	filtered := FilterOutput(cmdName, cmdArgs, result.Stdout, level)
	if filtered != result.Stdout {
		result.Stdout = filtered
		result.Filtered = true
	}

	return result, nil
}

// FilterOutput applies command-specific output filtering to the given text.
// It detects the command type from cmdName (e.g. "git", "npm", "cargo")
// and applies the appropriate filter strategy. Pass the raw output text
// and get back the filtered version.
func FilterOutput(cmd string, args []string, output string, level Level) string {
	switch cmd {
	case "git":
		return filterGit(args, output, level)
	case "npm", "pnpm", "yarn", "bun":
		return filterNpm(cmd, args, output, level)
	case "cargo":
		return filterCargo(args, output, level)
	case "pytest", "py.test", "python", "python3":
		return filterPytest(args, output, level)
	case "docker":
		return filterDocker(args, output, level)
	case "ls", "exa", "eza":
		return filterLs(args, output, level)
	case "tree":
		return filterTree(args, output, level)
	case "cat", "head", "tail":
		return filterCat(args, output, level)
	case "grep", "rg", "ag":
		return filterGrep(args, output, level)
	case "find":
		return filterFind(args, output, level)
	case "curl", "wget":
		return filterHttp(args, output, level)
	case "psql", "mysql", "sqlite3":
		return filterSql(args, output, level)
	default:
		return filterGeneric(output, level)
	}
}

// --- Git filters ---

func filterGit(args []string, output string, level Level) string {
	if len(args) == 0 {
		return output
	}
	switch args[0] {
	case "status":
		return filterGitStatus(output, level)
	case "log":
		return filterGitLog(args, output, level)
	case "diff":
		return filterGitDiff(output, level)
	default:
		return filterGeneric(output, level)
	}
}

func filterGitStatus(output string, level Level) string {
	lines := strings.Split(output, "\n")
	var filtered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip "Untracked files:" header and the hint line
		if trimmed == "Untracked files:" || strings.HasPrefix(trimmed, "(use \"git") {
			continue
		}
		// Keep status lines (modified, added, deleted, etc.)
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func filterGitLog(args []string, output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		if len(lines) > 20 {
			return strings.Join(lines[:20], "\n") + "\n... (truncated)"
		}
	}
	return output
}

func filterGitDiff(output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		var filtered []string
		for _, line := range lines {
			// Keep only diff headers and changed lines
			if strings.HasPrefix(line, "diff ") ||
				strings.HasPrefix(line, "--- ") ||
				strings.HasPrefix(line, "+++ ") ||
				strings.HasPrefix(line, "@@") ||
				strings.HasPrefix(line, "+") ||
				strings.HasPrefix(line, "-") {
				filtered = append(filtered, line)
			}
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// --- NPM filters ---

func filterNpm(cmd string, args []string, output string, level Level) string {
	if level >= LevelMinimal {
		lines := strings.Split(output, "\n")
		var filtered []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Skip npm warnings and info noise
			if strings.Contains(trimmed, "npm warn") || strings.Contains(trimmed, "npm notice") {
				continue
			}
			if trimmed != "" {
				filtered = append(filtered, line)
			}
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// --- Cargo filters ---

func filterCargo(args []string, output string, level Level) string {
	if level >= LevelMinimal {
		lines := strings.Split(output, "\n")
		var filtered []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Keep warnings, errors, and compilation status
			if strings.Contains(trimmed, "warning[") ||
				strings.Contains(trimmed, "error[") ||
				strings.Contains(trimmed, "error:") ||
				strings.Contains(trimmed, "Compiling") ||
				strings.Contains(trimmed, "Finished") ||
				strings.Contains(trimmed, "Running") ||
				strings.Contains(trimmed, "test result:") {
				filtered = append(filtered, line)
			}
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// --- Pytest filters ---

func filterPytest(args []string, output string, level Level) string {
	if level >= LevelMinimal {
		lines := strings.Split(output, "\n")
		var filtered []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			// Keep test results, failures, and summary
			if strings.Contains(trimmed, "PASSED") ||
				strings.Contains(trimmed, "FAILED") ||
				strings.Contains(trimmed, "ERROR") ||
				strings.Contains(trimmed, "=== ") ||
				strings.Contains(trimmed, "--- ") ||
				strings.HasPrefix(trimmed, "E ") {
				filtered = append(filtered, line)
			}
		}
		if len(filtered) == 0 {
			// If nothing matched, return the last few lines
			if len(lines) > 5 {
				return strings.Join(lines[len(lines)-5:], "\n")
			}
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// --- Docker filters ---

func filterDocker(args []string, output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		if len(lines) > 30 {
			return strings.Join(lines[:30], "\n") + "\n... (truncated)"
		}
	}
	return output
}

// --- Ls filters ---

func filterLs(args []string, output string, level Level) string {
	if level >= LevelMinimal {
		lines := strings.Split(output, "\n")
		noise := map[string]bool{
			"node_modules": true, ".git": true, "target": true,
			"__pycache__": true, ".next": true, "dist": true,
			".DS_Store": true, ".vscode": true, ".idea": true,
		}
		var filtered []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			parts := strings.Fields(trimmed)
			if len(parts) == 0 {
				continue
			}
			name := parts[len(parts)-1]
			if noise[name] {
				continue
			}
			filtered = append(filtered, line)
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// --- Tree filters ---

func filterTree(args []string, output string, level Level) string {
	if level >= LevelMinimal {
		lines := strings.Split(output, "\n")
		noise := []string{"node_modules", ".git", "target", "__pycache__", ".next", "dist"}
		var filtered []string
		for _, line := range lines {
			skip := false
			for _, n := range noise {
				if strings.Contains(line, n) {
					skip = true
					break
				}
			}
			if !skip {
				filtered = append(filtered, line)
			}
		}
		if len(filtered) > 50 {
			return strings.Join(filtered[:50], "\n") + "\n... (truncated)"
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// --- Cat/Head/Tail filters ---

func filterCat(args []string, output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		if len(lines) > 50 {
			return strings.Join(lines[:50], "\n") + "\n... (truncated)"
		}
	}
	return output
}

// --- Grep filters ---

func filterGrep(args []string, output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		if len(lines) > 30 {
			return strings.Join(lines[:30], "\n") + "\n... (truncated)"
		}
	}
	return output
}

// --- Find filters ---

func filterFind(args []string, output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		if len(lines) > 30 {
			return strings.Join(lines[:30], "\n") + "\n... (truncated)"
		}
	}
	return output
}

// --- HTTP filters ---

func filterHttp(args []string, output string, level Level) string {
	if level >= LevelMinimal {
		// Only keep headers and first few lines of body
		lines := strings.Split(output, "\n")
		if len(lines) > 20 {
			return strings.Join(lines[:20], "\n") + "\n... (truncated)"
		}
	}
	return output
}

// --- SQL filters ---

func filterSql(args []string, output string, level Level) string {
	// SQL output is usually already compact
	return output
}

// --- Generic filter ---

func filterGeneric(output string, level Level) string {
	if level >= LevelAggressive {
		lines := strings.Split(output, "\n")
		// Remove blank lines and comments
		var filtered []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
				continue
			}
			filtered = append(filtered, line)
		}
		if len(filtered) > 30 {
			return strings.Join(filtered[:30], "\n") + "\n... (truncated)"
		}
		return strings.Join(filtered, "\n")
	}
	return output
}

// FilterError represents a filter execution error.
type FilterError struct {
	Msg string
}

func (e *FilterError) Error() string {
	return e.Msg
}
