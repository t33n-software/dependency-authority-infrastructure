package packaging

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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

// bindingManifest mirrors the tenant binding manifest (repo-bindings/v1) for
// the self-consistency proofs of the canonical adoption. The home-side proof
// against the canonical masters is owned by the verify-canonical tool; these
// tests bind the tenant files to the manifest.
type bindingManifest struct {
	Home struct {
		Repository string `json:"repository"`
		SHA        string `json:"sha"`
	} `json:"home"`
	Callers []struct {
		File   string `json:"file"`
		Master string `json:"master"`
		SHA256 string `json:"sha256"`
	} `json:"callers"`
	Files struct {
		Lefthook      fileBinding `json:"lefthook"`
		Gitattributes fileBinding `json:"gitattributes"`
		Gitignore     fileBinding `json:"gitignore"`
		Dependabot    fileBinding `json:"dependabot"`
	} `json:"files"`
	Codeowners struct {
		Path         string `json:"path"`
		DefaultOwner string `json:"defaultOwner"`
	} `json:"codeowners"`
}

type fileBinding struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func readBindingManifest(t *testing.T) bindingManifest {
	t.Helper()
	var manifest bindingManifest
	if err := json.Unmarshal([]byte(readRepositoryFile(t, "repo-bindings.json")), &manifest); err != nil {
		t.Fatalf("repo-bindings.json is not valid JSON: %v", err)
	}
	if manifest.Home.Repository != "t33n-software/repository-governance" {
		t.Fatalf("the manifest binds home %q", manifest.Home.Repository)
	}
	return manifest
}

