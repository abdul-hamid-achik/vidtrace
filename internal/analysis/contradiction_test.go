package analysis

import "testing"

func TestDetectContradiction(t *testing.T) {
	t.Parallel()

	if !detectContradiction("Login failed after submit and cannot continue.", "Login successfully completed and saved.") {
		t.Fatal("expected contradiction when ticket fails and evidence succeeds")
	}
	if detectContradiction("Login failed after submit.", "Login failed after submit error.") {
		t.Fatal("should not contradict when evidence also shows failure")
	}
	if detectContradiction("User opens dashboard.", "Dashboard loads successfully.") {
		t.Fatal("should not contradict without failure language in the ticket")
	}
}
