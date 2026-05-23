package redact

import (
	"os"
	"regexp"
	"strings"
)

var usersPath = regexp.MustCompile(`(?i)([A-Za-z]:\\Users\\)[^\\]+`)

// String replaces common machine-specific path prefixes with placeholders.
// Safe to call on empty strings and on non-Windows builds (env vars may be empty).
func String(s string) string {
	if s == "" {
		return s
	}
	for _, key := range []string{
		"USERPROFILE", "LOCALAPPDATA", "APPDATA", "ProgramData",
		"HOMEDRIVE", "HOMEPATH", "TEMP", "TMP",
	} {
		if v := os.Getenv(key); v != "" && len(v) > 2 {
			s = strings.ReplaceAll(s, v, "<"+strings.ToLower(key)+">")
		}
	}
	s = usersPath.ReplaceAllString(s, `${1}<user>`)
	if u := os.Getenv("USERNAME"); u != "" && len(u) > 1 {
		s = strings.ReplaceAll(s, u, "<user>")
	}
	return s
}
