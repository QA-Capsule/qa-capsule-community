package core

import (
	"strings"
)

// ReportFormat identifies how an inbound CI payload should be parsed.
type ReportFormat string

const (
	ReportFormatJSON      ReportFormat = "json"
	ReportFormatJUnitXML  ReportFormat = "junit_xml"
)

// IngestPayload is the normalized input to the execution hub reporter.
type IngestPayload struct {
	Format    ReportFormat
	Framework string
	Flags     ExecutionFlags
	JSON      map[string]interface{}
	XML       []byte
}

// IngestPayloadResult is the unified output: matrix report + failure alerts for ingest.
type IngestPayloadResult struct {
	Report   UnifiedExecutionReport
	Failures []UnifiedAlert
}

// UnifiedReporter transforms framework-agnostic CI payloads into hub models.
type UnifiedReporter struct{}

// DefaultUnifiedReporter is the package-level reporter used by webhooks.
var DefaultUnifiedReporter = &UnifiedReporter{}

// Normalize parses JSON or JUnit XML into a UnifiedExecutionReport and ingestable failures.
func (r *UnifiedReporter) Normalize(in IngestPayload) IngestPayloadResult {
	if r == nil {
		r = DefaultUnifiedReporter
	}
	framework := strings.TrimSpace(in.Framework)
	flags := in.Flags
	if in.JSON != nil && executionFlagsUnset(flags) {
		flags = ExecutionFlagsFromPayload(in.JSON)
	}
	if flags.Type == ExecutionTypeUnknown {
		flags.Type = ExecutionTypeReal
	}

	switch in.Format {
	case ReportFormatJUnitXML:
		junit := ParseJUnitReport(in.XML, framework)
		report := junit.Report
		report.Flags = mergeExecutionFlags(flags, report.Flags)
		return IngestPayloadResult{Report: report, Failures: junit.Failures}
	default:
		if in.JSON == nil {
			return IngestPayloadResult{
				Report: UnifiedExecutionReport{
					SchemaVersion: executionReportSchemaVersion,
					Flags:         flags,
					Framework:     framework,
				},
			}
		}
		report := BuildReportFromPayload(in.JSON, flags, framework)
		return IngestPayloadResult{
			Report:   report,
			Failures: AlertsFromReport(report, ParseAlertsFromRaw(in.JSON)),
		}
	}
}

// DetectReportFormat chooses json vs junit_xml from path suffix and Content-Type.
func DetectReportFormat(path, contentType string) ReportFormat {
	if strings.HasSuffix(strings.ToLower(path), "/upload") {
		return ReportFormatJUnitXML
	}
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "xml") {
		return ReportFormatJUnitXML
	}
	return ReportFormatJSON
}

func executionFlagsUnset(f ExecutionFlags) bool {
	envUnset := f.Env == "" || f.Env == ExecutionEnvUnknown
	typeUnset := f.Type == "" || f.Type == ExecutionTypeUnknown
	return envUnset && typeUnset
}

// AlertsFromReport returns failure alerts, preferring explicit parser output then matrix rows.
func AlertsFromReport(report UnifiedExecutionReport, explicit []UnifiedAlert) []UnifiedAlert {
	if len(explicit) > 0 {
		return explicit
	}
	var out []UnifiedAlert
	for _, tc := range report.Tests {
		if tc.Status != "fail" && tc.Status != "flaky" {
			continue
		}
		out = append(out, UnifiedAlert{
			Name:            tc.Name,
			Error:           tc.ErrorMessage,
			ConsoleLogs:     tc.ConsoleLogs,
			ErrorLogs:       tc.ErrorLogs,
			Status:          "CRITICAL",
			ExecutionTimeMs: tc.DurationMs,
		})
	}
	return out
}
