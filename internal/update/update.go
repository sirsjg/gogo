package update

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

const brewFormula = "sirsjg/gogo/gogo"

// ANSI escape codes for terminal formatting
const (
	bold  = "\033[1m"
	reset = "\033[0m"
)

// brewInfo represents the relevant parts of brew info --json=v2 output
type brewInfo struct {
	Formulae []struct {
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	} `json:"formulae"`
}

// Check displays the current version and available homebrew version
func Check(w io.Writer, currentVersion string) error {
	brewVersion, err := getBrewVersion()
	if err != nil {
		return fmt.Errorf("failed to check homebrew version: %w", err)
	}

	current := normalizeVersion(currentVersion)
	available := normalizeVersion(brewVersion)

	isNewer := compareVersions(available, current) > 0

	if isNewer {
		fmt.Fprintf(w, "gogo %s → %s%s%s\n", current, bold, available, reset)
		fmt.Fprintf(w, "\nUpgrade with: brew upgrade %s\n", brewFormula)
	} else {
		fmt.Fprintf(w, "gogo %s → %s (up to date)\n", current, available)
	}

	return nil
}

// getBrewVersion fetches the latest version from homebrew
func getBrewVersion() (string, error) {
	// Check if brew is available
	if _, err := exec.LookPath("brew"); err != nil {
		return "", fmt.Errorf("homebrew not installed")
	}

	cmd := exec.Command("brew", "info", "--json=v2", brewFormula)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "No available formula") {
				return "", fmt.Errorf("formula not found - tap with: brew tap sirsjg/gogo")
			}
			return "", fmt.Errorf("brew command failed: %s", stderr)
		}
		return "", err
	}

	var info brewInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("failed to parse brew info: %w", err)
	}

	if len(info.Formulae) == 0 {
		return "", fmt.Errorf("formula not found in homebrew")
	}

	return info.Formulae[0].Versions.Stable, nil
}

// normalizeVersion strips 'v' prefix if present
func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// compareVersions compares two semver strings
// Returns: >0 if a > b, <0 if a < b, 0 if equal
func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}

		if numA > numB {
			return 1
		}
		if numA < numB {
			return -1
		}
	}

	return 0
}
