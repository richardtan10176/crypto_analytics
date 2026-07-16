package config

import "strings"

// Symbols parses a comma-separated SYMBOLS env var (e.g. "BTCUSDT, ethusdt")
// into a deduplicated, upper-cased, whitespace-trimmed slice.
func Symbols(raw string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
