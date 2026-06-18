package harness

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type CheckResult struct {
	Module  string      `json:"module"`
	Profile string      `json:"profile"`
	Passed  bool        `json:"passed"`
	Checks  []CheckItem `json:"checks"`
	Summary string      `json:"summary"`
}

type CheckItem struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type options struct {
	stdout io.Writer
	stderr io.Writer
}

func Run(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	opts := options{stdout: stdout, stderr: stderr}
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "generate":
		return runGenerate(args[1:], opts)
	case "check":
		return runCheck(args[1:], opts)
	case "validate":
		return runValidate(args[1:], opts)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: xlib-harness <generate|check|validate> [options]")
}

func runGenerate(args []string, opts options) int {
	output, force, positional, err := parseGenerateArgs(args)
	if err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(opts.stderr, "generate requires exactly one module name")
		return 2
	}
	files, err := Generate(positional[0], output, force)
	if err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
	for _, f := range files {
		fmt.Fprintf(opts.stdout, "created %s\n", f)
	}
	return 0
}

func runCheck(args []string, opts options) int {
	profile, asJSON, positional, err := parseCheckArgs(args)
	if err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 2
	}
	if len(positional) != 1 {
		fmt.Fprintln(opts.stderr, "check requires exactly one module path")
		return 2
	}
	res := Check(positional[0], profile)
	writeResult(opts.stdout, res, asJSON)
	if !res.Passed {
		return 1
	}
	return 0
}

func parseGenerateArgs(args []string) (string, bool, []string, error) {
	output := "."
	force := false
	positional := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--output":
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "-") {
				return "", false, nil, errors.New("--output requires a value")
			}
			output = args[i]
		case "--force":
			force = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", false, nil, fmt.Errorf("unknown generate flag %s", args[i])
			}
			positional = append(positional, args[i])
		}
	}
	return output, force, positional, nil
}

func parseCheckArgs(args []string) (string, bool, []string, error) {
	profile := "full"
	asJSON := false
	positional := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "-") {
				return "", false, nil, errors.New("--profile requires a value")
			}
			profile = args[i]
		case "--json":
			asJSON = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", false, nil, fmt.Errorf("unknown check flag %s", args[i])
			}
			positional = append(positional, args[i])
		}
	}
	return profile, asJSON, positional, nil
}

func runValidate(args []string, opts options) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(opts.stderr)
	template := fs.Bool("template", false, "validate embedded template")
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !*template || fs.NArg() != 0 {
		fmt.Fprintln(opts.stderr, "validate currently supports only --template")
		return 2
	}
	dir, err := os.MkdirTemp("", "xlib-harness-template-*")
	if err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)
	if _, err := Generate("template-self", dir, false); err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
	res := Check(filepath.Join(dir, "module", "template-self"), "full")
	writeResult(opts.stdout, res, *asJSON)
	if !res.Passed {
		return 1
	}
	return 0
}

func writeResult(w io.Writer, res CheckResult, asJSON bool) {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
		return
	}
	status := "PASS"
	if !res.Passed {
		status = "FAIL"
	}
	fmt.Fprintf(w, "%s %s (%s)\n", status, res.Module, res.Profile)
	for _, item := range res.Checks {
		itemStatus := "PASS"
		if !item.Passed {
			itemStatus = "FAIL"
		}
		fmt.Fprintf(w, "[%s] %s: %s\n", itemStatus, item.Name, item.Detail)
	}
	fmt.Fprintln(w, res.Summary)
}

var moduleNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func Generate(module, outputRoot string, force bool) ([]string, error) {
	if !moduleNameRE.MatchString(module) || strings.Contains(module, "..") || strings.ContainsAny(module, `/\\`) {
		return nil, fmt.Errorf("invalid module name %q: use letters, numbers, dash, or underscore only", module)
	}
	root, err := filepath.Abs(outputRoot)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(root, "module", module)
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, err
	}
	allowedPrefix := filepath.Join(root, "module") + string(os.PathSeparator)
	if !strings.HasPrefix(cleanTarget+string(os.PathSeparator), allowedPrefix) {
		return nil, errors.New("refusing to write outside output module directory")
	}
	entries := map[string]string{
		"SPEC.md":                             specTemplate(module),
		"TRACEABILITY.md":                     traceTemplate(module),
		"goal.md":                             fmt.Sprintf("# %s Goal\n\nDeliver a compliant Foundation module with documented FR, AC, and TC coverage.\n", module),
		"IMPLEMENTATION-PLAN.md":              fmt.Sprintf("# %s Implementation Plan\n\n1. Confirm SPEC.\n2. Implement tasks.\n3. Verify traceability.\n", module),
		filepath.Join("tasks", "TASK-001.md"): fmt.Sprintf("# TASK-001 %s bootstrap\n\nImplement and verify the module skeleton.\n", module),
	}
	paths := make([]string, 0, len(entries))
	for rel := range entries {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	created := make([]string, 0, len(paths)+1)
	if err := os.MkdirAll(filepath.Join(target, "tasks"), 0o755); err != nil {
		return nil, err
	}
	created = append(created, filepath.ToSlash(filepath.Join("module", module, "tasks")))
	for _, rel := range paths {
		dst := filepath.Join(target, rel)
		if !force {
			if _, err := os.Stat(dst); err == nil {
				return nil, fmt.Errorf("target file exists: %s", dst)
			}
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dst, []byte(entries[rel]), 0o644); err != nil {
			return nil, err
		}
		created = append(created, filepath.ToSlash(filepath.Join("module", module, rel)))
	}
	sort.Strings(created)
	return created, nil
}

func Check(modulePath, profile string) CheckResult {
	res := CheckResult{Module: modulePath, Profile: profile, Passed: true}
	add := func(name string, pass bool, detail string) {
		res.Checks = append(res.Checks, CheckItem{Name: name, Passed: pass, Detail: detail})
		if !pass {
			res.Passed = false
		}
	}
	if _, err := os.Stat(modulePath); err != nil {
		add("module-exists", false, err.Error())
		res.Summary = "module path is not readable"
		return res
	}
	switch profile {
	case "spec":
		runSpecChecks(modulePath, add)
		runFormatChecks(modulePath, add)
	case "boundary":
		runBoundaryChecks(modulePath, add)
	case "full":
		runSpecChecks(modulePath, add)
		runBoundaryChecks(modulePath, add)
		runFormatChecks(modulePath, add)
		runTraceChecks(modulePath, add)
	default:
		add("profile", false, "unknown profile: "+profile)
	}
	failed := 0
	for _, item := range res.Checks {
		if !item.Passed {
			failed++
		}
	}
	if failed == 0 {
		res.Summary = fmt.Sprintf("all %d checks passed", len(res.Checks))
	} else {
		res.Summary = fmt.Sprintf("%d of %d checks failed", failed, len(res.Checks))
	}
	return res
}

func runSpecChecks(modulePath string, add func(string, bool, string)) {
	required := []string{"SPEC.md", "TRACEABILITY.md", "goal.md", "IMPLEMENTATION-PLAN.md"}
	for _, rel := range required {
		p := filepath.Join(modulePath, rel)
		info, err := os.Stat(p)
		add("required-file:"+rel, err == nil && !info.IsDir(), detailForFile(p, err))
	}
	tasksInfo, err := os.Stat(filepath.Join(modulePath, "tasks"))
	add("required-dir:tasks", err == nil && tasksInfo.IsDir(), detailForFile(filepath.Join(modulePath, "tasks"), err))
	spec, err := os.ReadFile(filepath.Join(modulePath, "SPEC.md"))
	if err != nil {
		add("spec-readable", false, err.Error())
		return
	}
	text := string(spec)
	missing := missingTokens(text, []string{"FR-001", "WHEN", "THEN", "Acceptance Criteria", "TC-001"})
	add("spec-fr-ac-tc-structure", len(missing) == 0, missingDetail(missing, "SPEC includes FR/WHEN/THEN/AC/TC markers"))
	headings := countHeadings(text)
	add("spec-section-depth", headings >= 6, fmt.Sprintf("%d markdown headings found", headings))
}

func runBoundaryChecks(modulePath string, add func(string, bool, string)) {
	forbidden := []string{"observex", "configx", "resiliencx", "schedulex", "testkitx", "xlib-standard"}
	var hits []string
	_ = filepath.WalkDir(modulePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			hits = append(hits, fmt.Sprintf("%s: %v", path, readErr))
			return nil
		}
		for _, token := range forbidden {
			if strings.Contains(string(data), token) {
				hits = append(hits, fmt.Sprintf("%s imports or references forbidden dependency %s", filepath.ToSlash(path), token))
			}
		}
		return nil
	})
	add("dependency-boundary", len(hits) == 0, missingDetail(hits, "no forbidden production dependencies found"))
}

