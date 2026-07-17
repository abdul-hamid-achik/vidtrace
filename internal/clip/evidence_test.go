package clip

import (
	"testing"
)

func TestSpecsFromEvidencePadsAndMerges(t *testing.T) {
	t.Parallel()

	specs := SpecsFromEvidence([]EvidenceHit{
		{TimeSeconds: 10, Label: "a"},
		{TimeSeconds: 11, Label: "b"},
		{TimeSeconds: 40, Label: "c"},
	}, 2)
	if len(specs) != 2 {
		t.Fatalf("expected 2 merged specs, got %d: %#v", len(specs), specs)
	}
	if specs[0].StartSec != 8 || specs[0].EndSec != 13 {
		t.Fatalf("first range = %.1f-%.1f, want 8-13", specs[0].StartSec, specs[0].EndSec)
	}
	if specs[1].StartSec != 38 || specs[1].EndSec != 42 {
		t.Fatalf("second range = %.1f-%.1f, want 38-42", specs[1].StartSec, specs[1].EndSec)
	}
}

func TestSpecsFromEvidenceDefaultPad(t *testing.T) {
	t.Parallel()

	specs := SpecsFromEvidence([]EvidenceHit{{TimeSeconds: 1, Label: "hit"}}, 0)
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].StartSec != 0 || specs[0].EndSec != 3 {
		t.Fatalf("default pad range = %.1f-%.1f, want 0-3", specs[0].StartSec, specs[0].EndSec)
	}
}
