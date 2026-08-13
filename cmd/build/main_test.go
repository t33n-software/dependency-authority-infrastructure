package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func restoreSeams(t *testing.T) {
	t.Helper()
	originalExit := exitProcess
	originalArgs := commandArgs
	originalRun := runExternalCommand
	originalFindGo := findGoFiles
	originalFindHcl := findHclRoots
	originalRead := readSource
	originalFormat := formatSource
	originalCreate := createDirectory
	originalCurrent := currentDirectory
	originalLocate := locateExecutable
	t.Cleanup(func() {
		exitProcess = originalExit
		commandArgs = originalArgs
		runExternalCommand = originalRun
		findGoFiles = originalFindGo
		findHclRoots = originalFindHcl
		readSource = originalRead
		formatSource = originalFormat
		createDirectory = originalCreate
		currentDirectory = originalCurrent
		locateExecutable = originalLocate
	})
}

const validTofuVersionOutput = "OpenTofu v" + requiredOpenTofuVersion + "\non windows_amd64\n"

func fakeRunnerOK() commandRunner {
	return func(_ context.Context, _ []string, _ string, executable string, arguments ...string) ([]byte, error) {
		if executable == "tofu" && len(arguments) == 1 && arguments[0] == "version" {
			return []byte(validTofuVersionOutput), nil
		}
		return nil, nil
	}
}

func fakeFinderOK(files []string) goFileFinder {
	return func(string) ([]string, error) {
		return files, nil
	}
}

func fakeHclRootsOK(roots []string) hclRootFinder {
	return func(string) ([]string, error) {
		return roots, nil
	}
}

func fakeReaderOK() sourceReader {
	return func(string) ([]byte, error) {
		return []byte("package main\n"), nil
	}
}

func fakeFormatterIdentity() sourceFormatter {
	return func(source []byte) ([]byte, error) {
		return source, nil
	}
}

func fakeCreatorOK() directoryCreator {
	return func(string, os.FileMode) error {
		return nil
	}
}

func fakeCurrentDirectoryOK() workingDirectoryGetter {
	return func() (string, error) {
		return "C:\\repo", nil
	}
}

func fakeLocatorOK() executableLocator {
	return func(name string) (string, error) {
		return name, nil
	}
}

func TestMainExitsWithRunResult(t *testing.T) {
	restoreSeams(t)
	exitCode := -1
	exitProcess = func(code int) { exitCode = code }
	commandArgs = []string{"build"}
	runExternalCommand = fakeRunnerOK()
	findGoFiles = fakeFinderOK(nil)
	findHclRoots = fakeHclRootsOK([]string{"modules/example"})
	readSource = fakeReaderOK()
	formatSource = fakeFormatterIdentity()
	createDirectory = fakeCreatorOK()
	currentDirectory = fakeCurrentDirectoryOK()
	locateExecutable = fakeLocatorOK()

	main()

	if exitCode != 0 {
		t.Fatalf("main() exit code = %d, want 0", exitCode)
	}
}

func TestRunRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"unexpected"}, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 2 {
		t.Fatalf("run() = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage: build") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestRunWithNilContextUsesBackground(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example"}), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr = %q", code, stderr.String())
	}
}

func TestRunStopsWhenFormattingFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingFinder := func(string) ([]string, error) {
		return nil, errors.New("walk failure")
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), failingFinder, fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "list Go files") {
		t.Fatalf("stderr = %q, want list error", stderr.String())
	}
}

func TestRunStopsWhenSourceQualityFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingRunner := func(context.Context, []string, string, string, ...string) ([]byte, error) {
		return nil, errors.New("step failure")
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		failingRunner, fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "verify module checksums") {
		t.Fatalf("stderr = %q, want first step failure", stderr.String())
	}
}

func TestRunStopsWhenTofuIsMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingLocator := func(string) (string, error) {
		return "", errors.New("executable not found")
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), failingLocator)
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "locate tofu executable") {
		t.Fatalf("stderr = %q, want locator error", stderr.String())
	}
}

func TestRunStopsWhenTofuVersionReadFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingRunner := func(_ context.Context, _ []string, _ string, executable string, _ ...string) ([]byte, error) {
		if executable == "tofu" {
			return nil, errors.New("tofu failure")
		}
		return nil, nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		failingRunner, fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "read OpenTofu version") {
		t.Fatalf("stderr = %q, want version error", stderr.String())
	}
}

func TestRunStopsWhenTofuVersionMismatches(t *testing.T) {
	var stdout, stderr bytes.Buffer
	mismatchRunner := func(_ context.Context, _ []string, _ string, executable string, arguments ...string) ([]byte, error) {
		if executable == "tofu" && len(arguments) == 1 && arguments[0] == "version" {
			return []byte("OpenTofu v0.0.1\n"), nil
		}
		return nil, nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		mismatchRunner, fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "OpenTofu version") {
		t.Fatalf("stderr = %q, want version mismatch", stderr.String())
	}
}

func TestRunStopsWhenTofuFormattingFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingRunner := func(_ context.Context, _ []string, _ string, executable string, arguments ...string) ([]byte, error) {
		if executable == "tofu" && len(arguments) == 1 && arguments[0] == "version" {
			return []byte(validTofuVersionOutput), nil
		}
		if executable == "tofu" && len(arguments) > 0 && arguments[0] == "fmt" {
			return nil, errors.New("fmt failure")
		}
		return nil, nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		failingRunner, fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "check OpenTofu formatting") {
		t.Fatalf("stderr = %q, want formatting failure", stderr.String())
	}
}

func TestRunStopsWhenHclRootListingFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingRoots := func(string) ([]string, error) {
		return nil, errors.New("walk failure")
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), failingRoots, fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "list OpenTofu roots") {
		t.Fatalf("stderr = %q, want root listing error", stderr.String())
	}
}

func TestRunStopsWhenNoHclRoots(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK(nil), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no OpenTofu roots found") {
		t.Fatalf("stderr = %q, want missing roots error", stderr.String())
	}
}

func TestRunStopsWhenWorkingDirectoryResolutionFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingDirectory := func() (string, error) {
		return "", errors.New("getwd failure")
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example"}), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), failingDirectory, fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "resolve working directory") {
		t.Fatalf("stderr = %q, want working directory error", stderr.String())
	}
}

func TestRunStopsWhenPluginCacheCreationFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingCreator := func(path string, _ os.FileMode) error {
		if path == tofuPluginCacheDirectory {
			return errors.New("mkdir failure")
		}
		return nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example"}), fakeReaderOK(), fakeFormatterIdentity(), failingCreator, fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "create tofu plugin cache directory") {
		t.Fatalf("stderr = %q, want plugin cache mkdir error", stderr.String())
	}
}

func TestRunStopsWhenTofuRootStepFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingRunner := func(_ context.Context, _ []string, _ string, executable string, arguments ...string) ([]byte, error) {
		if executable == "tofu" && len(arguments) == 1 && arguments[0] == "version" {
			return []byte(validTofuVersionOutput), nil
		}
		if executable == "tofu" && len(arguments) > 0 && arguments[0] == "init" {
			return nil, errors.New("init failure")
		}
		return nil, nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		failingRunner, fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example"}), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "init OpenTofu root") {
		t.Fatalf("stderr = %q, want init failure", stderr.String())
	}
}

func TestRunStopsWhenBuildDirectoryCreationFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingCreator := func(path string, _ os.FileMode) error {
		if path == linuxBuildDirectory {
			return errors.New("mkdir failure")
		}
		return nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		fakeRunnerOK(), fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example"}), fakeReaderOK(), fakeFormatterIdentity(), failingCreator, fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "create build directory") {
		t.Fatalf("stderr = %q, want build directory error", stderr.String())
	}
}

func TestRunStopsWhenLinuxBuildFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	failingRunner := func(_ context.Context, _ []string, _ string, executable string, arguments ...string) ([]byte, error) {
		if executable == "tofu" && len(arguments) == 1 && arguments[0] == "version" {
			return []byte(validTofuVersionOutput), nil
		}
		if executable == "go" && len(arguments) > 0 && arguments[0] == "build" {
			return nil, errors.New("linux build failure")
		}
		return nil, nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		failingRunner, fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example"}), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 1 {
		t.Fatalf("run() = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "linux build failure") {
		t.Fatalf("stderr = %q, want linux build failure", stderr.String())
	}
}

func TestRunSuccessExecutesEveryGate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	executed := make([]string, 0)
	recorder := func(_ context.Context, environment []string, directory string, executable string, arguments ...string) ([]byte, error) {
		call := executable + " " + strings.Join(arguments, " ")
		executed = append(executed, directory+"|"+strings.Join(environment, ",")+"|"+call)
		if call == "tofu version" {
			return []byte(validTofuVersionOutput), nil
		}
		return nil, nil
	}
	code := run(context.Background(), nil, &stdout, &stderr,
		recorder, fakeFinderOK(nil), fakeHclRootsOK([]string{"modules/example", "stacks/dep-intake"}), fakeReaderOK(), fakeFormatterIdentity(), fakeCreatorOK(), fakeCurrentDirectoryOK(), fakeLocatorOK())
	if code != 0 {
		t.Fatalf("run() = %d, want 0; stderr = %q", code, stderr.String())
	}
	joined := strings.Join(executed, "\n")
	// The production code builds the cache path with filepath.Join, so the path
	// separator follows the host platform; derive the expectation the same way.
	tofuRootEnvironmentPrefix := "OPENTOFU_ENFORCE_GPG_VALIDATION=true,TF_IN_AUTOMATION=true,TF_INPUT=false,TF_PLUGIN_CACHE_DIR=" + filepath.Join("C:\\repo", tofuPluginCacheDirectory)
	for _, required := range []string{
		"go mod verify",
		"go mod tidy -diff",
		"go test -mod=readonly ./...",
		"go run -mod=readonly ./cmd/check-coverage",
		"go test -mod=readonly -race ./...",
		"go vet ./...",
		"tofu version",
		"tofu fmt -check -recursive",
		"modules/example|" + tofuRootEnvironmentPrefix + "|tofu init -backend=false -input=false -no-color",
		"modules/example|" + tofuRootEnvironmentPrefix + "|tofu validate -no-color",
		"stacks/dep-intake|" + tofuRootEnvironmentPrefix + "|tofu init -backend=false -input=false -no-color",
		"stacks/dep-intake|" + tofuRootEnvironmentPrefix + "|tofu validate -no-color",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("executed steps do not contain %q:\n%s", required, joined)
		}
	}
	for _, gate := range gatePackages {
		build := "go build -mod=readonly -trimpath -o " + linuxBuildDirectory + "/" + gate + " ./cmd/" + gate
		if !strings.Contains(joined, build) {
			t.Fatalf("executed steps do not contain %q:\n%s", build, joined)
		}
		provenance := "go version -m " + linuxBuildDirectory + "/" + gate
		if !strings.Contains(joined, provenance) {
			t.Fatalf("executed steps do not contain %q:\n%s", provenance, joined)
		}
	}
	if !strings.Contains(stdout.String(), "completed successfully") {
		t.Fatalf("stdout = %q, want success message", stdout.String())
	}
}

func TestSourceQualitySteps(t *testing.T) {
	steps := sourceQualitySteps()
	if len(steps) != 6 {
		t.Fatalf("sourceQualitySteps() = %d steps, want 6", len(steps))
	}
	joined := make([]string, 0, len(steps))
	for _, step := range steps {
		joined = append(joined, step.executable+" "+strings.Join(step.arguments, " "))
	}
	for _, required := range []string{
		"go mod verify",
		"go mod tidy -diff",
		"go test -mod=readonly ./...",
		"go run -mod=readonly ./cmd/check-coverage",
		"go test -mod=readonly -race ./...",
		"go vet ./...",
	} {
		if !strings.Contains(strings.Join(joined, "\n"), required) {
			t.Fatalf("sourceQualitySteps() does not contain %q", required)
		}
	}
}

