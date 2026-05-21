package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type incidentTimeWindow struct {
	From string
	To   string
}

func incidentTimeWindowFromRequest(r *http.Request) incidentTimeWindow {
	now := time.Now().UTC()
	from := now.Add(-5 * time.Minute)
	to := now

	preset := strings.TrimSpace(r.URL.Query().Get("range"))
	if preset == "" {
		preset = "5m"
	}

	if preset == "custom" {
		fromSet := false
		toSet := false
		if f := parseTimeParam(r.URL.Query().Get("from")); !f.IsZero() {
			from = f
			fromSet = true
		}
		if t := parseTimeParam(r.URL.Query().Get("to")); !t.IsZero() {
			to = t
			toSet = true
		}
		if !fromSet || !toSet {
			// Incomplete custom range — fall back to last 24h instead of silent 5m
			from = now.Add(-24 * time.Hour)
			to = now
		}
		if to.Before(from) {
			from, to = to, from
		}
		// Include full end minute when "to" has no seconds
		if to.Second() == 0 {
			to = to.Add(59*time.Second + 999*time.Millisecond)
		}
		return formatIncidentWindow(from, to)
	}

	switch preset {
	case "all":
		from = now.Add(-365 * 24 * time.Hour)
	case "today":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case "yesterday":
		y := now.Add(-24 * time.Hour)
		from = time.Date(y.Year(), y.Month(), y.Day(), 0, 0, 0, 0, time.UTC)
		to = time.Date(y.Year(), y.Month(), y.Day(), 23, 59, 59, 0, time.UTC)
	default:
		if d, ok := parseDurationPreset(preset); ok {
			from = now.Add(-d)
		}
	}

	return formatIncidentWindow(from, to)
}

func parseDurationPreset(preset string) (time.Duration, bool) {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if len(preset) < 2 {
		return 0, false
	}
	unit := preset[len(preset)-1]
	numStr := preset[:len(preset)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n < 1 {
		return 0, false
	}
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, true
	case 'h':
		return time.Duration(n) * time.Hour, true
	case 'd':
		return time.Duration(n) * 24 * time.Hour, true
	default:
		return 0, false
	}
}

func parseTimeParam(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func formatIncidentWindow(from, to time.Time) incidentTimeWindow {
	return incidentTimeWindow{
		From: from.UTC().Format("2006-01-02 15:04:05"),
		To:   to.UTC().Format("2006-01-02 15:04:05"),
	}
}

// evolutionBucketExpr picks hour/day/week grouping based on window size.
func evolutionBucketExpr(from, to time.Time) string {
	d := to.Sub(from)
	if d <= 36*time.Hour {
		return `strftime('%Y-%m-%d %H:00', created_at)`
	}
	if d <= 31*24*time.Hour {
		return `date(created_at)`
	}
	return `date(created_at, 'weekday 0', '-6 days')`
}

func evolutionBucketLabel(from, to time.Time) string {
	d := to.Sub(from)
	if d <= 36*time.Hour {
		return "hour"
	}
	if d <= 31*24*time.Hour {
		return "day"
	}
	return "week"
}
