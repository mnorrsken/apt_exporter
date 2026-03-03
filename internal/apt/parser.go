package apt

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// PackageUpgrade represents a single pending package upgrade.
type PackageUpgrade struct {
	Package     string
	FromVersion string
	ToVersion   string
	Origin      string
	Arch        string
}

// ParseResult holds all parsed upgrade information.
type ParseResult struct {
	Packages []PackageUpgrade
}

// PendingByOriginArch returns aggregated counts keyed by (origin, arch).
func (r *ParseResult) PendingByOriginArch() map[[2]string]int {
	counts := make(map[[2]string]int)
	for _, pkg := range r.Packages {
		key := [2]string{pkg.Origin, pkg.Arch}
		counts[key]++
	}
	return counts
}

// TotalPending returns total number of pending upgrades.
func (r *ParseResult) TotalPending() int {
	return len(r.Packages)
}

// Parse parses the output of `apt-get --just-print dist-upgrade`.
// It extracts lines starting with "Inst" and parses two formats:
//
//	Inst <pkg> [<cur_ver>] (<new_ver> <origin> [<arch>])
//	Inst <pkg> (<new_ver> <origin> [<arch>])
func Parse(output string) (*ParseResult, error) {
	result := &ParseResult{}
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Inst ") {
			continue
		}

		pkg := parseLine(line)
		if pkg != nil {
			result.Packages = append(result.Packages, *pkg)
		}
	}

	return result, scanner.Err()
}

// parseLine parses a single "Inst" line from apt-get output.
func parseLine(line string) *PackageUpgrade {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil
	}

	pkg := &PackageUpgrade{
		Package: fields[1],
	}

	// Determine if current version is present (field starts with "[").
	start := 2
	if strings.HasPrefix(fields[2], "[") {
		pkg.FromVersion = stripBrackets(fields[2])
		start = 3
	}

	// Parse remaining fields: new_ver, origin, arch
	// Strip all parentheses and brackets.
	var remaining []string
	for i := start; i < len(fields); i++ {
		cleaned := stripBrackets(fields[i])
		if cleaned != "" {
			remaining = append(remaining, cleaned)
		}
	}

	if len(remaining) >= 1 {
		pkg.ToVersion = remaining[0]
	}
	if len(remaining) >= 2 {
		pkg.Origin = remaining[1]
	}
	if len(remaining) >= 3 {
		pkg.Arch = remaining[2]
	}

	return pkg
}

// stripBrackets removes (, ), [, ] characters from a string.
func stripBrackets(s string) string {
	r := strings.NewReplacer("(", "", ")", "", "[", "", "]", "")
	return r.Replace(s)
}

// CheckReboot returns true if the reboot-required file exists.
func CheckReboot(rootfs string) bool {
	path := filepath.Join(rootfs, "run", "reboot-required")
	_, err := os.Stat(path)
	return err == nil
}
