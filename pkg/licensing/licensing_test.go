package licensing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicenseFileNamesContains(t *testing.T) {
	l := LicenseFileNames{}

	for _, stem := range []string{
		"license", "licence", "unlicense", "copyright", "copying",
		"license.txt", "licence.txt", "unlicense.txt", "copyright.txt", "copying.txt",
		"license.md", "licence.md", "unlicense.md", "copyright.md", "copying.md",
		"license.rst", "licence.rst", "unlicense.rst", "copyright.rst", "copying.rst",
		"LICENSE", "License.txt",
	} {
		assert.True(t, l.Contains(stem), "stem %q should be a license file name", stem)
	}

	for _, stem := range []string{"readme", "readme.md", "notice", "changelog", "foo", ""} {
		assert.False(t, l.Contains(stem), "stem %q should not be a license file name", stem)
	}
}

func TestSPDXLicenseIDByURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
		have bool
	}{
		{"http://www.apache.org/licenses/license-2.0", "Apache-2.0", true},
		{"https://www.apache.org/licenses/license-2.0", "Apache-2.0", true},
		{"http://www.apache.org/licenses/license-2.0.txt", "Apache-2.0", true},
		{"https://www.apache.org/licenses/license-2.0.txt", "Apache-2.0", true},
		{"http://opensource.org/licenses/mit-license", "MIT", true},
		{"https://opensource.org/licenses/mit-license", "MIT", true},
		{"http://opensource.org/licenses/mit", "MIT", true},
		{"https://opensource.org/licenses/mit", "MIT", true},
		{"http://www.gnu.org/licenses/gpl-2.0", "GPL-2.0", true},
		{"https://www.gnu.org/licenses/gpl-2.0", "GPL-2.0", true},
		{"http://www.gnu.org/licenses/gpl-3.0", "GPL-3.0", true},
		{"https://www.gnu.org/licenses/gpl-3.0", "GPL-3.0", true},
		{"http://www.gnu.org/licenses/lgpl-2.1", "LGPL-2.1", true},
		{"https://www.gnu.org/licenses/lgpl-2.1", "LGPL-2.1", true},
		{"http://www.gnu.org/licenses/lgpl-3.0", "LGPL-3.0", true},
		{"https://www.gnu.org/licenses/lgpl-3.0", "LGPL-3.0", true},
		{"https://opensource.org/licenses/bsd-2-clause", "BSD-2-Clause", true},
		{"http://opensource.org/licenses/bsd-2-clause", "BSD-2-Clause", true},
		{"https://opensource.org/licenses/bsd-3-clause", "BSD-3-Clause", true},
		{"http://opensource.org/licenses/bsd-3-clause", "BSD-3-Clause", true},
		{"https://opensource.org/licenses/isc", "ISC", true},
		{"http://opensource.org/licenses/isc", "ISC", true},
		{"https://spdx.org/licenses/unlicense.html", "Unlicense", true},
		{"http://spdx.org/licenses/unlicense.html", "Unlicense", true},
		{"https://opensource.org/licenses/mit-0", "MIT-0", true},
		{"http://opensource.org/licenses/mit-0", "MIT-0", true},
		{"http://www.eclipse.org/legal/epl-v10.html", "EPL-1.0", true},
		{"https://www.eclipse.org/legal/epl-v10.html", "EPL-1.0", true},
		{"https://example.org/licenses/unknown", "", false},
		{"", "", false},
		{"   ", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, ok := SPDXLicenseIDByURL(tt.url)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.have, ok)
		})
	}
}

