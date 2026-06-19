package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
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

func TestStdlibHarnessAPI(t *testing.T) {
	dir := t.TempDir()
	h := StdlibHarness{OutputRoot: dir}
	result, err := h.Generate("api-module")
	if err != nil {
		t.Fatalf("StdlibHarness.Generate returned error: %v", err)
	}
	if result.Module != "api-module" || result.Root != dir || len(result.FilesCreated) == 0 {
		t.Fatalf("unexpected generate result: %+v", result)
	}
	if res := h.Check(filepath.Join(dir, "module", "api-module"), ProfileFull); !res.Passed {
		t.Fatalf("StdlibHarness.Check failed: %+v", res)
	}
	if _, err := h.Generate("api-module"); err == nil {
		t.Fatal("expected StdlibHarness.Generate to return existing-file error")
	}

	override := t.TempDir()
	result, err = h.Generate("api-module-override", WithOutputRoot(override), WithForce(true))
	if err != nil {
		t.Fatalf("StdlibHarness.Generate with options returned error: %v", err)
	}
	if result.Root != override {
		t.Fatalf("override root=%q, want %q", result.Root, override)
	}

	defaultRoot := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(defaultRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	result, err = (StdlibHarness{}).Generate("default-root")
	if err != nil {
		t.Fatalf("StdlibHarness.Generate default root returned error: %v", err)
	}
	if result.Root != "." {
		t.Fatalf("default root=%q, want .", result.Root)
	}
}

func TestGenerateOperationalFailures(t *testing.T) {
	t.Run("root abs fails", func(t *testing.T) {
		replaceHook(t, &absPath, func(string) (string, error) {
			return "", errors.New("abs root failed")
		})
		if _, err := Generate("demo", t.TempDir(), false); err == nil || !strings.Contains(err.Error(), "abs root failed") {
			t.Fatalf("err=%v, want root abs failure", err)
		}
	})

	t.Run("target abs fails", func(t *testing.T) {
		calls := 0
		replaceHook(t, &absPath, func(path string) (string, error) {
			calls++
			if calls == 2 {
				return "", errors.New("abs target failed")
			}
			return filepath.Clean(path), nil
		})
		if _, err := Generate("demo", t.TempDir(), false); err == nil || !strings.Contains(err.Error(), "abs target failed") {
			t.Fatalf("err=%v, want target abs failure", err)
		}
	})

	t.Run("target escape is refused", func(t *testing.T) {
		calls := 0
		replaceHook(t, &absPath, func(path string) (string, error) {
			calls++
			if calls == 1 {
				return "/safe/root", nil
			}
			return "/outside/module/demo", nil
		})
		if _, err := Generate("demo", t.TempDir(), false); err == nil || !strings.Contains(err.Error(), "outside output module") {
			t.Fatalf("err=%v, want escape refusal", err)
		}
	})

	t.Run("stat fails", func(t *testing.T) {
		replaceHook(t, &statPath, func(string) (os.FileInfo, error) {
			return nil, errors.New("stat denied")
		})
		if _, err := Generate("demo", t.TempDir(), false); err == nil || !strings.Contains(err.Error(), "stat denied") {
			t.Fatalf("err=%v, want stat failure", err)
		}
	})

	t.Run("task dir mkdir fails", func(t *testing.T) {
		replaceHook(t, &mkdirAll, func(string, fs.FileMode) error {
			return errors.New("mkdir task failed")
		})
		if _, err := Generate("demo", t.TempDir(), true); err == nil || !strings.Contains(err.Error(), "mkdir task failed") {
			t.Fatalf("err=%v, want mkdir failure", err)
		}
	})

	t.Run("file dir mkdir fails", func(t *testing.T) {
		calls := 0
		replaceHook(t, &mkdirAll, func(string, fs.FileMode) error {
			calls++
			if calls == 2 {
				return errors.New("mkdir file failed")
			}
			return nil
		})
		if _, err := Generate("demo", t.TempDir(), true); err == nil || !strings.Contains(err.Error(), "mkdir file failed") {
			t.Fatalf("err=%v, want file mkdir failure", err)
		}
	})

	t.Run("write fails", func(t *testing.T) {
		replaceHook(t, &writeFile, func(string, []byte, fs.FileMode) error {
			return errors.New("write failed")
		})
		if _, err := Generate("demo", t.TempDir(), true); err == nil || !strings.Contains(err.Error(), "write failed") {
			t.Fatalf("err=%v, want write failure", err)
		}
	})
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

func TestSpecMissingFilesAreItemized(t *testing.T) {
	dir := t.TempDir()
	res := Check(dir, "spec")
	if res.Passed {
		t.Fatalf("empty module should fail spec checks: %+v", res)
	}
	for _, want := range []string{"required-file:SPEC.md", "required-dir:tasks", "spec-readable"} {
		if !strings.Contains(flattenNamesAndDetails(res), want) {
			t.Fatalf("result missing %q: %+v", want, res)
		}
	}
}

func TestSpecReadFailureIsReported(t *testing.T) {
	replaceHook(t, &readFile, func(string) ([]byte, error) {
		return nil, errors.New("read spec failed")
	})
	res := Check("../../fixtures/compliant-module", "spec")
	if res.Passed {
		t.Fatalf("spec read failure should fail: %+v", res)
	}
	if !strings.Contains(flattenDetails(res), "read spec failed") {
		t.Fatalf("missing read failure detail: %+v", res)
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

func TestBoundaryOperationalFailures(t *testing.T) {
	t.Run("walk callback and root errors", func(t *testing.T) {
		replaceHook(t, &walkDir, func(root string, fn fs.WalkDirFunc) error {
			_ = fn(filepath.Join(root, "blocked.go"), fakeDirEntry{name: "blocked.go"}, errors.New("walk callback failed"))
			return errors.New("walk root failed")
		})
		res := Check("../../fixtures/compliant-module", "boundary")
		if res.Passed {
			t.Fatalf("walk errors should fail boundary: %+v", res)
		}
		for _, want := range []string{"walk callback failed", "walk root failed"} {
			if !strings.Contains(flattenDetails(res), want) {
				t.Fatalf("missing %q in %+v", want, res)
			}
		}
	})

	t.Run("read errors", func(t *testing.T) {
		replaceHook(t, &walkDir, func(root string, fn fs.WalkDirFunc) error {
			if err := fn(filepath.Join(root, "go.mod"), fakeDirEntry{name: "go.mod"}, nil); err != nil {
				return err
			}
			return fn(filepath.Join(root, "bad.go"), fakeDirEntry{name: "bad.go"}, nil)
		})
		replaceHook(t, &readFile, func(path string) ([]byte, error) {
			return nil, fmtError("read failed for " + filepath.Base(path))
		})
		res := Check("../../fixtures/compliant-module", "boundary")
		if res.Passed {
			t.Fatalf("read errors should fail boundary: %+v", res)
		}
		for _, want := range []string{"read failed for go.mod", "read failed for bad.go"} {
			if !strings.Contains(flattenDetails(res), want) {
				t.Fatalf("missing %q in %+v", want, res)
			}
		}
	})

	t.Run("go mod forbidden dependency", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\nrequire github.com/ZoneCNH/xlib-standard v0.0.0\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
		res := Check(dir, "boundary")
		if res.Passed {
			t.Fatalf("forbidden go.mod dependency should fail: %+v", res)
		}
		if !strings.Contains(flattenDetails(res), "xlib-standard") {
			t.Fatalf("missing dependency detail: %+v", res)
		}
	})

	t.Run("invalid go import syntax", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package bad\nimport \n"), 0o644); err != nil {
			t.Fatalf("write bad.go: %v", err)
		}
		res := Check(dir, "boundary")
		if res.Passed {
			t.Fatalf("invalid go file should fail boundary: %+v", res)
		}
		if !strings.Contains(flattenDetails(res), "missing import path") {
			t.Fatalf("missing parser detail: %+v", res)
		}
	})
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

func flattenNamesAndDetails(res CheckResult) string {
	var b strings.Builder
	for _, item := range res.Checks {
		b.WriteString(item.Name)
		b.WriteByte(':')
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

	out.Reset()
	stderr.Reset()
	if code := Run([]string{"validate", "--bogus"}, &out, &stderr); code != 2 {
		t.Fatalf("validate flag error code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
	}
}

func TestRunValidateOperationalFailures(t *testing.T) {
	t.Run("mkdir temp fails", func(t *testing.T) {
		replaceHook(t, &mkdirTemp, func(string, string) (string, error) {
			return "", errors.New("temp failed")
		})
		var out, stderr bytes.Buffer
		if code := Run([]string{"validate", "--template"}, &out, &stderr); code != 1 {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "temp failed") {
			t.Fatalf("stderr missing temp failure: %q", stderr.String())
		}
	})

	t.Run("generate fails", func(t *testing.T) {
		replaceHook(t, &generateDocs, func(string, string, bool) ([]string, error) {
			return nil, errors.New("generate failed")
		})
		var out, stderr bytes.Buffer
		if code := Run([]string{"validate", "--template"}, &out, &stderr); code != 1 {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "generate failed") {
			t.Fatalf("stderr missing generate failure: %q", stderr.String())
		}
	})

	t.Run("check fails", func(t *testing.T) {
		replaceHook(t, &checkDocs, func(modulePath, profile string) CheckResult {
			return CheckResult{
				Module:  modulePath,
				Profile: profile,
				Passed:  false,
				Checks:  []CheckItem{{Name: "forced", Passed: false, Detail: "forced fail"}},
				Summary: "forced summary",
			}
		})
		var out, stderr bytes.Buffer
		if code := Run([]string{"validate", "--template"}, &out, &stderr); code != 1 {
			t.Fatalf("code=%d stdout=%s stderr=%s", code, out.String(), stderr.String())
		}
		if !strings.Contains(out.String(), "forced fail") {
			t.Fatalf("stdout missing forced check detail: %q", out.String())
		}
	})
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

func TestFormatOperationalFailuresAndRules(t *testing.T) {
	t.Run("walk errors", func(t *testing.T) {
		replaceHook(t, &walkDir, func(root string, fn fs.WalkDirFunc) error {
			_ = fn(filepath.Join(root, "blocked.md"), fakeDirEntry{name: "blocked.md"}, errors.New("format callback failed"))
			return errors.New("format walk failed")
		})
		res := Check("../../fixtures/compliant-module", "spec")
		if res.Passed {
			t.Fatalf("format walk errors should fail spec profile: %+v", res)
		}
		for _, want := range []string{"format callback failed", "format walk failed"} {
			if !strings.Contains(flattenDetails(res), want) {
				t.Fatalf("missing %q in %+v", want, res)
			}
		}
	})

	t.Run("open fails", func(t *testing.T) {
		replaceHook(t, &openScanFile, func(string) (io.ReadCloser, error) {
			return nil, errors.New("open failed")
		})
		res := Check("../../fixtures/compliant-module", "spec")
		if res.Passed {
			t.Fatalf("open failure should fail spec profile: %+v", res)
		}
		if !strings.Contains(flattenDetails(res), "open failed") {
			t.Fatalf("missing open failure: %+v", res)
		}
	})

	t.Run("scanner fails", func(t *testing.T) {
		replaceHook(t, &openScanFile, func(string) (io.ReadCloser, error) {
			return failingReadCloser{}, nil
		})
		res := Check("../../fixtures/compliant-module", "spec")
		if res.Passed {
			t.Fatalf("scanner failure should fail spec profile: %+v", res)
		}
		if !strings.Contains(flattenDetails(res), "scan failed") {
			t.Fatalf("missing scanner failure: %+v", res)
		}
	})

	t.Run("line rules", func(t *testing.T) {
		table := markdownTableState{}
		var issues []string
		issues = append(issues, formatLineIssues("doc.md", 1, "| A | B |", &table)...)
		issues = append(issues, formatLineIssues("doc.md", 2, "| A | B | C |", &table)...)
		issues = append(issues, formatLineIssues("doc.md", 3, "", &table)...)
		issues = append(issues, formatLineIssues("doc.md", 4, "| A | B |", &table)...)
		issues = append(issues, formatLineIssues("doc.md", 5, "[ok](README.md) [bad]()", &table)...)
		detail := strings.Join(issues, "\n")
		for _, want := range []string{"table column count changed", "empty markdown link target"} {
			if !strings.Contains(detail, want) {
				t.Fatalf("issues=%v, want %q", issues, want)
			}
		}
	})
}

func TestTraceabilityReadFailure(t *testing.T) {
	replaceHook(t, &readFile, func(path string) ([]byte, error) {
		return nil, errors.New("trace read failed for " + filepath.Base(path))
	})
	res := Check("../../fixtures/compliant-module", "full")
	if res.Passed {
		t.Fatalf("trace read failure should fail full profile: %+v", res)
	}
	if !strings.Contains(flattenDetails(res), "trace read failed") {
		t.Fatalf("missing trace read detail: %+v", res)
	}
}

func TestTableAndSpecHelpersReportMalformedRows(t *testing.T) {
	spec := strings.Join([]string{
		"| ID | Requirement | WHEN | THEN |",
		"| --- | --- | --- | --- |",
		"| FR-999 | bad | WHEN triggered | missing outcome |",
		"| AC ID | FR Ref | Criterion |",
		"| --- | --- | --- |",
		"| AC-999 | FR-999 | no testcase here |",
		"| TC ID | Covers | Command |",
		"| --- | --- | --- |",
		"| TC-999 | AC-999 | inspect manually |",
	}, "\n")
	for name, issues := range map[string][]string{
		"fr": functionalRequirementIssues(spec),
		"ac": acceptanceCriteriaIssues(spec),
		"tc": testCaseIssues(spec),
	} {
		if len(issues) == 0 {
			t.Fatalf("%s issues empty for malformed rows", name)
		}
	}
	if got := tableRowID("| | | |"); got != "" {
		t.Fatalf("blank table row id=%q, want empty", got)
	}
}

func replaceHook[T any](t *testing.T, target *T, replacement T) {
	t.Helper()
	original := *target
	*target = replacement
	t.Cleanup(func() {
		*target = original
	})
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string {
	return f.name
}

func (f fakeDirEntry) IsDir() bool {
	return f.dir
}

func (f fakeDirEntry) Type() fs.FileMode {
	if f.dir {
		return fs.ModeDir
	}
	return 0
}

func (f fakeDirEntry) Info() (fs.FileInfo, error) {
	return nil, errors.New("fake entry has no info")
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("scan failed")
}

func (failingReadCloser) Close() error {
	return nil
}

type fmtError string

func (e fmtError) Error() string {
	return string(e)
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

func BenchmarkGenerate(b *testing.B) {
	root := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Generate("bench-module-"+strconv.Itoa(i), root, false); err != nil {
			b.Fatalf("Generate returned error: %v", err)
		}
	}
}

func BenchmarkCheckFullProfile(b *testing.B) {
	dir := b.TempDir()
	if _, err := Generate("bench-module", dir, false); err != nil {
		b.Fatalf("Generate returned error: %v", err)
	}
	modulePath := filepath.Join(dir, "module", "bench-module")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if res := Check(modulePath, "full"); !res.Passed {
			b.Fatalf("Check failed: %+v", res)
		}
	}
}
