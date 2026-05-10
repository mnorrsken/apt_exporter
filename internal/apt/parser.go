package apt

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
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

// instLineRE matches an apt-get "Inst" line and captures pkg, optional
// current version, new version, origin list, and arch. Anything after the
// closing ")" (e.g. a "[trigger:arch]" field on auto-installed packages)
// is intentionally not consumed so the trigger cannot be misread as arch.
//
//	Inst <pkg> [<cur_ver>] (<new_ver> <origin>[, <origin2>...] [<arch>])
//	Inst <pkg> (<new_ver> <origin>[, <origin2>...] [<arch>])
var instLineRE = regexp.MustCompile(`^Inst (\S+)(?: \[([^\]]+)\])? \((\S+) (.+) \[([^\]]+)\]\)`)

// Parse parses the output of `apt-get --just-print dist-upgrade`.
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
	m := instLineRE.FindStringSubmatch(line)
	if m == nil {
		return nil
	}

	// Multiple origins are comma-separated; keep the first.
	origin := strings.TrimSpace(strings.SplitN(m[4], ",", 2)[0])

	return &PackageUpgrade{
		Package:     m[1],
		FromVersion: m[2],
		ToVersion:   m[3],
		Origin:      origin,
		Arch:        m[5],
	}
}

// CheckReboot returns true if the reboot-required file exists.
func CheckReboot(rootfs string) bool {
	path := filepath.Join(rootfs, "run", "reboot-required")
	_, err := os.Stat(path)
	return err == nil
}
