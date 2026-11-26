package datareader

import "regexp"

// FormatClass describes pyreadstat's interpretation of a SAS format.
type FormatClass string

const (
	FormatNone     FormatClass = ""
	FormatDate     FormatClass = "date"
	FormatDateTime FormatClass = "datetime"
	FormatTime     FormatClass = "time"
)

// Matches pyreadstat's format_regex, including its case sensitivity.
var sasFormatPattern = regexp.MustCompile(`^([A-Z][A-Z0-9]*[A-Z])(?:[0-9]+(?:\.[0-9]*)?|\.?)$`)
var sasTimeFormats = formatSet("TIME HHMM TOD TIMEAMPM IS8601TM E8601TM B8601TM")

// SASFormatClass classifies a SAS format, optionally including width and precision.
func SASFormatClass(format string) FormatClass {
	m := sasFormatPattern.FindStringSubmatch(format)
	if m == nil {
		return FormatNone
	}
	switch {
	case sasDateFormats[m[1]]:
		return FormatDate
	case sasDatetimeFormats[m[1]]:
		return FormatDateTime
	case sasTimeFormats[m[1]]:
		return FormatTime
	default:
		return FormatNone
	}
}
