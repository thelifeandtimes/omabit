package runtimequalify

import (
	"regexp"
	"strings"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var logEscape = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b\[[0-?]*[ -/]*[@-~]`)
var logKey = regexp.MustCompile(`0w[0-9a-zA-Z.~-]+`)
var logCredential = regexp.MustCompile(`(?i)authorization|set-cookie|cookie:|boot\.key|BEGIN .*PRIVATE KEY`)

// Sanitize before bounding output so truncation cannot retain part of a known
// credential token. Logs are plain text; terminal escapes have no UI authority.
func sanitizeLogs(text string) ([]string, bool) {
	text = logEscape.ReplaceAllString(text, "")
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(text, "�"))
	text = logKey.ReplaceAllString(text, "[redacted key]")
	lines := []string{}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if logCredential.MatchString(line) {
			line = "[sensitive log line omitted]"
		}
		lines = append(lines, line)
	}
	truncated := false
	if len(lines) > protocol.MaxLogLines {
		lines = lines[len(lines)-protocol.MaxLogLines:]
		truncated = true
	}
	total := 0
	for i := len(lines) - 1; i >= 0; i-- {
		if len(lines[i]) > 4096 {
			lines[i] = "[oversized log line omitted]"
			truncated = true
		}
		total += len(lines[i]) + 1
		if total > protocol.MaxLogBytes {
			lines = lines[i+1:]
			truncated = true
			break
		}
	}
	return lines, truncated
}
