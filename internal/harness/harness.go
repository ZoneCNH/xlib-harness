package harness

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type GateProfile string

const (
	ProfileSpec     GateProfile = "spec"
	ProfileBoundary GateProfile = "boundary"
	ProfileFull     GateProfile = "full"
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

type GenerateResult struct {
	Module       string   `json:"module"`
	Root         string   `json:"root"`
	FilesCreated []string `json:"files_created"`
	Warnings     []string `json:"warnings,omitempty"`
}

type GenerateOption func(*generateConfig)

type Generator interface {
	Generate(module string, opts ...GenerateOption) (*GenerateResult, error)
}

type HarnessGate interface {
	Check(modulePath string, profile GateProfile) CheckResult
}

type StdlibHarness struct {
	OutputRoot string
	Force      bool
}

type generateConfig struct {
	outputRoot string
	force      bool
}

type options struct {
	stdout io.Writer
	stderr io.Writer
}

var (
	absPath      = filepath.Abs
	statPath     = os.Stat
	mkdirAll     = os.MkdirAll
	writeFile    = os.WriteFile
	readFile     = os.ReadFile
	walkDir      = filepath.WalkDir
	mkdirTemp    = os.MkdirTemp
	removeAll    = os.RemoveAll
	openScanFile = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	generateDocs = Generate
	checkDocs    = Check
)

func WithOutputRoot(root string) GenerateOption {
	return func(cfg *generateConfig) {
		cfg.outputRoot = root
	}
}

func WithForce(force bool) GenerateOption {
	return func(cfg *generateConfig) {
		cfg.force = force
	}
}

func (h StdlibHarness) Generate(module string, opts ...GenerateOption) (*GenerateResult, error) {
	cfg := generateConfig{outputRoot: h.OutputRoot, force: h.Force}
	if cfg.outputRoot == "" {
		cfg.outputRoot = "."
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	files, err := Generate(module, cfg.outputRoot, cfg.force)
	if err != nil {
		return nil, err
	}
	return &GenerateResult{Module: module, Root: cfg.outputRoot, FilesCreated: files}, nil
}

func (h StdlibHarness) Check(modulePath string, profile GateProfile) CheckResult {
	return Check(modulePath, string(profile))
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
	files, err := generateDocs(positional[0], output, force)
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
	res := checkDocs(positional[0], profile)
	if err := writeResult(opts.stdout, res, asJSON); err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
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
	dir, err := mkdirTemp("", "xlib-harness-template-*")
	if err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
	defer removeAll(dir)
	if _, err := generateDocs("template-self", dir, false); err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
	res := checkDocs(filepath.Join(dir, "module", "template-self"), "full")
	if err := writeResult(opts.stdout, res, *asJSON); err != nil {
		fmt.Fprintln(opts.stderr, err)
		return 1
	}
	if !res.Passed {
		return 1
	}
	return 0
}

func writeResult(w io.Writer, res CheckResult, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	var firstErr error
	remember := func(_ int, err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	status := "PASS"
	if !res.Passed {
		status = "FAIL"
	}
	remember(fmt.Fprintf(w, "%s %s (%s)\n", status, res.Module, res.Profile))
	for _, item := range res.Checks {
		itemStatus := "PASS"
		if !item.Passed {
			itemStatus = "FAIL"
		}
		remember(fmt.Fprintf(w, "[%s] %s: %s\n", itemStatus, item.Name, item.Detail))
	}
	remember(fmt.Fprintln(w, res.Summary))
	return firstErr
}

var moduleNameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func Generate(module, outputRoot string, force bool) ([]string, error) {
	if !moduleNameRE.MatchString(module) || strings.Contains(module, "..") || strings.ContainsAny(module, `/\\`) {
		return nil, fmt.Errorf("invalid module name %q: use letters, numbers, dash, or underscore only", module)
	}
	root, err := absPath(outputRoot)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(root, "module", module)
	cleanTarget, err := absPath(target)
	if err != nil {
		return nil, err
	}
	allowedPrefix := filepath.Join(root, "module") + string(os.PathSeparator)
	if !strings.HasPrefix(cleanTarget+string(os.PathSeparator), allowedPrefix) {
		return nil, errors.New("refusing to write outside output module directory")
	}
	entries := map[string]string{
		"README.md":                           readmeTemplate(module),
		"SPEC.md":                             specTemplate(module),
		"TRACEABILITY.md":                     traceTemplate(module),
		"goal.md":                             fmt.Sprintf("# %s Goal\n\nDeliver a compliant Foundation module with documented FR, AC, and TC coverage.\n", module),
		"IMPLEMENTATION-PLAN.md":              fmt.Sprintf("# %s Implementation Plan\n\n1. Confirm SPEC.\n2. Implement tasks.\n3. Verify traceability.\n", module),
		"ACCEPTANCE.md":                       acceptanceTemplate(module),
		"FEATURES.md":                         featuresTemplate(module),
		filepath.Join("tasks", "TASK-001.md"): fmt.Sprintf("# TASK-001 %s bootstrap\n\nImplement and verify the module skeleton.\n", module),
		"Makefile":                            makefileTemplate(),
		filepath.Join(".github", "workflows", "ci.yml"): ciWorkflowTemplate(module),
	}
	paths := make([]string, 0, len(entries))
	for rel := range entries {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	created := make([]string, 0, len(paths)+1)
	if !force {
		for _, rel := range paths {
			dst := filepath.Join(target, rel)
			if _, err := statPath(dst); err == nil {
				return nil, fmt.Errorf("target file exists: %s", dst)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
	}
	if err := mkdirAll(filepath.Join(target, "tasks"), 0o755); err != nil {
		return nil, err
	}
	created = append(created, filepath.ToSlash(filepath.Join("module", module, "tasks")))
	for _, rel := range paths {
		dst := filepath.Join(target, rel)
		if err := mkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if err := writeFile(dst, []byte(entries[rel]), 0o644); err != nil {
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
	if _, err := statPath(modulePath); err != nil {
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
		runCIReferenceChecks(modulePath, add)
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
		info, err := statPath(p)
		add("required-file:"+rel, err == nil && !info.IsDir(), detailForFile(p, err))
	}
	tasksInfo, err := statPath(filepath.Join(modulePath, "tasks"))
	add("required-dir:tasks", err == nil && tasksInfo.IsDir(), detailForFile(filepath.Join(modulePath, "tasks"), err))
	spec, err := readFile(filepath.Join(modulePath, "SPEC.md"))
	if err != nil {
		add("spec-readable", false, err.Error())
		return
	}
	text := string(spec)
	missing := missingTokens(text, []string{"FR-001", "WHEN", "THEN", "Acceptance Criteria", "TC-001"})
	add("spec-fr-ac-tc-structure", len(missing) == 0, missingDetail(missing, "SPEC includes FR/WHEN/THEN/AC/TC markers"))
	headings := countHeadings(text)
	add("spec-section-depth", headings >= 24, fmt.Sprintf("%d markdown headings found", headings))
	sectionGaps := missingSpecSections(text)
	add("spec-23-section-structure", len(sectionGaps) == 0, missingDetail(sectionGaps, "all 23 canonical SPEC sections present"))
	frIssues := functionalRequirementIssues(text)
	add("spec-fr-when-then", len(frIssues) == 0, missingDetail(frIssues, "all FR rows include WHEN and THEN"))
	acIssues := acceptanceCriteriaIssues(text)
	add("spec-ac-verifiability", len(acIssues) == 0, missingDetail(acIssues, "acceptance criteria map to verifiable test cases"))
	tcIssues := testCaseIssues(text)
	add("spec-test-commands", len(tcIssues) == 0, missingDetail(tcIssues, "test cases include runnable commands"))
}

func runBoundaryChecks(modulePath string, add func(string, bool, string)) {
	var issues []string
	walkErr := walkDir(modulePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", filepath.ToSlash(path), err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == "go.mod" {
			data, readErr := readFile(path)
			if readErr != nil {
				issues = append(issues, fmt.Sprintf("%s: %v", filepath.ToSlash(path), readErr))
				return nil
			}
			issues = append(issues, forbiddenModuleRefs(path, string(data))...)
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := readFile(path)
		if readErr != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", filepath.ToSlash(path), readErr))
			return nil
		}
		issues = append(issues, forbiddenImportRefs(path, data)...)
		return nil
	})
	if walkErr != nil {
		issues = append(issues, walkErr.Error())
	}
	add("dependency-boundary", len(issues) == 0, missingDetail(issues, "no forbidden production dependencies found"))
}

func runFormatChecks(modulePath string, add func(string, bool, string)) {
	var issues []string
	walkErr := walkDir(modulePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", filepath.ToSlash(path), err))
			return nil
		}
		if d.IsDir() || !(strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			return nil
		}
		file, openErr := openScanFile(path)
		if openErr != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", path, openErr))
			return nil
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		line := 0
		table := markdownTableState{}
		for scanner.Scan() {
			line++
			txt := scanner.Text()
			issues = append(issues, formatLineIssues(path, line, txt, &table)...)
		}
		if err := scanner.Err(); err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", path, err))
		}
		return nil
	})
	if walkErr != nil {
		issues = append(issues, walkErr.Error())
	}
	add("markdown-format", len(issues) == 0, missingDetail(issues, "markdown/yaml formatting passed"))
}

func runTraceChecks(modulePath string, add func(string, bool, string)) {
	specBytes, specErr := readFile(filepath.Join(modulePath, "SPEC.md"))
	traceBytes, traceErr := readFile(filepath.Join(modulePath, "TRACEABILITY.md"))
	if specErr != nil || traceErr != nil {
		add("traceability-readable", false, fmt.Sprintf("SPEC: %v TRACEABILITY: %v", specErr, traceErr))
		return
	}
	frs := uniqueMatches(`FR-\d{3}`, string(specBytes))
	acs := uniqueMatches(`AC-\d{3}`, string(specBytes))
	tcs := uniqueMatches(`TC-\d{3}`, string(specBytes))
	trace := string(traceBytes)
	var gaps []string
	for _, fr := range frs {
		if !traceRowCloses(trace, fr) {
			gaps = append(gaps, "missing closed "+fr+" -> AC -> TC row in TRACEABILITY.md")
		}
	}
	for _, ac := range acs {
		if !strings.Contains(trace, ac) {
			gaps = append(gaps, "missing "+ac+" in TRACEABILITY.md")
		}
	}
	for _, tc := range tcs {
		if !strings.Contains(trace, tc) {
			gaps = append(gaps, "missing "+tc+" in TRACEABILITY.md")
		}
	}
	add("traceability-chain", len(gaps) == 0, missingDetail(gaps, "FR/AC/TC chain closed"))
}

func runCIReferenceChecks(modulePath string, add func(string, bool, string)) {
	var issues []string
	makefile := filepath.Join(modulePath, "Makefile")
	if info, err := statPath(makefile); err != nil {
		issues = append(issues, "missing Makefile at module root")
	} else if info.IsDir() {
		issues = append(issues, "Makefile path is a directory, not a file")
	} else if data, readErr := readFile(makefile); readErr != nil {
		issues = append(issues, fmt.Sprintf("read Makefile: %v", readErr))
	} else if !makefileHasCITarget(string(data)) {
		issues = append(issues, "Makefile has no ci target")
	}
	workflowCount := 0
	walkErr := walkDir(filepath.Join(modulePath, ".github", "workflows"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", filepath.ToSlash(path), err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			workflowCount++
		}
		return nil
	})
	if walkErr != nil {
		issues = append(issues, fmt.Sprintf(".github/workflows not readable: %v", walkErr))
	} else if workflowCount == 0 {
		issues = append(issues, ".github/workflows has no workflow (.yml/.yaml) files")
	}
	add("ci-reference", len(issues) == 0, missingDetail(issues, "Makefile ci target and GitHub workflow present"))
}

func makefileHasCITarget(data string) bool {
	for _, raw := range strings.Split(data, "\n") {
		line := strings.TrimRight(raw, " \t")
		if strings.HasPrefix(line, "ci:") || strings.HasPrefix(strings.TrimLeft(raw, " \t"), "ci:") {
			return true
		}
	}
	return false
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
	inFence := false
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if !inFence && strings.HasPrefix(scanner.Text(), "#") {
			count++
		}
	}
	return count
}

func missingSpecSections(text string) []string {
	seen := map[int]bool{}
	re := regexp.MustCompile(`^##\s+(\d+)\.`)
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		match := re.FindStringSubmatch(scanner.Text())
		if len(match) == 2 {
			n, err := strconv.Atoi(match[1])
			if err == nil {
				seen[n] = true
			}
		}
	}
	var missing []string
	for i := 1; i <= 23; i++ {
		if !seen[i] {
			missing = append(missing, fmt.Sprintf("missing section %d", i))
		}
	}
	return missing
}