// hashRepositoryFile hashes the LF-normalized repository file; the canonical
// .gitattributes makes the checkout LF, and the normalization keeps the
// derivation tolerant as the second line of defense.
func hashRepositoryFile(t *testing.T, path string) string {
	t.Helper()
	normalized := strings.ReplaceAll(readRepositoryFile(t, path), "\r\n", "\n")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func TestCanonicalCallersMatchTheBindingManifest(t *testing.T) {
	manifest := readBindingManifest(t)
	want := map[string]string{
		".github/workflows/ci.yml":                    "hosting-platforms/github/workflows/callers/go/ci.yml",
		".github/workflows/codeql.yml":                "hosting-platforms/github/workflows/callers/go/codeql.yml",
		".github/workflows/dependency-review.yml":     "hosting-platforms/github/workflows/callers/go/dependency-review.yml",
		".github/workflows/canonical-conformance.yml": "hosting-platforms/github/workflows/callers/go/canonical-conformance.yml",
	}
	if len(manifest.Callers) != len(want) {
		t.Fatalf("the manifest carries %d callers, want %d", len(manifest.Callers), len(want))
	}
	for _, caller := range manifest.Callers {
		master, found := want[caller.File]
		if !found {
			t.Fatalf("the manifest carries an unexpected caller %q", caller.File)
		}
		if caller.Master != master {
			t.Fatalf("caller %q binds master %q, want %q", caller.File, caller.Master, master)
		}
		if hash := hashRepositoryFile(t, caller.File); hash != caller.SHA256 {
			t.Fatalf("the tenant caller %s hashes to %s, want the bound %s", caller.File, hash, caller.SHA256)
		}
		content := readRepositoryFile(t, caller.File)
		if !strings.Contains(content, "uses: "+manifest.Home.Repository+"/.github/workflows/reusable-") {
			t.Fatalf("the tenant caller %s does not reference a home payload", caller.File)
		}
		if !strings.Contains(content, "@"+manifest.Home.SHA) {
			t.Fatalf("the tenant caller %s does not pin the bound home SHA", caller.File)
		}
		if !strings.Contains(content, `branches: [main, develop, "release/**", "support/**"]`) {
			t.Fatalf("the tenant caller %s does not cover every shared line", caller.File)
		}
	}
}

func TestCanonicalFileFamilyMatchesTheBindingManifest(t *testing.T) {
	manifest := readBindingManifest(t)
	for _, topic := range []fileBinding{
		manifest.Files.Lefthook,
		manifest.Files.Gitattributes,
		manifest.Files.Dependabot,
	} {
		if hash := hashRepositoryFile(t, topic.Path); hash != topic.SHA256 {
			t.Fatalf("the canonical file %s hashes to %s, want the bound %s", topic.Path, hash, topic.SHA256)
		}
	}
	// The gitignore topic is prefix-mode in the home verifier: the canonical
	// core is a verbatim prefix and project additions live below the mark.
	gitignore := readRepositoryFile(t, manifest.Files.Gitignore.Path)
	canonicalCore := "# Local build and test outputs.\n/.build/\n/dist/\n/coverage/\n/.cache/\n*.coverprofile\n*.test\n*.out\n*.cov\n\n# -- project additions below this line --\n"
	if !strings.HasPrefix(gitignore, canonicalCore) {
		t.Fatal("the gitignore does not carry the canonical core as a verbatim prefix with the project-block mark")
	}
	for _, preserved := range []string{
		"**/.terraform/",
		"*.tfstate",
		"*.tfvars",
		"modules/**/.terraform.lock.hcl",
		"policy-bindings/.terraform.lock.hcl",
	} {
		if !strings.Contains(gitignore, preserved) {
			t.Fatalf("the gitignore does not preserve the project pattern %q below the mark", preserved)
		}
	}

	codeowners := readRepositoryFile(t, manifest.Codeowners.Path)
	if !strings.Contains(codeowners, "* "+manifest.Codeowners.DefaultOwner) {
		t.Fatalf("the ownership file does not bind the default owner %q", manifest.Codeowners.DefaultOwner)
	}
}

func TestConformanceWorkflowBindsTheVerifier(t *testing.T) {
	manifest := readBindingManifest(t)
	content := readRepositoryFile(t, ".github/workflows/canonical-conformance.yml")
	for _, required := range []string{
		"permissions: {}",
		"name: Canonical conformance",
		"uses: " + manifest.Home.Repository + "/.github/workflows/reusable-canonical-conformance.yml@" + manifest.Home.SHA,
		`branches: [main, develop, "release/**", "support/**"]`,
	} {
		if !strings.Contains(content, required) {
			t.Fatalf("the canonical conformance workflow does not contain %q", required)
		}
	}
}

func TestCapabilityPackDeclarationBindsTheOpenTofuGates(t *testing.T) {
	quality := readRepositoryFile(t, "git-governance.quality.json")
	for _, required := range []string{
		`"schemaVersion": 4`,
		`"extends"`,
		`"opentofu@1"`,
	} {
		if !strings.Contains(quality, required) {
			t.Fatalf("git-governance.quality.json does not contain %q", required)
		}
	}

	// The pack contract in the shared-kernel registry is the single OpenTofu
	// contract; the duplicated repository-local convention document is removed.
	if _, err := os.Stat(repositoryPath("docs", "conventions", "infrastructure-as-code", "OPENTOFU-ENGINE-CONVENTION.md")); !os.IsNotExist(err) {
		t.Fatal("the duplicated OpenTofu convention document must not exist; the pack contract is the single contract")
	}

	// The canonical CI caller carries no repository-local OpenTofu setup: the
	// pack provisions the engine through its digest- and signature-bound
	// recipe in the constant provisioning seam of the payload.
	ci := readRepositoryFile(t, ".github/workflows/ci.yml")
	for _, forbidden := range []string{"setup-opentofu", "tofu_version", "OPENTOFU_ENFORCE_GPG_VALIDATION"} {
		if strings.Contains(ci, forbidden) {
			t.Fatalf("the canonical CI caller carries the repository-local OpenTofu setup %q; provisioning is pack-owned", forbidden)
		}
	}

	// The OpenTofu gates are pack-owned and run in the canonical quality lane;
	// no repo-local gate chain copy exists that could carry them.
	for _, chainCopy := range []string{"cmd/build", "cmd/check-coverage"} {
		if _, err := os.Stat(repositoryPath(filepath.FromSlash(chainCopy))); !os.IsNotExist(err) {
			t.Fatalf("the repo-local gate chain copy %s must not exist; the gates are pack-owned or canonical", chainCopy)
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
	// The governed adoption surface is exempt from the organization-coordinate
	// scan: the canonical callers and the conformance lane reference the home
	// coordinate, the binding manifest records the home pin, lefthook.yml names
	// the governed Git toolchain binary, the rule-sets conventions README names
	// the canonical organization source of truth for the GitHub rule-sets,
	// TRACEABILITY.md records this repository's own governed decisions, and the
	// license-hub onboarding values name this repository's own canonical source
	// coordinate under the digest-locked, byte-verified render contract — source
	// and tool references, not organization or tenant bindings of this core.
	governedReferenceExempt := []string{
		".github/workflows/ci.yml",
		".github/workflows/codeql.yml",
		".github/workflows/dependency-review.yml",
		".github/workflows/canonical-conformance.yml",
		"repo-bindings.json",
		"docs/conventions/hosting-plattform/github/rule-sets/README.md",
		"docs/TRACEABILITY.md",
		"lefthook.yml",
		"license.values.json",
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
		`"schemaVersion": 4`,
		`"language": "go"`,
		`"version": "1.26.6"`,
		`"extends": ["opentofu@1"]`,
		"dependency-authority-infrastructure-source-quality",
	} {
		if !strings.Contains(quality, required) {
			t.Fatalf("git-governance.quality.json does not contain %q", required)
		}
	}

	var qualityConfig struct {
		Gates []struct {
			Name    string   `json:"name"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"gates"`
	}
	if err := json.Unmarshal([]byte(quality), &qualityConfig); err != nil {
		t.Fatalf("git-governance.quality.json is not valid JSON: %v", err)
	}
	if len(qualityConfig.Gates) != 1 {
		t.Fatalf("git-governance.quality.json carries %d gates, want exactly the canonical gate chain", len(qualityConfig.Gates))
	}
	if qualityConfig.Gates[0].Name != "dependency-authority-infrastructure-source-quality" ||
		qualityConfig.Gates[0].Command != "go" ||
		!slices.Equal(qualityConfig.Gates[0].Args, []string{"tool", "-modfile", "tools/go.mod", "quality-gate"}) {
		t.Fatal("the gate does not invoke the canonical gate chain through the tooling module pin")
	}
	for _, forbidden := range []string{`"./cmd/build"`, `"./cmd/check-coverage"`, `"defaults"`, `"project"`} {
		if strings.Contains(quality, forbidden) {
			t.Fatalf("git-governance.quality.json still contains %s", forbidden)
		}
	}
	for _, chainCopy := range []string{"cmd/build", "cmd/check-coverage"} {
		if _, err := os.Stat(repositoryPath(filepath.FromSlash(chainCopy))); !os.IsNotExist(err) {
			t.Fatalf("the repo-local gate chain copy %s must not exist", chainCopy)
		}
	}

	lefthook := readRepositoryFile(t, "lefthook.yml")
	if !strings.Contains(lefthook, "git-governance --interactive never validate pre-push --remote") {
		t.Fatal("lefthook.yml does not bind the canonical pre-push validation")
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
		"github.com/t33n-software/go-quality-authority/cmd/quality-gate",
		"github.com/t33n-software/go-quality-authority/cmd/check-coverage",
		"github.com/t33n-software/repository-governance/cmd/verify-canonical",
		"github.com/t33n-software/supply-chain-governance",
	} {
		if !strings.Contains(toolsMod, required) {
			t.Fatalf("tools/go.mod does not contain %q", required)
		}
	}
	if _, err := os.Stat(repositoryPath("tools", "go.sum")); err != nil {
		t.Fatalf("tools/go.sum is missing: %v", err)
	}

	manifest := readBindingManifest(t)
	for _, caller := range []string{"ci.yml", "codeql.yml"} {
		content := readRepositoryFile(t, ".github/workflows/"+caller)
		if !strings.Contains(content, "uses: "+manifest.Home.Repository+"/.github/workflows/reusable-") {
			t.Fatalf("the caller %s does not reference a home payload", caller)
		}
	}

	lefthook := readRepositoryFile(t, "lefthook.yml")
	for _, required := range []string{
		"commit-msg:",
		`git-governance --interactive never commit validate --message-file "{1}"`,
		"pre-push:",
		`git-governance --interactive never validate pre-push --remote "{1}"`,
	} {
		if !strings.Contains(lefthook, required) {
			t.Fatalf("lefthook.yml does not contain %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-5") {
		t.Fatal("TRACEABILITY.md does not contain DAI-5")
	}
}

func TestArtifactRegistryModuleBindsTheDockerWorkloadClass(t *testing.T) {
	variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "variables.tf")))
	for _, required := range []string{
		`contains(["GO", "NPM", "PYTHON", "GENERIC", "DOCKER"], var.format)`,
		`var.format != "DOCKER" || var.mode == "STANDARD_REPOSITORY"`,
		`var.remote_upstream == null || !contains(["GENERIC", "DOCKER"], var.format)`,
	} {
		if !strings.Contains(variables, required) {
			t.Fatalf("modules/artifact-registry/variables.tf does not bind %q", required)
		}
	}

	readme := readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "README.md"))
	if !strings.Contains(readme, "DOCKER") {
		t.Fatal("the artifact-registry module README does not document the DOCKER workload image class")
	}
}

func TestControlStackDeclaresTheWorkloadImageRegistries(t *testing.T) {
	main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	for _, required := range []string{
		`staging = {`,
		`release = {`,
		`module "workload_image_registries" {`,
		`for_each = local.workload_image_registries`,
		`repository_id = "${each.key}-controller-images"`,
		`format = "DOCKER"`,
		`mode = "STANDARD_REPOSITORY"`,
		`boundary = "dependency-authority"`,
		`zone = "control"`,
	} {
		if !strings.Contains(main, required) {
			t.Fatalf("stacks/dep-control/main.tf does not declare %q", required)
		}
	}
	if strings.Contains(main, "remote_upstream") {
		t.Fatal("the control stack must never bind a remote upstream for the workload image registries")
	}
	if strings.Contains(main, "ecosystem") {
		t.Fatal("the workload image registries carry boundary and zone labels only, never an ecosystem label")
	}

	variables := readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tf"))
	if !strings.Contains(variables, `variable "location"`) {
		t.Fatal("stacks/dep-control/variables.tf does not carry the instance-supplied location input")
	}

	outputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "outputs.tf")))
	for _, required := range []string{
		`output "workload_image_repository_ids"`,
		`output "workload_image_registry_uris"`,
	} {
		if !strings.Contains(outputs, required) {
			t.Fatalf("stacks/dep-control/outputs.tf does not export %q", required)
		}
	}

	readme := readRepositoryFile(t, filepath.Join("stacks", "dep-control", "README.md"))
	for _, required := range []string{"staging-controller-images", "release-controller-images"} {
		if !strings.Contains(readme, required) {
			t.Fatalf("the dep-control stack README does not document %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-9") {
		t.Fatal("TRACEABILITY.md does not contain DAI-9")
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
