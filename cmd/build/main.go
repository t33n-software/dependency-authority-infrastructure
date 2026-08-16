// Command build runs the Dependency Authority Infrastructure source-level quality gates.
package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	linuxBuildDirectory      = ".build/bin"
	tofuPluginCacheDirectory = ".build/tofu-plugin-cache"
	requiredOpenTofuVersion  = "1.12.5"
)

// gatePackages lists the Go gate binaries the Linux/AMD64 gate builds.
var gatePackages = []string{"build", "check-coverage"}

type commandRunner func(context.Context, []string, string, string, ...string) ([]byte, error)

type goFileFinder func(string) ([]string, error)

type hclRootFinder func(string) ([]string, error)

type sourceReader func(string) ([]byte, error)

type sourceFormatter func([]byte) ([]byte, error)

type directoryCreator func(string, os.FileMode) error

type workingDirectoryGetter func() (string, error)

type executableLocator func(string) (string, error)

type step struct {
	name        string
	environment []string
	directory   string
	executable  string
	arguments   []string
}

var (
	exitProcess        = os.Exit
	commandArgs        = os.Args
	runExternalCommand = runCommand
	findGoFiles        = goFiles
	findHclRoots       = hclRoots
	readSource         = os.ReadFile
	formatSource       = format.Source
	createDirectory    = os.MkdirAll
	currentDirectory   = os.Getwd
	locateExecutable   = exec.LookPath
)

func main() {
	exitProcess(run(
		context.Background(),
		commandArgs[1:],
		os.Stdout,
		os.Stderr,
		runExternalCommand,
		findGoFiles,
		findHclRoots,
		readSource,
		formatSource,
		createDirectory,
		currentDirectory,
		locateExecutable,
	))
}