func functionalRequirementIssues(text string) []string {
	var issues []string
	frRows := 0
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(tableRowID(line), "FR-") {
			continue
		}
		frRows++
		if !strings.Contains(line, "WHEN") || !strings.Contains(line, "THEN") {
			issues = append(issues, "FR row missing WHEN/THEN: "+line)
		}
	}
	if frRows == 0 {
		issues = append(issues, "missing FR table rows")
	}
	return issues
}

func acceptanceCriteriaIssues(text string) []string {
	var issues []string
	acRows := 0
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(tableRowID(line), "AC-") {
			continue
		}
		acRows++
		if !regexp.MustCompile(`TC-\d{3}`).MatchString(line) {
			issues = append(issues, "AC row missing TC evidence: "+line)
		}
	}
	if acRows == 0 {
		issues = append(issues, "missing AC table rows")
	}
	return issues
}

func testCaseIssues(text string) []string {
	var issues []string
	tcRows := 0
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(tableRowID(line), "TC-") {
			continue
		}
		tcRows++
		if !strings.Contains(line, "xlib-harness") && !strings.Contains(line, "go test") && !strings.Contains(line, "make ") {
			issues = append(issues, "TC row missing runnable command: "+line)
		}
	}
	if tcRows == 0 {
		issues = append(issues, "missing TC table rows")
	}
	return issues
}

