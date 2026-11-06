package secret

import (
	"bufio"
	"io"
	"regexp"
)

type Scanner struct{}

func NewScanner(rules interface{}) *Scanner {
	return &Scanner{}
}

type ScanArgs struct {
	FilePath string
	Content  io.Reader
	Binary   bool
}

type ScanResult struct {
	Findings []Finding
}

type Finding struct {
	Code CodeLocation
}

type CodeLocation struct {
	Lines []Line
}

type Line struct {
	IsCause bool
	Number  int
	Content string
}

var (
	privateKeyBeginPattern = regexp.MustCompile(`(?m)^-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	privateKeyEndPattern   = regexp.MustCompile(`(?m)^-----END [A-Z ]*PRIVATE KEY-----`)
)

func (s *Scanner) Scan(args ScanArgs) ScanResult {
	result := ScanResult{}
	if args.Content == nil {
		return result
	}

	scanner := bufio.NewScanner(args.Content)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	inKeyBlock := false
	for i, line := range lines {
		lineNum := i + 1
		if privateKeyBeginPattern.MatchString(line) {
			inKeyBlock = true
			continue
		}
		if inKeyBlock {
			if privateKeyEndPattern.MatchString(line) {
				inKeyBlock = false
				continue
			}
			result.Findings = append(result.Findings, Finding{
				Code: CodeLocation{
					Lines: []Line{{IsCause: true, Number: lineNum, Content: line}},
				},
			})
		}
	}

	return result
}