func TestTofuFormatStep(t *testing.T) {
	step := tofuFormatStep()
	if step.executable != "tofu" {
		t.Fatalf("tofuFormatStep().executable = %q, want tofu", step.executable)
	}
	if strings.Join(step.arguments, " ") != "fmt -check -recursive" {
		t.Fatalf("tofuFormatStep().arguments = %v", step.arguments)
	}
	if step.directory != "." {
		t.Fatalf("tofuFormatStep().directory = %q, want .", step.directory)
	}
	if len(step.environment) == 0 {
		t.Fatal("tofuFormatStep().environment is empty")
	}
}

func TestTofuRootSteps(t *testing.T) {
	steps := tofuRootSteps([]string{"modules/a", "modules/b"}, "C:\\repo\\.build\\cache")
	if len(steps) != 4 {
		t.Fatalf("tofuRootSteps() = %d steps, want 4", len(steps))
	}
	if steps[0].name != "init OpenTofu root modules/a" || steps[0].directory != "modules/a" {
		t.Fatalf("steps[0] = %+v", steps[0])
	}
	if steps[1].name != "validate OpenTofu root modules/a" || steps[1].directory != "modules/a" {
		t.Fatalf("steps[1] = %+v", steps[1])
	}
	if strings.Join(steps[0].arguments, " ") != "init -backend=false -input=false -no-color" {
		t.Fatalf("steps[0].arguments = %v", steps[0].arguments)
	}
	if strings.Join(steps[1].arguments, " ") != "validate -no-color" {
		t.Fatalf("steps[1].arguments = %v", steps[1].arguments)
	}
	if !strings.Contains(strings.Join(steps[0].environment, "\n"), "TF_PLUGIN_CACHE_DIR=C:\\repo\\.build\\cache") {
		t.Fatalf("steps[0].environment = %v", steps[0].environment)
	}
}

func TestTofuEnvironment(t *testing.T) {
	environment := tofuEnvironment()
	if len(environment) != 3 {
		t.Fatalf("tofuEnvironment() = %d entries, want 3", len(environment))
	}
	joined := strings.Join(environment, "\n")
	for _, required := range []string{
		"OPENTOFU_ENFORCE_GPG_VALIDATION=true",
		"TF_IN_AUTOMATION=true",
		"TF_INPUT=false",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("tofuEnvironment() does not contain %q", required)
		}
	}
}

func TestTofuRootEnvironment(t *testing.T) {
	joined := strings.Join(tofuRootEnvironment("C:\\repo\\.build\\cache"), "\n")
	for _, required := range []string{
		"OPENTOFU_ENFORCE_GPG_VALIDATION=true",
		"TF_IN_AUTOMATION=true",
		"TF_INPUT=false",
		"TF_PLUGIN_CACHE_DIR=C:\\repo\\.build\\cache",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("tofuRootEnvironment() does not contain %q", required)
		}
	}
}

func TestVerifyOpenTofuVersion(t *testing.T) {
	t.Run("read error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		failingRunner := func(context.Context, []string, string, string, ...string) ([]byte, error) {
			return nil, errors.New("boom")
		}
		if verifyOpenTofuVersion(context.Background(), &stdout, &stderr, failingRunner) {
			t.Fatal("verifyOpenTofuVersion() = true, want false")
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		mismatchRunner := func(context.Context, []string, string, string, ...string) ([]byte, error) {
			return []byte("OpenTofu v0.0.1\n"), nil
		}
		if verifyOpenTofuVersion(context.Background(), &stdout, &stderr, mismatchRunner) {
			t.Fatal("verifyOpenTofuVersion() = true, want false")
		}
		if !strings.Contains(stderr.String(), "OpenTofu version") {
			t.Fatalf("stderr = %q, want mismatch message", stderr.String())
		}
	})

	t.Run("match", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if !verifyOpenTofuVersion(context.Background(), &stdout, &stderr, fakeRunnerOK()) {
			t.Fatal("verifyOpenTofuVersion() = false, want true")
		}
	})

	t.Run("crlf normalized", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		crlfRunner := func(context.Context, []string, string, string, ...string) ([]byte, error) {
			return []byte("OpenTofu v" + requiredOpenTofuVersion + "\r\non windows_amd64\r\n"), nil
		}
		if !verifyOpenTofuVersion(context.Background(), &stdout, &stderr, crlfRunner) {
			t.Fatal("verifyOpenTofuVersion() = false, want true")
		}
	})
}