func tableRowID(line string) string {
	if !strings.HasPrefix(strings.TrimSpace(line), "|") {
		return ""
	}
	for _, cell := range strings.Split(line, "|") {
		cell = strings.TrimSpace(cell)
		if cell != "" {
			return cell
		}
	}
	return ""
}

var forbiddenDependencies = []string{"observex", "configx", "resiliencx", "schedulex", "testkitx", "xlib-standard"}

func forbiddenModuleRefs(path, text string) []string {
	var issues []string
	for _, dep := range forbiddenDependencies {
		if strings.Contains(text, "github.com/ZoneCNH/"+dep) {
			issues = append(issues, fmt.Sprintf("%s requires forbidden dependency %s", filepath.ToSlash(path), dep))
		}
	}
	return issues
}

func forbiddenImportRefs(path string, data []byte) []string {
	file, err := parser.ParseFile(token.NewFileSet(), path, data, parser.ImportsOnly)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", filepath.ToSlash(path), err)}
	}
	var issues []string
	for _, imp := range file.Imports {
		importPath, _ := strconv.Unquote(imp.Path.Value)
		for _, dep := range forbiddenDependencies {
			if importPath == "github.com/ZoneCNH/"+dep || strings.HasPrefix(importPath, "github.com/ZoneCNH/"+dep+"/") {
				issues = append(issues, fmt.Sprintf("%s imports forbidden dependency %s", filepath.ToSlash(path), dep))
			}
		}
	}
	return issues
}

