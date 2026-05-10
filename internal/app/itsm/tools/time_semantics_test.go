package tools

import (
	"testing"
	"time"

	"metis/internal/app/itsm/form"
)

func TestTimeSemanticHelpers_FieldFlagsAndTextParsing(t *testing.T) {
	t.Run("date range validation flag honors withTime mode and raw datetimes", func(t *testing.T) {
		fieldWithTime := FormField{
			Key:   "access_window",
			Label: "访问时间窗",
			Type:  form.FieldDateRange,
			Props: map[string]any{"withTime": true},
		}
		if !needsDateTimeRangeValidation(fieldWithTime, map[string]any{"start": "2026-05-11 10:00:00", "end": "2026-05-11 11:00:00"}, "") {
			t.Fatal("expected withTime date range to require datetime validation")
		}

		fieldDatetimeMode := FormField{
			Key:   "maintenance_window",
			Label: "维护时间段",
			Type:  form.FieldDateRange,
			Props: map[string]any{"mode": "datetime"},
		}
		if !needsDateTimeRangeValidation(fieldDatetimeMode, map[string]any{"start": "2026-05-11 10:00:00", "end": "2026-05-11 11:00:00"}, "") {
			t.Fatal("expected datetime mode date range to require datetime validation")
		}

		fieldByRawValue := FormField{
			Key:   "visit_period",
			Label: "访问时段",
			Type:  form.FieldDateRange,
		}
		if !needsDateTimeRangeValidation(fieldByRawValue, map[string]any{"start": "2026-05-11 10:00:00", "end": "2026-05-11 11:00:00"}, "") {
			t.Fatal("expected raw datetime component to trigger validation")
		}

		textField := FormField{
			Key:   "access_window_text",
			Label: "访问时间窗口",
			Type:  form.FieldText,
		}
		if !needsDateTimeRangeValidation(textField, "明天下午 3 点到 5 点", "请帮我申请明天下午 3 点到 5 点访问") {
			t.Fatal("expected semantic text field with range to require validation")
		}
	})

	t.Run("parse textual time ranges and reject malformed input", func(t *testing.T) {
		start, end, ok := parseTimeRangeString("2026-05-11 10:00 到 2026-05-11 11:30")
		if !ok || start != "2026-05-11 10:00" || end != "2026-05-11 11:30" {
			t.Fatalf("unexpected parsed range: ok=%v start=%q end=%q", ok, start, end)
		}

		start, end, ok = parseTimeRangeString("明天下午3点~5点")
		if !ok || start != "明天下午3点" || end != "5点" {
			t.Fatalf("unexpected relative parsed range: ok=%v start=%q end=%q", ok, start, end)
		}

		if _, _, ok := parseTimeRangeString("仅有一个时间点 10:00"); ok {
			t.Fatal("expected malformed textual range to be rejected")
		}
	})
}

