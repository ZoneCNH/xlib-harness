package harness

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCreatesRequiredModuleFiles(t *testing.T) {
	dir := t.TempDir()
	files, err := Generate("test-module", dir, false)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	want := []string{"SPEC.md", "TRACEABILITY.md", "goal.md", "IMPLEMENTATION-PLAN.md", filepath.Join("tasks", "TASK-001.md")}
	for _, rel := range want {
		if _, err := os.Stat(filepath.Join(dir, "module", "test-module", rel)); err != nil {
			t.Fatalf("missing generated %s: %v; files=%v", rel, err, files)
		}
	}
}

func TestGenerateRejectsTraversal(t *testing.T) {
	if _, err := Generate("../escape", t.TempDir(), false); err == nil {
		t.Fatal("expected traversal module name to be rejected")
	}
}

func TestCheckProfilesAndFailures(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		profile string
		pass    bool
		detail  string
	}{
		{"compliant spec", "../../fixtures/compliant-module", "spec", true, "all"},
		{"broken spec", "../../fixtures/broken-module", "spec", false, "missing"},
		{"bad dep", "../../fixtures/module-with-bad-dep", "boundary", false, "forbidden"},
		{"format", "../../fixtures/format-issues", "spec", false, "format"},
		{"trace", "../../fixtures/broken-trace", "full", false, "TRACEABILITY"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := Check(tc.path, tc.profile)
			if res.Passed != tc.pass {
				t.Fatalf("Passed=%v want %v: %+v", res.Passed, tc.pass, res)
			}
			if !strings.Contains(strings.ToLower(res.Summary+flattenDetails(res)), strings.ToLower(tc.detail)) {
				t.Fatalf("result does not mention %q: %+v", tc.detail, res)
			}
		})
	}
}

func TestBoundaryRejectsXlibStandardImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte(`package baddep

import _ "github.com/ZoneCNH/xlib-standard"
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	res := Check(dir, "boundary")
	if res.Passed {
		t.Fatalf("expected xlib-standard import to fail dependency boundary: %+v", res)
	}
	if !strings.Contains(flattenDetails(res), "xlib-standard") {
		t.Fatalf("expected boundary details to mention xlib-standard: %+v", res)
	}
}

func TestRunReturnsNonZeroForFailedGate(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"check", "../../fixtures/broken-module", "--profile", "spec"}, &out, &err)
	if code == 0 {
		t.Fatalf("expected failed check to exit nonzero; stdout=%s stderr=%s", out.String(), err.String())
	}
	if !strings.Contains(out.String(), "FAIL") {
		t.Fatalf("expected itemized failure output, got %q", out.String())
	}
}

func TestValidateTemplate(t *testing.T) {
	var out, err bytes.Buffer
	if code := Run([]string{"validate", "--template"}, &out, &err); code != 0 {
		t.Fatalf("validate failed code=%d stdout=%s stderr=%s", code, out.String(), err.String())
	}
}

func flattenDetails(res CheckResult) string {
	var b strings.Builder
	for _, item := range res.Checks {
		b.WriteString(item.Detail)
		b.WriteByte('\n')
	}
	return b.String()
}