type markdownTableState struct {
	columns int
}

func formatLineIssues(path string, line int, txt string, table *markdownTableState) []string {
	var issues []string
	slashPath := filepath.ToSlash(path)
	if strings.TrimRight(txt, " \t") != txt {
		issues = append(issues, fmt.Sprintf("%s:%d trailing whitespace", slashPath, line))
	}
	if strings.Contains(txt, "FORMAT-ISSUE") {
		issues = append(issues, fmt.Sprintf("%s:%d explicit format issue marker", slashPath, line))
	}
	issues = append(issues, markdownLinkIssues(slashPath, line, txt)...)
	if strings.HasPrefix(strings.TrimSpace(txt), "|") {
		cols := strings.Count(txt, "|")
		if table.columns == 0 {
			table.columns = cols
		} else if table.columns != cols {
			issues = append(issues, fmt.Sprintf("%s:%d table column count changed from %d to %d", slashPath, line, table.columns, cols))
		}
	} else if strings.TrimSpace(txt) == "" {
		table.columns = 0
	}
	return issues
}

func markdownLinkIssues(path string, line int, txt string) []string {
	var issues []string
	re := regexp.MustCompile(`\[[^\]]+\]\(([^)]*)\)`)
	for _, match := range re.FindAllStringSubmatch(txt, -1) {
		target := strings.TrimSpace(match[1])
		if target == "" {
			issues = append(issues, fmt.Sprintf("%s:%d empty markdown link target", path, line))
		}
	}
	return issues
}

