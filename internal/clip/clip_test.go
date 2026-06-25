package clip

import (
	"testing"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"18", 18},
		{"45", 45},
		{"0:18", 18},
		{"3:40", 220},
		{"14:50", 890},
		{"1:23:45", 5025},
		{"0:00", 0},
		{"0:00:30", 30},
		{"1:00:00", 3600},
	}
	for _, tc := range tests {
		got, err := ParseTimestamp(tc.input)
		if err != nil {
			t.Fatalf("ParseTimestamp(%q) error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("ParseTimestamp(%q) = %.0f, want %.0f", tc.input, got, tc.want)
		}
	}
}

func TestParseTimestampErrors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{""},
		{"abc"},
		{"1:2:3:4"},
		{"1:abc"},
		{"  "},
		{"12:30:45:99"},
	}
	for _, tc := range tests {
		_, err := ParseTimestamp(tc.input)
		if err == nil {
			t.Fatalf("ParseTimestamp(%q) expected error, got nil", tc.input)
		}
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		input     string
		wantStart float64
		wantEnd   float64
	}{
		{"0:18-3:40", 18, 220},
		{"3:40-4:05", 220, 245},
		{"14:50-16:14", 890, 974},
		{"0-2", 0, 2},
		{"0:01-0:02", 1, 2},
		{"1:23:45-1:24:00", 5025, 5040},
	}
	for _, tc := range tests {
		start, end, err := ParseRange(tc.input)
		if err != nil {
			t.Fatalf("ParseRange(%q) error: %v", tc.input, err)
		}
		if start != tc.wantStart {
			t.Fatalf("ParseRange(%q) start = %.0f, want %.0f", tc.input, start, tc.wantStart)
		}
		if end != tc.wantEnd {
			t.Fatalf("ParseRange(%q) end = %.0f, want %.0f", tc.input, end, tc.wantEnd)
		}
	}
}

func TestParseRangeErrors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{""},
		{"0:18"},
		{"0:18-"},
		{"-3:40"},
		{"3:40-3:40"}, // start == end
		{"4:05-3:40"}, // start > end
		{"abc-def"},
	}
	for _, tc := range tests {
		_, _, err := ParseRange(tc.input)
		if err == nil {
			t.Fatalf("ParseRange(%q) expected error, got nil", tc.input)
		}
	}
}

func TestParseLabelRange(t *testing.T) {
	tests := []struct {
		input     string
		wantLabel string
		wantStart float64
		wantEnd   float64
	}{
		{"issue1=0:18-3:40", "issue1", 18, 220},
		{"issue1-blank-row=0:18-3:40", "issue1-blank-row", 18, 220},
		{"blank_cells=3:40-4:05", "blank_cells", 220, 245},
	}
	for _, tc := range tests {
		label, start, end, err := ParseLabelRange(tc.input)
		if err != nil {
			t.Fatalf("ParseLabelRange(%q) error: %v", tc.input, err)
		}
		if label != tc.wantLabel {
			t.Fatalf("ParseLabelRange(%q) label = %q, want %q", tc.input, label, tc.wantLabel)
		}
		if start != tc.wantStart {
			t.Fatalf("ParseLabelRange(%q) start = %.0f, want %.0f", tc.input, start, tc.wantStart)
		}
		if end != tc.wantEnd {
			t.Fatalf("ParseLabelRange(%q) end = %.0f, want %.0f", tc.input, end, tc.wantEnd)
		}
	}
}

func TestParseLabelRangeErrors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{""},
		{"0:18-3:40"},        // no "=" sign
		{"=0:18-3:40"},       // empty label
		{"issue1=0:18"},      // incomplete range
		{"issue1=3:40-3:40"}, // start == end
	}
	for _, tc := range tests {
		_, _, _, err := ParseLabelRange(tc.input)
		if err == nil {
			t.Fatalf("ParseLabelRange(%q) expected error, got nil", tc.input)
		}
	}
}

func TestValidateSpec(t *testing.T) {
	// Valid spec should pass.
	if err := ValidateSpec(ClipSpec{Label: "test", StartSec: 0, EndSec: 10}); err != nil {
		t.Fatalf("valid spec returned error: %v", err)
	}

	// Negative start.
	if err := ValidateSpec(ClipSpec{Label: "test", StartSec: -1, EndSec: 10}); err == nil {
		t.Fatalf("expected error for negative start, got nil")
	}

	// End == start.
	if err := ValidateSpec(ClipSpec{Label: "test", StartSec: 5, EndSec: 5}); err == nil {
		t.Fatalf("expected error for end == start, got nil")
	}

	// End < start.
	if err := ValidateSpec(ClipSpec{Label: "test", StartSec: 10, EndSec: 5}); err == nil {
		t.Fatalf("expected error for end < start, got nil")
	}
}

func TestSafeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"issue1", "issue1"},
		{"issue1-blank-row", "issue1-blank-row"},
		{"issue 1", "issue_1"},
		{"issue/1", "issue_1"},
		{"  spaces  ", "spaces"},
		{"", ""},
	}
	for _, tc := range tests {
		got := SafeLabel(tc.input)
		if got != tc.want {
			t.Fatalf("SafeLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDefaultLabel(t *testing.T) {
	if got := DefaultLabel(1); got != "clip_01" {
		t.Fatalf("DefaultLabel(1) = %q, want %q", got, "clip_01")
	}
	if got := DefaultLabel(10); got != "clip_10" {
		t.Fatalf("DefaultLabel(10) = %q, want %q", got, "clip_10")
	}
}