func TestLinuxBuildSteps(t *testing.T) {
	steps := linuxBuildSteps()
	if len(steps) != len(gatePackages)*2 {
		t.Fatalf("linuxBuildSteps() = %d steps, want %d", len(steps), len(gatePackages)*2)
	}
	joined := make([]string, 0, len(steps))
	for _, step := range steps {
		joined = append(joined, strings.Join(step.environment, ",")+"|"+step.executable+" "+strings.Join(step.arguments, " "))
	}
	for _, gate := range gatePackages {
		if !strings.Contains(strings.Join(joined, "\n"), "GOOS=linux") {
			t.Fatalf("linuxBuildSteps() step for %q misses GOOS=linux", gate)
		}
		if !strings.Contains(strings.Join(joined, "\n"), "./cmd/"+gate) {
			t.Fatalf("linuxBuildSteps() misses build for %q", gate)
		}
	}
}

func TestCheckFormatting(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		failingFinder := func(string) ([]string, error) {
			return nil, errors.New("walk failure")
		}
		if checkFormatting(&stdout, &stderr, failingFinder, fakeReaderOK(), fakeFormatterIdentity()) {
			t.Fatal("checkFormatting() = true, want false")
		}
	})

	t.Run("read error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		failingReader := func(string) ([]byte, error) {
			return nil, errors.New("read failure")
		}
		if checkFormatting(&stdout, &stderr, fakeFinderOK([]string{"main.go"}), failingReader, fakeFormatterIdentity()) {
			t.Fatal("checkFormatting() = true, want false")
		}
	})

	t.Run("format error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		failingFormatter := func([]byte) ([]byte, error) {
			return nil, errors.New("format failure")
		}
		if checkFormatting(&stdout, &stderr, fakeFinderOK([]string{"main.go"}), fakeReaderOK(), failingFormatter) {
			t.Fatal("checkFormatting() = true, want false")
		}
	})

	t.Run("unformatted file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		reformatter := func([]byte) ([]byte, error) {
			return []byte("package main\n\n// changed\n"), nil
		}
		if checkFormatting(&stdout, &stderr, fakeFinderOK([]string{"main.go"}), fakeReaderOK(), reformatter) {
			t.Fatal("checkFormatting() = true, want false")
		}
		if !strings.Contains(stderr.String(), "main.go") {
			t.Fatalf("stderr = %q, want file name", stderr.String())
		}
	})

	t.Run("crlf normalized equal", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		crlfReader := func(string) ([]byte, error) {
			return []byte("package main\r\n"), nil
		}
		lfFormatter := func([]byte) ([]byte, error) {
			return []byte("package main\n"), nil
		}
		if !checkFormatting(&stdout, &stderr, fakeFinderOK([]string{"main.go"}), crlfReader, lfFormatter) {
			t.Fatalf("checkFormatting() = false, want true; stderr = %q", stderr.String())
		}
	})

	t.Run("formatted", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if !checkFormatting(&stdout, &stderr, fakeFinderOK([]string{"main.go"}), fakeReaderOK(), fakeFormatterIdentity()) {
			t.Fatal("checkFormatting() = false, want true")
		}
	})
}

func TestRunStepWritesOutputAndReportsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	okStep := step{name: "ok", executable: "go", arguments: []string{"version"}}
	okRunner := func(context.Context, []string, string, string, ...string) ([]byte, error) {
		return []byte("some-output"), nil
	}
	if !runStep(context.Background(), okStep, &stdout, &stderr, okRunner) {
		t.Fatal("runStep() = false, want true")
	}
	if !strings.Contains(stdout.String(), "some-output") {
		t.Fatalf("stdout = %q, want step output", stdout.String())
	}

	failingRunner := func(context.Context, []string, string, string, ...string) ([]byte, error) {
		return nil, errors.New("boom")
	}
	if runStep(context.Background(), okStep, &stdout, &stderr, failingRunner) {
		t.Fatal("runStep() = true, want false")
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("stderr = %q, want error", stderr.String())
	}
}

func TestRunStepsStopsOnFirstFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	calls := 0
	recorder := func(context.Context, []string, string, string, ...string) ([]byte, error) {
		calls++
		return nil, errors.New("boom")
	}
	steps := []step{
		{name: "first", executable: "go"},
		{name: "second", executable: "go"},
	}
	if runSteps(context.Background(), steps, &stdout, &stderr, recorder) {
		t.Fatal("runSteps() = true, want false")
	}
	if calls != 1 {
		t.Fatalf("runSteps() executed %d steps, want 1", calls)
	}
}

func TestGoFilesFindsGoSources(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.go", "b.txt", filepath.Join("sub", "c.go")} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, directory := range []string{".build", "vendor"} {
		path := filepath.Join(root, directory, "skip.go")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := goFiles(root)
	if err != nil {
		t.Fatalf("goFiles() error = %v", err)
	}
	want := []string{
		filepath.Join(root, "a.go"),
		filepath.Join(root, "sub", "c.go"),
	}
	if strings.Join(files, "|") != strings.Join(want, "|") {
		t.Fatalf("goFiles() = %v, want %v", files, want)
	}
}

func TestGoFilesPropagatesWalkError(t *testing.T) {
	if _, err := goFiles(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("goFiles() error = nil, want error")
	}
}

func TestHclRootsFindsTerraformRoots(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"versions.tf",
		"readme.txt",
		filepath.Join("modules", "x", "main.tf"),
		filepath.Join("stacks", "y", "versions.tf"),
		filepath.Join(".terraform", "skip.tf"),
		filepath.Join(".build", "skip.tf"),
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# content\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	roots, err := hclRoots(root)
	if err != nil {
		t.Fatalf("hclRoots() error = %v", err)
	}
	want := []string{
		root,
		filepath.Join(root, "modules", "x"),
		filepath.Join(root, "stacks", "y"),
	}
	if strings.Join(roots, "|") != strings.Join(want, "|") {
		t.Fatalf("hclRoots() = %v, want %v", roots, want)
	}
}

func TestHclRootsPropagatesWalkError(t *testing.T) {
	if _, err := hclRoots(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("hclRoots() error = nil, want error")
	}
}

func TestIgnoredDirectory(t *testing.T) {
	for _, name := range []string{".build", ".git", ".cache", ".terraform", "coverage", "dist", "vendor"} {
		if !ignoredDirectory(name) {
			t.Errorf("ignoredDirectory(%q) = false, want true", name)
		}
	}
	if ignoredDirectory("cmd") {
		t.Error("ignoredDirectory(cmd) = true, want false")
	}
}

func TestNormalizeLineEndings(t *testing.T) {
	if got := normalizeLineEndings([]byte("a\r\nb\r\n")); string(got) != "a\nb\n" {
		t.Fatalf("normalizeLineEndings() = %q, want %q", got, "a\nb\n")
	}
}

func TestRunCommand(t *testing.T) {
	output, err := runCommand(context.Background(), nil, "", "go", "version")
	if err != nil {
		t.Fatalf("runCommand(go version) error = %v", err)
	}
	if !strings.Contains(string(output), "go version") {
		t.Fatalf("runCommand(go version) = %q, want version output", string(output))
	}

	if _, err := runCommand(context.Background(), nil, ".", "go", "version"); err != nil {
		t.Fatalf("runCommand(go version, dot directory) error = %v", err)
	}

	if _, err := runCommand(context.Background(), nil, t.TempDir(), "go", "version"); err != nil {
		t.Fatalf("runCommand(go version, temp directory) error = %v", err)
	}

	if _, err := runCommand(context.Background(), nil, "", "definitely-not-a-real-command-xyz"); err == nil {
		t.Fatal("runCommand(unknown) error = nil, want error")
	}
}