func TestSPDXLicenseID(t *testing.T) {
	tests := []struct {
		name string
		want string
		have bool
	}{
		{"apache 2.0", "Apache-2.0", true},
		{"apache-2.0", "Apache-2.0", true},
		{"apache license 2.0", "Apache-2.0", true},
		{"mit", "MIT", true},
		{"gpl-2.0", "GPL-2.0", true},
		{"gnu general public license v2.0", "GPL-2.0", true},
		{"gnu general public license v2", "GPL-2.0", true},
		{"gpl-3.0", "GPL-3.0", true},
		{"gnu general public license v3.0", "GPL-3.0", true},
		{"gnu general public license v3", "GPL-3.0", true},
		{"lgpl-2.1", "LGPL-2.1", true},
		{"gnu lesser general public license v2.1", "LGPL-2.1", true},
		{"lgpl-3.0", "LGPL-3.0", true},
		{"gnu lesser general public license v3.0", "LGPL-3.0", true},
		{"bsd-2-clause", "BSD-2-Clause", true},
		{"bsd 2-clause", "BSD-2-Clause", true},
		{"bsd-3-clause", "BSD-3-Clause", true},
		{"bsd 3-clause", "BSD-3-Clause", true},
		{"isc", "ISC", true},
		{"unlicense", "Unlicense", true},
		{"eclipse public license, version 1.0", "EPL-1.0", true},
		{"mit-0", "MIT-0", true},
		{"unknown license", "", false},
		{"", "", false},
		{"   ", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SPDXLicenseID(tt.name)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.have, ok)
		})
	}
}

func TestSplitLicenses(t *testing.T) {
	assert.Nil(t, SplitLicenses(""))
	assert.Nil(t, SplitLicenses("   "))
	assert.Equal(t, []string{"MIT"}, SplitLicenses("MIT"))
	assert.Equal(t, []string{"Apache-2.0", "MIT"}, SplitLicenses("Apache-2.0, MIT"))
	assert.Equal(t, []string{"Apache-2.0", "MIT"}, SplitLicenses("Apache-2.0 or MIT"))
	assert.Equal(t, []string{"Apache-2.0", "MIT"}, SplitLicenses("Apache-2.0 and MIT"))
	assert.Equal(t, []string{"GPL-2.0 or later"}, SplitLicenses("GPL-2.0 or later"))
	assert.Equal(t, []string{"GPL-2.0 and later"}, SplitLicenses("GPL-2.0 and later"))
	assert.Equal(t, []string{"MIT", "Apache-2.0"}, SplitLicenses("MIT or Apache-2.0"))
	assert.Equal(t, []string{"MIT", "Apache-2.0"}, SplitLicenses("MIT and Apache-2.0"))
	assert.Equal(t, []string{"a", "b", "c"}, SplitLicenses("a, b, c"))
	assert.Equal(t, []string{"MIT OR Apache-2.0"}, SplitLicenses("MIT OR Apache-2.0"))
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
		have bool
	}{
		{"apache", "Apache License Version 2.0", []string{"Apache-2.0"}, true},
		{"mit", "The MIT License", []string{"MIT"}, true},
		{"mit license", "MIT License", []string{"MIT"}, true},
		{"gpl-3.0", "GNU General Public License version 3", []string{"GPL-3.0"}, true},
		{"gpl-2.0", "GNU General Public License version 2", []string{"GPL-2.0"}, true},
		{"lgpl-3.0", "GNU Lesser General Public License version 3", []string{"LGPL-3.0"}, true},
		{"lgpl-2.1", "GNU Lesser General Public License version 2.1", []string{"LGPL-2.1"}, true},
		{"bsd-2-clause", "BSD 2-Clause License", []string{"BSD-2-Clause"}, true},
		{"bsd-3-clause", "BSD 3-Clause License", []string{"BSD-3-Clause"}, true},
		{"isc", "ISC License", []string{"ISC"}, true},
		{"unlicense", "The Unlicense", []string{"Unlicense"}, true},
		{"parenthetical", "Apache (Software) License Version 2.0", []string{"Apache-2.0"}, true},
		{"unknown", "Some random license text", nil, false},
		{"empty", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finding, err := Classify(tt.name, strings.NewReader(tt.text), 0.5)
			assert.NoError(t, err)
			if !tt.have {
				assert.Nil(t, finding)
				return
			}
			assert.NotNil(t, finding)
			assert.Equal(t, tt.want, finding.Findings.Names())
		})
	}
}
