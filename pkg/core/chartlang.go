package core

import (
	"fmt"
	"strconv"
	"strings"
)

// ChartQuery is the parsed result of a QCL (QA Chart Language) definition.
type ChartQuery struct {
	ChartType string `json:"chart_type"` // line, bar, doughnut
	Metric    string `json:"metric"`     // incidents, flaky, mttr, finops_cost, finops_flaky_cost, ci_minutes
	RangeDays int    `json:"range_days"`
	GroupBy   string `json:"group_by"` // week, project
	Project   string `json:"project"`  // optional filter
	Title     string `json:"title"`
}

// ParseChartQuery parses QCL text into a ChartQuery.
//
// Example:
//
//	CHART line "Weekly incident volume"
//	METRIC incidents
//	RANGE 35d
//	GROUP week
//	PROJECT payment-api
func ParseChartQuery(input string) (ChartQuery, error) {
	q := ChartQuery{
		ChartType: "line",
		Metric:    "incidents",
		RangeDays: 35,
		GroupBy:   "week",
	}

	for _, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := splitQCLLine(line)
		if len(parts) == 0 {
			continue
		}

		key := strings.ToUpper(parts[0])
		switch key {
		case "CHART":
			if len(parts) >= 2 {
				q.ChartType = strings.ToLower(parts[1])
			}
			if len(parts) >= 3 {
				q.Title = strings.Trim(parts[2], `"`)
			}
		case "METRIC":
			if len(parts) >= 2 {
				q.Metric = strings.ToLower(parts[1])
			}
		case "RANGE":
			if len(parts) >= 2 {
				days, err := parseRangeDays(parts[1])
				if err != nil {
					return q, err
				}
				q.RangeDays = days
			}
		case "GROUP":
			if len(parts) >= 2 {
				q.GroupBy = strings.ToLower(parts[1])
			}
		case "PROJECT":
			if len(parts) >= 2 {
				q.Project = parts[1]
			}
		default:
			return q, fmt.Errorf("unknown QCL directive: %s", parts[0])
		}
	}

	if err := validateChartQuery(q); err != nil {
		return q, err
	}
	return q, nil
}

func splitQCLLine(line string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case (r == ' ' || r == '\t') && !inQuote:
			if cur.Len() > 0 {
				parts = append(parts, strings.Trim(cur.String(), `"`))
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, strings.Trim(cur.String(), `"`))
	}
	return parts
}

func parseRangeDays(token string) (int, error) {
	token = strings.ToLower(strings.TrimSpace(token))
	if strings.HasSuffix(token, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(token, "d"))
		if err != nil || n < 1 {
			return 0, fmt.Errorf("invalid RANGE: %s", token)
		}
		return n, nil
	}
	if strings.HasSuffix(token, "w") {
		n, err := strconv.Atoi(strings.TrimSuffix(token, "w"))
		if err != nil || n < 1 {
			return 0, fmt.Errorf("invalid RANGE: %s", token)
		}
		return n * 7, nil
	}
	if strings.HasSuffix(token, "y") {
		n, err := strconv.Atoi(strings.TrimSuffix(token, "y"))
		if err != nil || n < 1 {
			return 0, fmt.Errorf("invalid RANGE: %s", token)
		}
		return n * 365, nil
	}
	return 0, fmt.Errorf("RANGE must end with d, w, or y (e.g. 35d, 12w, 1y)")
}

func validateChartQuery(q ChartQuery) error {
	switch q.ChartType {
	case "line", "bar", "doughnut":
	default:
		return fmt.Errorf("CHART must be line, bar, or doughnut")
	}
	switch q.Metric {
	case "incidents", "flaky", "mttr", "finops_cost", "finops_flaky_cost", "ci_minutes", "resolved":
	default:
		return fmt.Errorf("unsupported METRIC: %s", q.Metric)
	}
	switch q.GroupBy {
	case "week", "project":
	default:
		return fmt.Errorf("GROUP must be week or project")
	}
	if q.RangeDays < 1 || q.RangeDays > 730 {
		return fmt.Errorf("RANGE out of bounds (1-730 days)")
	}
	return nil
}
