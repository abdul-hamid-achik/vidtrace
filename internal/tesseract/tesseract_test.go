package tesseract

import (
	"reflect"
	"testing"
)

func TestParseLanguages(t *testing.T) {
	// tesseract --list-langs prints a header line followed by one language per line.
	out := "List of available languages (3):\nosd\neng\nspa\n"
	got := parseLanguages(out)
	want := []string{"eng", "osd", "spa"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseLanguages = %v, want %v", got, want)
	}
}

func TestMissingLanguages(t *testing.T) {
	if got := MissingLanguages([]string{"eng", "spa", "fra"}, []string{"eng", "osd"}); !reflect.DeepEqual(got, []string{"spa", "fra"}) {
		t.Fatalf("missing = %v, want [spa fra]", got)
	}
	if got := MissingLanguages([]string{"eng"}, []string{"eng", "spa"}); len(got) != 0 {
		t.Fatalf("expected nothing missing, got %v", got)
	}
	// Blanks and duplicates are ignored, request order preserved.
	if got := MissingLanguages([]string{"spa", "", "spa", "fra"}, []string{"eng"}); !reflect.DeepEqual(got, []string{"spa", "fra"}) {
		t.Fatalf("missing = %v, want [spa fra]", got)
	}
}

func TestSplitLanguages(t *testing.T) {
	if got := SplitLanguages("eng+spa, fra"); !reflect.DeepEqual(got, []string{"eng", "fra", "spa"}) {
		t.Fatalf("SplitLanguages = %v, want [eng fra spa]", got)
	}
	if got := SplitLanguages(" "); len(got) != 0 {
		t.Fatalf("expected empty for blank input, got %v", got)
	}
}