func TestTimeSemanticHelpers_ResolveRelativeRangesAndMeridiem(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	base := time.Date(2026, 5, 10, 16, 0, 0, 0, loc)

	t.Run("whole range meridiem applies to trailing clock", func(t *testing.T) {
		result := resolveRelativeDateTimeRange("明天下午3点到5点", base)
		if result.NeedsClarification || result.Resolved == nil {
			t.Fatalf("expected resolvable relative range, got %+v", result)
		}
		if got := result.Resolved.Start.Format("2006-01-02 15:04"); got != "2026-05-11 15:00" {
			t.Fatalf("unexpected start time: %s", got)
		}
		if got := result.Resolved.End.Format("2006-01-02 15:04"); got != "2026-05-11 17:00" {
			t.Fatalf("unexpected end time: %s", got)
		}
	})

	t.Run("same day elapsed range rolls forward when no explicit day anchor", func(t *testing.T) {
		result := resolveRelativeDateTimeRange("10点到11点", base)
		if result.NeedsClarification || result.Resolved == nil {
			t.Fatalf("expected fallback resolvable range, got %+v", result)
		}
		if got := result.Resolved.Start.Format("2006-01-02 15:04"); got != "2026-05-11 10:00" {
			t.Fatalf("unexpected rolled start time: %s", got)
		}
		if got := result.Resolved.End.Format("2006-01-02 15:04"); got != "2026-05-11 11:00" {
			t.Fatalf("unexpected rolled end time: %s", got)
		}
	})

	t.Run("single relative time without end falls back to one hour slot", func(t *testing.T) {
		result := resolveRelativeDateTimeRange("明天下午3点处理", base)
		if result.NeedsClarification || result.Resolved == nil {
			t.Fatalf("expected single time to resolve to one hour slot, got %+v", result)
		}
		if got := result.Resolved.Start.Format("2006-01-02 15:04"); got != "2026-05-11 15:00" {
			t.Fatalf("unexpected single-slot start: %s", got)
		}
		if got := result.Resolved.End.Format("2006-01-02 15:04"); got != "2026-05-11 16:00" {
			t.Fatalf("unexpected single-slot end: %s", got)
		}
	})

	t.Run("relative wording without clock requires clarification", func(t *testing.T) {
		result := resolveRelativeDateTimeRange("明天下午安排一下", base)
		if !result.NeedsClarification {
			t.Fatalf("expected clarification for vague relative range, got %+v", result)
		}
	})

	t.Run("invalid clock is rejected", func(t *testing.T) {
		if _, ok := parseClockWithContext("下午25点"); ok {
			t.Fatal("expected invalid clock to be rejected")
		}
	})
}

func TestTimeSemanticHelpers_NormalizationAndValidation(t *testing.T) {
	field := FormField{
		Key:         "access_window",
		Label:       "访问时间窗",
		Type:        form.FieldDateRange,
		Description: "需要完整的访问时间段",
		Props:       map[string]any{"withTime": true},
	}

	t.Run("normalize relative date range into explicit timestamps", func(t *testing.T) {
		now := chinaNow()
		expectedStart := midnight(now).AddDate(0, 0, 1).Add(15 * time.Hour)
		expectedEnd := midnight(now).AddDate(0, 0, 1).Add(17 * time.Hour)
		normalized, changed := normalizeDateTimeSemanticFieldValue(field, map[string]any{}, "请申请明天下午3点到5点访问")
		if !changed {
			t.Fatal("expected relative range to be normalized")
		}
		value, ok := normalized.(map[string]any)
		if !ok {
			t.Fatalf("expected normalized map value, got %#v", normalized)
		}
		if value["start"] != expectedStart.Format("2006-01-02 15:04:05") || value["end"] != expectedEnd.Format("2006-01-02 15:04:05") {
			t.Fatalf("unexpected normalized range: %#v", value)
		}
	})

	t.Run("validate detects mismatch with original request range", func(t *testing.T) {
		warning := validateDateTimeRangeField(field, map[string]any{
			"start": "2026-05-11 14:00:00",
			"end":   "2026-05-11 17:00:00",
		}, "请申请明天下午3点到5点访问")
		if warning == nil || warning.Type != "invalid_datetime_range" {
			t.Fatalf("expected mismatch warning, got %+v", warning)
		}
	})

	t.Run("validate rejects past and inverted ranges", func(t *testing.T) {
		pastWarning := validateDateTimeRangeField(field, map[string]any{
			"start": "2026-05-10 08:00:00",
			"end":   "2026-05-10 10:00:00",
		}, "")
		if pastWarning == nil || pastWarning.Type != "past_datetime_range" {
			t.Fatalf("expected past range warning, got %+v", pastWarning)
		}

		invertedWarning := validateDateTimeRangeField(field, map[string]any{
			"start": "2026-05-11 11:00:00",
			"end":   "2026-05-11 10:00:00",
		}, "")
		if invertedWarning == nil || invertedWarning.Type != "invalid_datetime_range" {
			t.Fatalf("expected inverted range warning, got %+v", invertedWarning)
		}
	})
}
