package packaging

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var moduleNames = []string{
	"artifact-registry",
	"evidence-archive",
	"logging",
	"network",
	"repository-iam",
	"recovery",
	"workload-identity",
}

var stackNames = []string{
	"dep-control",
	"dep-evidence",
	"dep-intake",
	"dep-approved",
	"dep-quarantine",
}

func TestSourceWorkflowsEmitOnlyEstablishedSharedLineChecks(t *testing.T) {
	for _, workflow := range []string{
		".github/workflows/ci.yml",
		".github/workflows/codeql.yml",
		".github/workflows/dependency-review.yml",
	} {
		content := readRepositoryFile(t, workflow)
		for _, forbidden := range []string{"release/**", "support/**"} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s unexpectedly targets %q before the governed release lifecycle exists", workflow, forbidden)
			}
		}
		for _, required := range []string{"main", "develop"} {
			if !strings.Contains(content, required) {
				t.Fatalf("%s does not target %q", workflow, required)
			}
		}
	}

	ci := readRepositoryFile(t, ".github/workflows/ci.yml")
	for _, required := range []string{
		"Quality gates (linux-amd64)",
		"go run -mod=readonly ./cmd/build",
		"actions/checkout@9f698171ed81b15d1823a05fc7211befd50c8ae0",
		"actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c",
		"opentofu/setup-opentofu@a1320f892987e89d278cc92dc5adc984fb93aca4",
		"tofu_version",
		"1.12.5",
		"OPENTOFU_ENFORCE_GPG_VALIDATION",
		"OpenTofu v1.12.5",
	} {
		if !strings.Contains(ci, required) {
			t.Fatalf("CI workflow does not contain %q", required)
		}
	}
	for _, forbidden := range []string{
		"hashicorp/setup-terraform",
		"terraform fmt",
		"terraform validate",
		"terraform init",
		"terraform plan",
		"terraform apply",
	} {
		if strings.Contains(ci, forbidden) {
			t.Fatalf("CI workflow contains forbidden Terraform engine reference %q; the IaC engine is OpenTofu", forbidden)
		}
	}

	dependencyReview := readRepositoryFile(t, ".github/workflows/dependency-review.yml")
	for _, required := range []string{
		"Dependency admission review",
		"fail-on-severity: low",
		"fail-on-scopes: runtime,development,unknown",
		"actions/dependency-review-action@2031cfc080254a8a887f58cffee85186f0e49e48",
	} {
		if !strings.Contains(dependencyReview, required) {
			t.Fatalf("dependency review workflow does not contain %q", required)
		}
	}

	dependabot := readRepositoryFile(t, ".github/dependabot.yml")
	for _, required := range []string{"gomod", "github-actions", "terraform"} {
		if !strings.Contains(dependabot, required) {
			t.Fatalf("dependabot.yml does not cover ecosystem %q", required)
		}
	}
}

func TestOrganizationRulesetAdoptionHasNoLocalLegacyDefinitions(t *testing.T) {
	if _, err := os.Stat(repositoryPath("docs", "hosting-platforms")); !os.IsNotExist(err) {
		t.Fatalf("legacy ruleset location must not exist")
	}

	conventions := readRepositoryFile(t, filepath.Join("docs", "conventions", "hosting-plattform", "github", "rule-sets", "README.md"))
	for _, required := range []string{
		"git-governance",
		"quality-gates=linux-only",
		"~ALL",
	} {
		if !strings.Contains(conventions, required) {
			t.Fatalf("rule-set conventions README does not contain %q", required)
		}
	}
}

func TestGovernanceDocumentationPreservesCoreInstanceAndTenantBoundaries(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs/architecture/ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md",
		"docs/development/VERIFICATION.md",
	} {
		content := strings.ToLower(readRepositoryFile(t, path))
		for _, required := range []string{"core", "instance", "tenant"} {
			if !strings.Contains(content, required) {
				t.Fatalf("%s does not document %q boundary", path, required)
			}
		}
	}

	adr := readRepositoryFile(t, "docs/architecture/ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")
	for _, required := range []string{
		"never contains concrete organization",
		"never contains tenant",
		"control",
		"intake",
		"quarantine",
		"approved",
		"evidence",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR does not contain %q", required)
		}
	}
}

