package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration parses a duration string (e.g. "1d", "3600", "1h30m").
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if _, err := strconv.Atoi(s); err == nil {
		seconds, _ := strconv.Atoi(s)
		return time.Duration(seconds) * time.Second, nil
	}
	return time.ParseDuration(s)
}

// FormatDuration formats d in a human-readable short form (s, m, h, d).
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		days := d.Hours() / 24
		return fmt.Sprintf("%.1fd", days)
	}
}

// GetTimezoneOffset returns the offset in seconds for the given timezone name.
func GetTimezoneOffset(tz string) (int, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 0, err
	}
	now := time.Now().In(loc)
	_, offset := now.Zone()
	return offset, nil
}

// FormatTime formats t according to the named format (rfc3339, rfc822, rfc1123, unix, unix_milli, unix_micro, unix_nano) or custom layout.
func FormatTime(t time.Time, format string) string {
	switch format {
	case "rfc3339":
		return t.Format(time.RFC3339)
	case "rfc822":
		return t.Format(time.RFC822)
	case "rfc1123":
		return t.Format(time.RFC1123)
	case "unix":
		return fmt.Sprintf("%d", t.Unix())
	case "unix_milli":
		return fmt.Sprintf("%d", t.UnixMilli())
	case "unix_micro":
		return fmt.Sprintf("%d", t.UnixMicro())
	case "unix_nano":
		return fmt.Sprintf("%d", t.UnixNano())
	default:
		return t.Format(format)
	}
}

// TimeAgo returns a human-readable "time ago" string for t.
func TimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / (7 * 24))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / (30 * 24))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(diff.Hours() / (365 * 24))
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// StartOfDay returns t with time set to 00:00:00.
func StartOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// EndOfDay returns t with time set to 23:59:59.999999999.
func EndOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek returns the start of the week (Monday 00:00:00).
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return StartOfDay(t.AddDate(0, 0, -weekday+1))
}

// EndOfWeek returns the end of the week (Sunday 23:59:59).
func EndOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return EndOfDay(t.AddDate(0, 0, 7-weekday))
}

// StartOfMonth returns the first day of t's month at 00:00:00.
func StartOfMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the last day of t's month at 23:59:59.
func EndOfMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month+1, 0, 23, 59, 59, 999999999, t.Location())
}
