package runtimequalify

import (
	"regexp"
	"strings"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

var logEscape = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b\[[0-?]*[ -/]*[@-~]`)
var logKey = regexp.MustCompile(`0w[0-9a-zA-Z.~-]+`)
var logCode = regexp.MustCompile(`[a-z]{6}(?:-[a-z]{6}){3}`)
var logComet = regexp.MustCompile(`~[a-z]{6}(?:-[a-z]{6}){3}--[a-z]{6}(?:-[a-z]{6}){3}`)
var logCredential = regexp.MustCompile(`(?i)password|authorization|set-cookie|cookie:|boot\.key|BEGIN .*PRIVATE KEY`)

// A public comet has eight words, unlike the ambiguous four-word login-code
// shape. Preserve validated whole comet names while sanitizing surrounding text.
func redactLogCodes(text string) string {
	var out strings.Builder
	start := 0
	for _, match := range logComet.FindAllStringIndex(text, -1) {
		end := match[1]
		if end < len(text) && (text[end] >= 'a' && text[end] <= 'z' || text[end] == '-') {
			continue
		}
		name := text[match[0]:end]
		if protocol.ShipClass(name) != "comet" {
			continue
		}
		out.WriteString(logCode.ReplaceAllString(text[start:match[0]], "[redacted code or identity]"))
		out.WriteString(name)
		start = end
	}
	out.WriteString(logCode.ReplaceAllString(text[start:], "[redacted code or identity]"))
	return out.String()
}

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
	text = redactLogCodes(text)
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