func run(
	ctx context.Context,
	arguments []string,
	stdout io.Writer,
	stderr io.Writer,
	execute commandRunner,
	locateGoFiles goFileFinder,
	locateHclRoots hclRootFinder,
	read sourceReader,
	format sourceFormatter,
	makeDirectory directoryCreator,
	currentDir workingDirectoryGetter,
	locate executableLocator,
) int {
	if len(arguments) != 0 {
		fmt.Fprintln(stderr, "usage: build")
		return 2
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !checkFormatting(stdout, stderr, locateGoFiles, read, format) {
		return 1
	}
	if !runSteps(ctx, sourceQualitySteps(), stdout, stderr, execute) {
		return 1
	}
	if _, err := locate("tofu"); err != nil {
		fmt.Fprintln(stderr, "locate tofu executable:", err)
		return 1
	}
	if !verifyOpenTofuVersion(ctx, stdout, stderr, execute) {
		return 1
	}
	if !runSteps(ctx, []step{tofuFormatStep()}, stdout, stderr, execute) {
		return 1
	}
	roots, err := locateHclRoots(".")
	if err != nil {
		fmt.Fprintln(stderr, "list OpenTofu roots:", err)
		return 1
	}
	if len(roots) == 0 {
		fmt.Fprintln(stderr, "no OpenTofu roots found")
		return 1
	}
	if err := makeDirectory(tofuPluginCacheDirectory, 0o755); err != nil {
		fmt.Fprintln(stderr, "create tofu plugin cache directory:", err)
		return 1
	}
	workingDirectory, err := currentDir()
	if err != nil {
		fmt.Fprintln(stderr, "resolve working directory:", err)
		return 1
	}
	if !runSteps(ctx, tofuRootSteps(roots, filepath.Join(workingDirectory, tofuPluginCacheDirectory)), stdout, stderr, execute) {
		return 1
	}
	if err := makeDirectory(linuxBuildDirectory, 0o755); err != nil {
		fmt.Fprintln(stderr, "create build directory:", err)
		return 1
	}
	if !runSteps(ctx, linuxBuildSteps(), stdout, stderr, execute) {
		return 1
	}

	fmt.Fprintln(stdout, "Dependency Authority Infrastructure source-level build completed successfully.")
	return 0
}

func sourceQualitySteps() []step {
	return []step{
		{
			name:       "verify module checksums",
			executable: "go",
			arguments:  []string{"mod", "verify"},
		},
		{
			name:       "verify module metadata",
			executable: "go",
			arguments:  []string{"mod", "tidy", "-diff"},
		},
		{
			name:       "download build tool dependencies",
			executable: "go",
			arguments:  []string{"-C", "tools", "mod", "download"},
		},
		{
			name:       "verify build tool dependencies",
			executable: "go",
			arguments:  []string{"-C", "tools", "mod", "verify"},
		},
		{
			name:       "verify build tool metadata",
			executable: "go",
			arguments:  []string{"-C", "tools", "mod", "tidy", "-diff"},
		},
		{
			name:       "run lint",
			executable: "go",
			arguments:  []string{"tool", "-modfile", "tools/go.mod", "staticcheck", "./..."},
		},
		{
			name:       "run unit tests",
			executable: "go",
			arguments:  []string{"test", "-mod=readonly", "./..."},
		},
		{
			name:       "enforce complete statement coverage",
			executable: "go",
			arguments:  []string{"run", "-mod=readonly", "./cmd/check-coverage"},
		},
		{
			name:       "run race detector",
			executable: "go",
			arguments:  []string{"test", "-mod=readonly", "-race", "./..."},
		},
		{
			name:       "run static analysis",
			executable: "go",
			arguments:  []string{"vet", "./..."},
		},
		{
			name:       "run vulnerability analysis",
			executable: "go",
			arguments:  []string{"tool", "-modfile", "tools/go.mod", "govulncheck", "./..."},
		},
		{
			name:       "validate Lefthook configuration",
			executable: "go",
			arguments:  []string{"tool", "-modfile", "tools/go.mod", "lefthook", "validate"},
		},
	}
}

func tofuFormatStep() step {
	return step{
		name:        "check OpenTofu formatting",
		environment: tofuEnvironment(),
		directory:   ".",
		executable:  "tofu",
		arguments:   []string{"fmt", "-check", "-recursive"},
	}
}

func tofuRootSteps(roots []string, cacheDirectory string) []step {
	steps := make([]step, 0, len(roots)*2)
	for _, root := range roots {
		steps = append(steps,
			step{
				name:        "init OpenTofu root " + root,
				environment: tofuRootEnvironment(cacheDirectory),
				directory:   root,
				executable:  "tofu",
				arguments:   []string{"init", "-backend=false", "-input=false", "-no-color"},
			},
			step{
				name:        "validate OpenTofu root " + root,
				environment: tofuRootEnvironment(cacheDirectory),
				directory:   root,
				executable:  "tofu",
				arguments:   []string{"validate", "-no-color"},
			},
		)
	}
	return steps
}

func tofuEnvironment() []string {
	return []string{
		"OPENTOFU_ENFORCE_GPG_VALIDATION=true",
		"TF_IN_AUTOMATION=true",
		"TF_INPUT=false",
	}
}

func tofuRootEnvironment(cacheDirectory string) []string {
	return append(tofuEnvironment(), "TF_PLUGIN_CACHE_DIR="+cacheDirectory)
}

func verifyOpenTofuVersion(ctx context.Context, stdout io.Writer, stderr io.Writer, execute commandRunner) bool {
	fmt.Fprintln(stdout, "==> verify OpenTofu version")
	output, err := execute(ctx, nil, ".", "tofu", "version")
	if err != nil {
		fmt.Fprintf(stderr, "read OpenTofu version: %v\n", err)
		return false
	}
	firstLine, _, _ := strings.Cut(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	expected := "OpenTofu v" + requiredOpenTofuVersion
	if firstLine != expected {
		fmt.Fprintf(stderr, "OpenTofu version = %q, want %q\n", firstLine, expected)
		return false
	}
	return true
}

func linuxBuildSteps() []step {
	steps := make([]step, 0, len(gatePackages)*2)
	for _, gate := range gatePackages {
		binaryPath := linuxBuildDirectory + "/" + gate
		steps = append(steps,
			step{
				name:        "build Linux AMD64 " + gate,
				environment: []string{"CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64"},
				executable:  "go",
				arguments:   []string{"build", "-mod=readonly", "-trimpath", "-o", binaryPath, "./cmd/" + gate},
			},
			step{
				name:       "record Linux module provenance for " + gate,
				executable: "go",
				arguments:  []string{"version", "-m", binaryPath},
			},
		)
	}
	return steps
}

func runSteps(ctx context.Context, steps []step, stdout io.Writer, stderr io.Writer, execute commandRunner) bool {
	for _, step := range steps {
		if !runStep(ctx, step, stdout, stderr, execute) {
			return false
		}
	}
	return true
}

func runStep(ctx context.Context, step step, stdout io.Writer, stderr io.Writer, execute commandRunner) bool {
	fmt.Fprintln(stdout, "==>", step.name)
	output, err := execute(ctx, step.environment, step.directory, step.executable, step.arguments...)
	if len(output) > 0 {
		_, _ = stdout.Write(output)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", step.name, err)
		return false
	}
	return true
}

func checkFormatting(stdout io.Writer, stderr io.Writer, locateGoFiles goFileFinder, read sourceReader, format sourceFormatter) bool {
	files, err := locateGoFiles(".")
	if err != nil {
		fmt.Fprintln(stderr, "list Go files:", err)
		return false
	}

	fmt.Fprintln(stdout, "==> check Go formatting")
	unformatted := make([]string, 0)
	for _, file := range files {
		source, err := read(file)
		if err != nil {
			fmt.Fprintln(stderr, "read Go source:", err)
			return false
		}
		formatted, err := format(source)
		if err != nil {
			fmt.Fprintln(stderr, "format Go source:", err)
			return false
		}
		if !bytes.Equal(normalizeLineEndings(source), normalizeLineEndings(formatted)) {
			unformatted = append(unformatted, file)
		}
	}
	if len(unformatted) > 0 {
		fmt.Fprintln(stderr, "the following files require gofmt:")
		fmt.Fprintln(stderr, strings.Join(unformatted, "\n"))
		return false
	}
	return true
}

func normalizeLineEndings(source []byte) []byte {
	return bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
}

func goFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func hclRoots(root string) ([]string, error) {
	roots := make(map[string]struct{})
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".tf" {
			roots[filepath.Dir(path)] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	directories := make([]string, 0, len(roots))
	for directory := range roots {
		directories = append(directories, directory)
	}
	sort.Strings(directories)
	return directories, nil
}

func ignoredDirectory(name string) bool {
	switch name {
	case ".build", ".git", ".cache", ".terraform", "coverage", "dist", "vendor":
		return true
	default:
		return false
	}
}

func runCommand(ctx context.Context, environment []string, directory string, executable string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Env = append(os.Environ(), environment...)
	if directory != "" && directory != "." {
		command.Dir = directory
	}
	return command.CombinedOutput()
}