func TestOpenTofuConventionIsDocumented(t *testing.T) {
	content := readRepositoryFile(t, filepath.Join(
		"docs", "conventions", "infrastructure-as-code", "OPENTOFU-ENGINE-CONVENTION.md",
	))
	for _, required := range []string{
		"OpenTofu",
		"MPL 2.0",
		"BUSL",
		"Linux Foundation",
		"client-side state and plan encryption",
		"OPENTOFU_ENFORCE_GPG_VALIDATION=true",
		"Sigstore",
		"`.terraform.lock.hcl`",
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("OpenTofu convention does not contain %q", required)
		}
	}
}

func TestModuleAndStackLayoutIsComplete(t *testing.T) {
	moduleFiles := []string{"main.tf", "variables.tf", "outputs.tf", "versions.tf", "README.md"}
	for _, module := range moduleNames {
		for _, file := range moduleFiles {
			path := repositoryPath("modules", module, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing module file %q: %v", path, err)
			}
		}
	}
	for _, file := range moduleFiles {
		path := repositoryPath("policy-bindings", file)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing policy-bindings file %q: %v", path, err)
		}
	}
	for _, stack := range stackNames {
		for _, file := range append(moduleFiles, ".terraform.lock.hcl") {
			path := repositoryPath("stacks", stack, file)
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("missing stack file %q: %v", path, err)
			}
		}
	}
}