func runFormatChecks(modulePath string, add func(string, bool, string)) {
	var issues []string
	_ = filepath.WalkDir(modulePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", path, openErr))
			return nil
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		line := 0
		for scanner.Scan() {
			line++
			txt := scanner.Text()
			if strings.TrimRight(txt, " \t") != txt {
				issues = append(issues, fmt.Sprintf("%s:%d trailing whitespace", filepath.ToSlash(path), line))
			}
			if strings.Contains(txt, "FORMAT-ISSUE") {
				issues = append(issues, fmt.Sprintf("%s:%d explicit format issue marker", filepath.ToSlash(path), line))
			}
		}
		if err := scanner.Err(); err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", path, err))
		}
		return nil
	})
	add("markdown-format", len(issues) == 0, missingDetail(issues, "markdown/yaml formatting passed"))
}

func runTraceChecks(modulePath string, add func(string, bool, string)) {
	specBytes, specErr := os.ReadFile(filepath.Join(modulePath, "SPEC.md"))
	traceBytes, traceErr := os.ReadFile(filepath.Join(modulePath, "TRACEABILITY.md"))
	if specErr != nil || traceErr != nil {
		add("traceability-readable", false, fmt.Sprintf("SPEC: %v TRACEABILITY: %v", specErr, traceErr))
		return
	}
	frs := uniqueMatches(`FR-\d{3}`, string(specBytes))
	trace := string(traceBytes)
	var gaps []string
	for _, fr := range frs {
		if !strings.Contains(trace, fr) {
			gaps = append(gaps, "missing "+fr+" in TRACEABILITY.md")
		}
	}
	for _, token := range []string{"AC-001", "TC-001"} {
		if !strings.Contains(trace, token) {
			gaps = append(gaps, "missing "+token+" in TRACEABILITY.md")
		}
	}
	add("traceability-chain", len(gaps) == 0, missingDetail(gaps, "FR/AC/TC chain closed"))
}

func detailForFile(path string, err error) string {
	if err != nil {
		return err.Error()
	}
	return filepath.ToSlash(path) + " present"
}

func missingTokens(text string, tokens []string) []string {
	var missing []string
	for _, token := range tokens {
		if !strings.Contains(text, token) {
			missing = append(missing, "missing "+token)
		}
	}
	return missing
}

func missingDetail(items []string, ok string) string {
	if len(items) == 0 {
		return ok
	}
	return strings.Join(items, "; ")
}

func countHeadings(text string) int {
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "#") {
			count++
		}
	}
	return count
}

func uniqueMatches(expr, text string) []string {
	re := regexp.MustCompile(expr)
	found := map[string]bool{}
	for _, match := range re.FindAllString(text, -1) {
		found[match] = true
	}
	out := make([]string, 0, len(found))
	for match := range found {
		out = append(out, match)
	}
	sort.Strings(out)
	return out
}

func specTemplate(module string) string {
	return fmt.Sprintf(`# %s SPEC

## 1. Summary

Generated Foundation module skeleton.

## 2. Goals

- Provide a traceable implementation plan.

## 3. Functional Requirements

| ID | Requirement | WHEN | THEN |
| --- | --- | --- | --- |
| FR-001 | bootstrap | WHEN the module is generated | THEN required documentation and tasks exist |

## 4. Acceptance Criteria

| AC ID | FR Ref | Criterion |
| --- | --- | --- |
| AC-001 | FR-001 | Generated module contains SPEC.md, TRACEABILITY.md, goal.md, tasks/, and IMPLEMENTATION-PLAN.md |

## 5. Tests

| TC ID | Covers | Command |
| --- | --- | --- |
| TC-001 | FR-001 / AC-001 | xlib-harness check . --profile full |

## 6. Boundaries

Only stdlib and approved Foundation dependencies are allowed.
`, module)
}

func traceTemplate(module string) string {
	return fmt.Sprintf(`# %s TRACEABILITY

| FR ID | AC ID | TC ID | Status |
| --- | --- | --- | --- |
| FR-001 | AC-001 | TC-001 | PASS |
`, module)
}
