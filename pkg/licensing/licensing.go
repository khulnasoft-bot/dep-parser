package licensing

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

const LicenseFilePrefix = "file://"

// LicenseFileNames is the set of license file name stems that qualify as license files.
type LicenseFileNames struct{}

// Contains reports whether stem is a known license file name stem.
func (LicenseFileNames) Contains(stem string) bool {
	switch strings.ToLower(stem) {
	case "license", "licence", "unlicense", "copyright", "copying",
		"license.txt", "licence.txt", "unlicense.txt", "copyright.txt", "copying.txt",
		"license.md", "licence.md", "unlicense.md", "copyright.md", "copying.md",
		"license.rst", "licence.rst", "unlicense.rst", "copyright.rst", "copying.rst":
		return true
	}
	return false
}

// SPDXLicenseIDByURL maps a license URL to its SPDX identifier.
// An unresolved URL returns ("", false).
// All case values are lowercase because the input is lowercased before comparison.
func SPDXLicenseIDByURL(url string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(url)) {
	case "http://www.apache.org/licenses/license-2.0", "https://www.apache.org/licenses/license-2.0",
		"http://www.apache.org/licenses/license-2.0.txt", "https://www.apache.org/licenses/license-2.0.txt":
		return "Apache-2.0", true
	case "http://opensource.org/licenses/mit-license", "https://opensource.org/licenses/mit-license",
		"http://opensource.org/licenses/mit", "https://opensource.org/licenses/mit":
		return "MIT", true
	case "http://www.gnu.org/licenses/gpl-2.0", "https://www.gnu.org/licenses/gpl-2.0":
		return "GPL-2.0", true
	case "http://www.gnu.org/licenses/gpl-3.0", "https://www.gnu.org/licenses/gpl-3.0":
		return "GPL-3.0", true
	case "http://www.gnu.org/licenses/lgpl-2.1", "https://www.gnu.org/licenses/lgpl-2.1":
		return "LGPL-2.1", true
	case "http://www.gnu.org/licenses/lgpl-3.0", "https://www.gnu.org/licenses/lgpl-3.0":
		return "LGPL-3.0", true
	case "https://opensource.org/licenses/bsd-2-clause", "http://opensource.org/licenses/bsd-2-clause":
		return "BSD-2-Clause", true
	case "https://opensource.org/licenses/bsd-3-clause", "http://opensource.org/licenses/bsd-3-clause":
		return "BSD-3-Clause", true
	case "https://opensource.org/licenses/isc", "http://opensource.org/licenses/isc":
		return "ISC", true
	case "https://spdx.org/licenses/unlicense.html", "http://spdx.org/licenses/unlicense.html":
		return "Unlicense", true
	case "https://opensource.org/licenses/mit-0", "http://opensource.org/licenses/mit-0":
		return "MIT-0", true
	case "http://www.eclipse.org/legal/epl-v10.html", "https://www.eclipse.org/legal/epl-v10.html":
		return "EPL-1.0", true
	}
	return "", false
}

// SPDXLicenseID maps a license name to its SPDX identifier.
func SPDXLicenseID(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "apache 2.0", "apache-2.0", "apache license 2.0":
		return "Apache-2.0", true
	case "mit":
		return "MIT", true
	case "gpl-2.0", "gnu general public license v2.0", "gnu general public license v2":
		return "GPL-2.0", true
	case "gpl-3.0", "gnu general public license v3.0", "gnu general public license v3":
		return "GPL-3.0", true
	case "lgpl-2.1", "gnu lesser general public license v2.1":
		return "LGPL-2.1", true
	case "lgpl-3.0", "gnu lesser general public license v3.0":
		return "LGPL-3.0", true
	case "bsd-2-clause", "bsd 2-clause":
		return "BSD-2-Clause", true
	case "bsd-3-clause", "bsd 3-clause":
		return "BSD-3-Clause", true
	case "isc":
		return "ISC", true
	case "unlicense":
		return "Unlicense", true
	case "eclipse public license, version 1.0":
		return "EPL-1.0", true
	case "mit-0":
		return "MIT-0", true
	}
	return "", false
}

// SplitLicenses splits a license string into individual license names.
// It handles separators like ",", "or", "and" as standalone words
// (not when followed by "later" which is part of a version specification).
func SplitLicenses(s string) []string {
	if s == "" {
		return nil
	}
	commaParts := strings.Split(s, ",")
	var result []string
	for _, part := range commaParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, " or ") && !strings.Contains(part, " or later") {
			for _, sp := range strings.Split(part, " or ") {
				if sp = strings.TrimSpace(sp); sp != "" {
					result = append(result, sp)
				}
			}
		} else if strings.Contains(part, " and ") && !strings.Contains(part, " and later") {
			for _, sp := range strings.Split(part, " and ") {
				if sp = strings.TrimSpace(sp); sp != "" {
					result = append(result, sp)
				}
			}
		} else {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// findings holds the results of license classification.
type findings struct {
	names []string
}

// Names returns the detected license names.
func (f findings) Names() []string {
	return f.names
}

// LicenseFinding is the result of classifying a license file.
type LicenseFinding struct {
	Findings findings
}

// Classify examines a file's name and content to detect its license.
func Classify(name string, file io.Reader, confidenceLevel float64) (*LicenseFinding, error) {
	scanner := bufio.NewScanner(file)
	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	text := strings.ToLower(content.String())

	// Remove parenthetical content (e.g. "Apache (Software) License" -> "Apache License")
	// so that parenthesized qualifiers do not prevent pattern matching.
	re := regexp.MustCompile(`\([^)]*\)`)
	text = re.ReplaceAllString(text, " ")
	// Collapse multiple spaces introduced by removal.
	text = strings.Join(strings.Fields(text), " ")

	// Detect common license headers
	switch {
	case strings.Contains(text, "apache license") && strings.Contains(text, "version 2.0"):
		return &LicenseFinding{Findings: findings{names: []string{"Apache-2.0"}}}, nil
	case strings.Contains(text, "mit license") || strings.Contains(text, "the mit license"):
		return &LicenseFinding{Findings: findings{names: []string{"MIT"}}}, nil
	case strings.Contains(text, "gnu general public license") && strings.Contains(text, "version 3"):
		return &LicenseFinding{Findings: findings{names: []string{"GPL-3.0"}}}, nil
	case strings.Contains(text, "gnu general public license") && strings.Contains(text, "version 2"):
		return &LicenseFinding{Findings: findings{names: []string{"GPL-2.0"}}}, nil
	case strings.Contains(text, "gnu lesser general public license") && strings.Contains(text, "version 3"):
		return &LicenseFinding{Findings: findings{names: []string{"LGPL-3.0"}}}, nil
	case strings.Contains(text, "gnu lesser general public license") && strings.Contains(text, "version 2"):
		return &LicenseFinding{Findings: findings{names: []string{"LGPL-2.1"}}}, nil
	case strings.Contains(text, "bsd 2-clause") || strings.Contains(text, "bsd-2-clause"):
		return &LicenseFinding{Findings: findings{names: []string{"BSD-2-Clause"}}}, nil
	case strings.Contains(text, "bsd 3-clause") || strings.Contains(text, "bsd-3-clause"):
		return &LicenseFinding{Findings: findings{names: []string{"BSD-3-Clause"}}}, nil
	case strings.Contains(text, "isc license"):
		return &LicenseFinding{Findings: findings{names: []string{"ISC"}}}, nil
	case strings.Contains(text, "the unlicense"):
		return &LicenseFinding{Findings: findings{names: []string{"Unlicense"}}}, nil
	}

	return nil, nil
}
