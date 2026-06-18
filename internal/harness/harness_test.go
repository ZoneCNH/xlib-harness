package harness

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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

func TestGeneratePreflightsExistingTargets(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "module", "demo", "SPEC.md")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatalf("create existing target dir: %v", err)
	}
	if err := os.WriteFile(existing, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("create existing target: %v", err)
	}

	if _, err := Generate("demo", dir, false); err == nil {
		t.Fatal("expected existing target to fail before writing any template files")
	}
	if _, err := os.Stat(filepath.Join(dir, "module", "demo", "IMPLEMENTATION-PLAN.md")); !os.IsNotExist(err) {
		t.Fatalf("generate wrote a partial template before failing; err=%v", err)
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

func TestRunJSONFailureIsItemizedAndNonZero(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"check", "--json", "../../fixtures/broken-module", "--profile", "spec"}, &out, &err)
	if code == 0 {
		t.Fatalf("expected failed JSON check to exit nonzero; stdout=%s stderr=%s", out.String(), err.String())
	}
	for _, want := range []string{`"passed": false`, `"checks"`, `"spec-fr-ac-tc-structure"`, `"spec-section-depth"`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("JSON failure output missing %s: %s", want, out.String())
		}
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

func TestRunHelpAndUnknownCommand(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := Run([]string{"help"}, &out, &stderr); code != 0 {
		t.Fatalf("help code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "usage: xlib-harness") {
		t.Fatalf("help output missing usage: %q", out.String())
	}

	out.Reset()
	stderr.Reset()
	if code := Run([]string{"bogus"}, &out, &stderr); code != 2 {
		t.Fatalf("unknown command code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `unknown command "bogus"`) || !strings.Contains(stderr.String(), "usage: xlib-harness") {
		t.Fatalf("unknown command stderr missing details: %q", stderr.String())
	}

	if code := Run(nil, nil, nil); code != 2 {
		t.Fatalf("empty args with nil writers code=%d, want 2", code)
	}
}

func TestRunGenerateCreatesFilesAndHonorsForce(t *testing.T) {
	dir := t.TempDir()
	var out, stderr bytes.Buffer
	code := Run([]string{"generate", "cli-module", "--output", dir}, &out, &stderr)
	if code != 0 {
		t.Fatalf("generate code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "created module/cli-module/SPEC.md") {
		t.Fatalf("generate stdout missing created file: %q", out.String())
	}

	out.Reset()
	stderr.Reset()
	if code := Run([]string{"generate", "cli-module", "--output", dir}, &out, &stderr); code != 1 {
		t.Fatalf("generate over existing file code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "target file exists") {
		t.Fatalf("expected existing-file error, got %q", stderr.String())
	}

	out.Reset()
	stderr.Reset()
	if code := Run([]string{"generate", "cli-module", "--output", dir, "--force"}, &out, &stderr); code != 0 {
		t.Fatalf("force generate code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
}

func TestRunGenerateRejectsBadCLIArgs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing module", []string{"generate"}, "generate requires exactly one module name"},
		{"missing output", []string{"generate", "--output"}, "--output requires a value"},
		{"unknown flag", []string{"generate", "--bogus", "mod"}, "unknown generate flag --bogus"},
		{"invalid module", []string{"generate", "bad/name", "--output", t.TempDir()}, "invalid module name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if code := Run(tc.args, &out, &stderr); code == 0 {
				t.Fatalf("expected nonzero code for %v; stdout=%s stderr=%s", tc.args, out.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr=%q, want substring %q", stderr.String(), tc.want)
			}
		})
	}
}

func TestRunCheckJSONAndNegativeCLIPaths(t *testing.T) {
	dir := t.TempDir()
	if _, err := Generate("json-module", dir, false); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	modulePath := filepath.Join(dir, "module", "json-module")

	var out, stderr bytes.Buffer
	code := Run([]string{"check", modulePath, "--profile", "full", "--json"}, &out, &stderr)
	if code != 0 {
		t.Fatalf("check json code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	var res CheckResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("check output is not valid JSON: %v\n%s", err, out.String())
	}
	if !res.Passed || res.Profile != "full" || res.Module != modulePath {
		t.Fatalf("unexpected check result: %+v", res)
	}

	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"unknown profile", []string{"check", modulePath, "--profile", "nope", "--json"}, 1, "unknown profile"},
		{"missing path", []string{"check", "--profile", "full"}, 2, "check requires exactly one module path"},
		{"missing profile", []string{"check", modulePath, "--profile"}, 2, "--profile requires a value"},
		{"unknown flag", []string{"check", modulePath, "--bogus"}, 2, "unknown check flag --bogus"},
		{"missing module path", []string{"check", filepath.Join(dir, "missing"), "--json"}, 1, "module path is not readable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out.Reset()
			stderr.Reset()
			if got := Run(tc.args, &out, &stderr); got != tc.code {
				t.Fatalf("code=%d want %d stdout=%s stderr=%s", got, tc.code, out.String(), stderr.String())
			}
			combined := out.String() + stderr.String()
			if !strings.Contains(combined, tc.want) {
				t.Fatalf("output=%q, want substring %q", combined, tc.want)
			}
		})
	}
}

func TestRunValidateTemplateJSONAndMisuse(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := Run([]string{"validate", "--template", "--json"}, &out, &stderr); code != 0 {
		t.Fatalf("validate json code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	var res CheckResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("validate output is not JSON: %v\n%s", err, out.String())
	}
	if !res.Passed || res.Profile != "full" {
		t.Fatalf("unexpected validate result: %+v", res)
	}

	out.Reset()
	stderr.Reset()
	if code := Run([]string{"validate", "--json"}, &out, &stderr); code != 2 {
		t.Fatalf("validate misuse code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "validate currently supports only --template") {
		t.Fatalf("validate misuse stderr missing guidance: %q", stderr.String())
	}
}

func TestCheckIsReadOnlyForGeneratedModule(t *testing.T) {
	dir := t.TempDir()
	if _, err := Generate("readonly-module", dir, false); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	modulePath := filepath.Join(dir, "module", "readonly-module")
	before := snapshotTree(t, modulePath)
	res := Check(modulePath, "full")
	if !res.Passed {
		t.Fatalf("generated module should pass full check: %+v", res)
	}
	after := snapshotTree(t, modulePath)
	if len(after) != len(before) {
		t.Fatalf("check mutated file count: before=%d after=%d", len(before), len(after))
	}
	for path, want := range before {
		got, ok := after[path]
		if !ok {
			t.Fatalf("check removed %s", path)
		}
		if got != want {
			t.Fatalf("check mutated %s: before=%q after=%q", path, want, got)
		}
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	items := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		items[filepath.ToSlash(rel)] = info.Mode().String() + ":" + info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z07:00") + ":" + strconv.FormatInt(info.Size(), 10)
		return nil
	}); err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return items
}