func traceRowCloses(trace, fr string) bool {
	scanner := bufio.NewScanner(strings.NewReader(trace))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fr) && regexp.MustCompile(`AC-\d{3}`).MatchString(line) && regexp.MustCompile(`TC-\d{3}`).MatchString(line) {
			return true
		}
	}
	return false
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
	sections := []struct {
		title string
		body  string
	}{
		{"Summary", "Generated Foundation module skeleton."},
		{"Goals", "- Provide a traceable implementation plan."},
		{"Non-Goals", "- Avoid adding runtime behavior before the module spec is approved."},
		{"Stakeholders", "- Module owner\n- Foundation maintainers"},
		{"Glossary", "- FR: Functional requirement\n- AC: Acceptance criterion\n- TC: Test case"},
		{"Functional Requirements", "| ID | Requirement | WHEN | THEN |\n| --- | --- | --- | --- |\n| FR-001 | bootstrap | WHEN the module is generated | THEN required documentation and tasks exist |"},
		{"Business Rules", "| ID | Rule |\n| --- | --- |\n| BR-001 | Generated work remains local until reviewed. |"},
		{"Acceptance Criteria", "| AC ID | FR Ref | Criterion |\n| --- | --- | --- |\n| AC-001 | FR-001 | TC-001 proves SPEC.md, TRACEABILITY.md, goal.md, tasks/, and IMPLEMENTATION-PLAN.md exist. |"},
		{"Tests", "| TC ID | Covers | Command |\n| --- | --- | --- |\n| TC-001 | FR-001 / AC-001 | xlib-harness check . --profile full |"},
		{"Traceability", "TRACEABILITY.md must close every FR -> AC -> TC chain."},
		{"Interfaces", "No public code interface is generated by this skeleton."},
		{"Data Model", "No persisted application data is generated by this skeleton."},
		{"Error Handling", "Generation and gate failures must report actionable file-level details."},
		{"Security", "No credentials or private endpoints may be committed."},
		{"Privacy", "No personal or private data is required."},
		{"Performance", "Local validation should complete in under one second for a skeleton module."},
		{"Observability", "Gate output must include pass/fail summaries and itemized checks."},
		{"Operations", "Use local commands first, then CI gates before release."},
		{"CI Gates", "Run go test ./..., go test ./... -race -count=1, go vet ./..., and xlib-harness check."},
		{"Migration", "Existing generated files are overwritten only when --force is explicit."},
		{"Risks", "- Incomplete traceability\n- Accidental cross-module dependencies"},
		{"Open Questions", "- Replace this generated skeleton with the module-specific approved spec."},
		{"Changelog", "- Initial generated skeleton."},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s SPEC\n\n", module)
	for i, section := range sections {
		fmt.Fprintf(&b, "## %d. %s\n\n%s\n\n", i+1, section.title, section.body)
	}
	return b.String()
}

func traceTemplate(module string) string {
	return fmt.Sprintf(`# %s TRACEABILITY

| FR ID | AC ID | TC ID | Status |
| --- | --- | --- | --- |
| FR-001 | AC-001 | TC-001 | PASS |
`, module)
}

func readmeTemplate(module string) string {
	return fmt.Sprintf(`# %s

> Foundation module entry point.

## Overview

%s is a Foundation module. See SPEC.md for requirements, TRACEABILITY.md for the FR/AC/TC matrix, and ACCEPTANCE.md for verification commands.
`, module, module)
}

func acceptanceTemplate(module string) string {
	return fmt.Sprintf(`# %s Acceptance

| AC ID | Requirement | Command | Expected | Status |
| --- | --- | --- | --- | --- |
| AC-001 | Module skeleton is generated and internally consistent | xlib-harness check . --profile full | all gates pass | PASS |
`, module)
}

func featuresTemplate(module string) string {
	return fmt.Sprintf(`# %s Features

| Feature ID | Capability | Status |
| --- | --- | --- |
| FR-001 | Module skeleton generation | Implemented |
`, module)
}

func makefileTemplate() string {
	return `.PHONY: build test vet ci

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

ci: build test vet
`
}

func ciWorkflowTemplate(module string) string {
	return fmt.Sprintf(`name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  validate:
    name: Validate %s
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true
      - run: make ci
`, module)
}