func TestCoreContainsNoConcreteBindings(t *testing.T) {
	forbiddenContent := []string{
		"cybert33n",
		"t33n-software",
		"git-governance",
		"europe-west3",
		"937088974261",
		"1065293691137",
		"1007556997805",
		"346339887743",
		"01c36d",
	}
	// Governed source references are not concrete bindings: the rule-set
	// conventions README names the canonical organization source of truth for
	// the GitHub rule-sets, TRACEABILITY.md records this repository's own
	// migration decisions, and lefthook.yml names the governed Git toolchain
	// binary for commit-msg validation — source and tool references, not
	// organization or tenant bindings of this core.
	governedReferenceExempt := []string{
		"docs/conventions/hosting-plattform/github/rule-sets/README.md",
		"docs/TRACEABILITY.md",
		"lefthook.yml",
	}
	for _, path := range repositoryFiles(t, []string{".tf", ".yml", ".yaml", ".json", ".md"}) {
		slashed := filepath.ToSlash(path)
		exempt := false
		for _, exemptPath := range governedReferenceExempt {
			if strings.HasSuffix(slashed, exemptPath) {
				exempt = true
				break
			}
		}
		if exempt {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		lowered := strings.ToLower(string(content))
		for _, forbidden := range forbiddenContent {
			if strings.Contains(lowered, forbidden) {
				t.Fatalf("%s contains concrete binding %q; the core never carries organization or tenant values", path, forbidden)
			}
		}
	}

	for _, path := range repositoryFiles(t, []string{".tf"}) {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		for _, forbidden := range []string{"?ref=main", "?ref=develop", "latest"} {
			if strings.Contains(string(content), forbidden) {
				t.Fatalf("%s contains mutable reference %q", path, forbidden)
			}
		}
	}
}

func TestOpenTofuPinsAreExactAndConsistent(t *testing.T) {
	for _, root := range append(
		append([]string{"policy-bindings"}, modulePaths()...),
		stackPaths()...,
	) {
		versions := normalizeWhitespace(readRepositoryFile(t, filepath.Join(root, "versions.tf")))
		for _, required := range []string{
			`required_version = "= 1.12.5"`,
			`source = "hashicorp/google"`,
			`version = "= 7.44.0"`,
		} {
			if !strings.Contains(versions, required) {
				t.Fatalf("%s/versions.tf does not contain exact pin %q", root, required)
			}
		}
		for _, forbidden := range []string{"~>", ">="} {
			if strings.Contains(versions, forbidden) {
				t.Fatalf("%s/versions.tf contains non-exact constraint %q", root, forbidden)
			}
		}
	}

	for _, stack := range stackNames {
		lock := readRepositoryFile(t, filepath.Join("stacks", stack, ".terraform.lock.hcl"))
		for _, required := range []string{"hashicorp/google", "7.44.0"} {
			if !strings.Contains(lock, required) {
				t.Fatalf("stacks/%s/.terraform.lock.hcl does not contain %q", stack, required)
			}
		}
	}
}

func TestModuleIdentityAndQualityContract(t *testing.T) {
	goMod := readRepositoryFile(t, "go.mod")
	for _, required := range []string{
		"module github.com/t33n-software/dependency-authority-infrastructure",
		"go 1.26",
		"toolchain go1.26.6",
	} {
		if !strings.Contains(goMod, required) {
			t.Fatalf("go.mod does not contain %q", required)
		}
	}

	quality := readRepositoryFile(t, "git-governance.quality.json")
	for _, required := range []string{
		"dependency-authority-infrastructure-source-quality",
		"./cmd/build",
	} {
		if !strings.Contains(quality, required) {
			t.Fatalf("git-governance.quality.json does not contain %q", required)
		}
	}

	lefthook := readRepositoryFile(t, "lefthook.yml")
	if !strings.Contains(lefthook, "go run -mod=readonly ./cmd/build") {
		t.Fatal("lefthook.yml does not run the source-quality gate")
	}
}

func TestGoToolchainAndBuildToolingContract(t *testing.T) {
	toolsMod := readRepositoryFile(t, filepath.Join("tools", "go.mod"))
	for _, required := range []string{
		"module github.com/t33n-software/dependency-authority-infrastructure/tools",
		"toolchain go1.26.6",
		"github.com/evilmartians/lefthook/v2",
		"golang.org/x/vuln/cmd/govulncheck",
		"honnef.co/go/tools/cmd/staticcheck",
	} {
		if !strings.Contains(toolsMod, required) {
			t.Fatalf("tools/go.mod does not contain %q", required)
		}
	}
	if _, err := os.Stat(repositoryPath("tools", "go.sum")); err != nil {
		t.Fatalf("tools/go.sum is missing: %v", err)
	}

	ci := readRepositoryFile(t, ".github/workflows/ci.yml")
	for _, required := range []string{
		`go-version: "1.26.6"`,
		`test "$(go env GOVERSION)" = "go1.26.6"`,
		"schedule:",
		"cron:",
	} {
		if !strings.Contains(ci, required) {
			t.Fatalf("CI workflow does not contain %q", required)
		}
	}

	codeql := readRepositoryFile(t, ".github/workflows/codeql.yml")
	for _, required := range []string{
		`go-version: "1.26.6"`,
		`test "$(go env GOVERSION)" = "go1.26.6"`,
	} {
		if !strings.Contains(codeql, required) {
			t.Fatalf("CodeQL workflow does not contain %q", required)
		}
	}

	lefthookContent := readRepositoryFile(t, "lefthook.yml")
	for _, required := range []string{
		"commit-msg:",
		`git-governance --interactive never commit validate --message-file "{1}"`,
		"pre-push:",
		"go run -mod=readonly ./cmd/build",
	} {
		if !strings.Contains(lefthookContent, required) {
			t.Fatalf("lefthook.yml does not contain %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-3") {
		t.Fatal("TRACEABILITY.md does not contain DAI-3")
	}
}

func modulePaths() []string {
	paths := make([]string, 0, len(moduleNames))
	for _, module := range moduleNames {
		paths = append(paths, filepath.Join("modules", module))
	}
	return paths
}

func stackPaths() []string {
	paths := make([]string, 0, len(stackNames))
	for _, stack := range stackNames {
		paths = append(paths, filepath.Join("stacks", stack))
	}
	return paths
}

func normalizeWhitespace(content string) string {
	return strings.Join(strings.Fields(content), " ")
}

func repositoryFiles(t *testing.T, extensions []string) []string {
	t.Helper()
	root := repositoryPath()
	matches := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".build", ".terraform", "coverage", "dist", "vendor":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		for _, extension := range extensions {
			if filepath.Ext(path) == extension {
				matches = append(matches, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%q) error = %v", root, err)
	}
	sort.Strings(matches)
	return matches
}

func readRepositoryFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(repositoryPath(filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func repositoryPath(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}
