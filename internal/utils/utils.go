package utils

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

var (
	reUptime  = regexp.MustCompile(`(?:(\d+)w)?(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?`)
	reMs      = regexp.MustCompile(`(?:(\d+)s)?(?:(\d+)ms)?(?:(\d+)us)?`)
	reVersion = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)
)

func ParseMktUptime(s string) int {
	m := reUptime.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	atoi := func(i int) int {
		n, _ := strconv.Atoi(m[i])
		return n
	}
	return atoi(5) + atoi(4)*60 + atoi(3)*3600 + atoi(2)*86400 + atoi(1)*604800
}

func ParseTimedelta(s string, msSpan bool) float64 {
	if msSpan {
		m := reMs.FindStringSubmatch(s)
		if m == nil {
			return 0
		}
		secs, _ := strconv.ParseFloat(m[1], 64)
		ms, _ := strconv.ParseFloat(m[2], 64)
		us, _ := strconv.ParseFloat(m[3], 64)
		return secs + ms/1000 + us/1000000
	}
	return float64(ParseMktUptime(s))
}

func Str2Bool(s string, defaultVal bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "t", "true", "on", "ok", "1":
		return true
	case "n", "no", "f", "false", "off", "fail", "0":
		return false
	default:
		return defaultVal
	}
}

// FormatInterfaceName formats an interface display name per the interface_name_format config.
// mode: "name" (default), "comment", or "combined".
func FormatInterfaceName(name, comment, mode string) string {
	if comment != "" && len(comment) > 20 {
		comment = comment[:20]
	}
	switch mode {
	case "comment":
		if comment != "" {
			return comment
		}
		return name
	case "combined":
		if comment != "" {
			return fmt.Sprintf("%s (%s)", name, comment)
		}
		return name
	default:
		if mode != "name" {
			slog.Warn("unknown interface_name_format, using 'name'", "format", mode)
		}
		return name
	}
}

func BuiltinWiFiCAPsMANVersion(version string) bool {
	major, minor := parseROSVersion(version)
	return major > 7 || (major == 7 && minor >= 13)
}

func RouterOS7Version(version string) bool {
	major, _ := parseROSVersion(version)
	return major >= 7
}

func parseROSVersion(version string) (major, minor int) {
	m := reVersion.FindStringSubmatch(version)
	if m == nil {
		return 0, 0
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	return major, minor
}
