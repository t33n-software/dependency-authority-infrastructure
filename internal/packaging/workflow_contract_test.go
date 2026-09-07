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
	"cloud-run-job",
	"evidence-archive",
	"forensics-readers",
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

func TestCloudRunJobModuleBindsTheGovernedConsumptionForm(t *testing.T) {
	main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "main.tf")))
	for _, required := range []string{
		`resource "google_cloud_run_v2_job" "this"`,
		`service_account = var.service_account_email`,
		`image = var.image`,
		`vpc_access {`,
		`network_interfaces {`,
		`network = var.network`,
		`subnetwork = var.subnetwork`,
		`egress = "ALL_TRAFFIC"`,
	} {
		if !strings.Contains(main, required) {
			t.Fatalf("modules/cloud-run-job/main.tf does not bind %q", required)
		}
	}

	variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "variables.tf")))
	for _, required := range []string{
		`^dep-[a-z0-9]+(-[a-z0-9]+)*$`,
		`^[^@\\s]+@sha256:[0-9a-f]{64}$`,
		`/release-[^/]+/`,
		`^projects/[^/]+/global/networks/[a-z][a-z0-9-]*$`,
		`^projects/[^/]+/regions/[a-z][a-z0-9-]+/subnetworks/[a-z][a-z0-9-]*$`,
	} {
		if !strings.Contains(variables, required) {
			t.Fatalf("modules/cloud-run-job/variables.tf does not bind the fail-closed rule %q", required)
		}
	}

	// The network origin is a mandatory instance binding: neither the network
	// nor the subnetwork input carries a default.
	for _, name := range []string{"network", "subnetwork"} {
		start := strings.Index(variables, `variable "`+name+`" {`)
		if start < 0 {
			t.Fatalf("modules/cloud-run-job/variables.tf does not carry the mandatory %s input", name)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if strings.Contains(segment, "default") {
			t.Fatalf("modules/cloud-run-job/variables.tf carries a default for %s; the network origin is an instance binding, never a default", name)
		}
	}

	readme := readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "README.md"))
	for _, required := range []string{"planned", "bound", "atomically", "release-class", "Direct VPC egress"} {
		if !strings.Contains(readme, required) {
			t.Fatalf("the cloud-run-job module README does not document %q", required)
		}
	}
}

func TestStacksDeclareTheCompleteWorkloadJobTopology(t *testing.T) {
	// The canonical job matrix: exactly one job per lane operation in its own
	// zone, each bound to the existing zone workload identity of its lane.
	jobs := map[string]map[string]string{
		"dep-intake": {
			"dep-intake-fetch": "fetcher",
		},
		"dep-control": {
			"dep-admission":    "admission",
			"dep-promotion":    "promotion",
			"dep-revalidation": "revalidation",
			"dep-revocation":   "revocation",
		},
		"dep-evidence": {
			"dep-evidence-write": "writer",
			"dep-evidence-audit": "auditor",
		},
	}

	declared := 0
	for stack, bindings := range jobs {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		for _, required := range []string{
			`module "workload_jobs" {`,
			`for_each = local.workload_jobs`,
			`service_account_email = module.workload_identity.service_account_emails[each.value.identity_key]`,
			`image = var.workload_job_images[each.key]`,
		} {
			if !strings.Contains(main, required) {
				t.Fatalf("stacks/%s/main.tf does not declare %q", stack, required)
			}
		}
		for name, identityKey := range bindings {
			if !strings.Contains(main, `"`+name+`" = {`) {
				t.Fatalf("stacks/%s/main.tf does not declare the canonical job %q", stack, name)
			}
			if !strings.Contains(main, `identity_key = "`+identityKey+`"`) {
				t.Fatalf("stacks/%s/main.tf does not bind the job %q to the identity key %q", stack, name, identityKey)
			}
			declared++
		}

		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		start := strings.Index(variables, `variable "workload_job_images" {`)
		if start < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the instance-supplied workload_job_images input", stack)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if strings.Contains(segment, "default") {
			t.Fatalf("stacks/%s/variables.tf carries a default for workload_job_images; the image digest is an instance binding, never a stack default", stack)
		}

		outputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "outputs.tf")))
		if !strings.Contains(outputs, `output "workload_job_ids"`) {
			t.Fatalf("stacks/%s/outputs.tf does not export the workload job IDs", stack)
		}
	}
	if declared != 7 {
		t.Fatalf("the stacks declare %d workload jobs, want the complete canonical topology of 7", declared)
	}

	// Zone purity: the quarantine and approved zones never carry workload jobs.
	for _, stack := range []string{"dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "workload_jobs") {
			t.Fatalf("stacks/%s must never declare workload jobs; the topology is zone-pure", stack)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-10") {
		t.Fatal("TRACEABILITY.md does not contain DAI-10")
	}
}

func TestStacksDeclareTheCanonicalIAMTargetMatrix(t *testing.T) {
	// Intake: the fetcher is the only writer; the matrix readers (canonically
	// the admission and promotion controllers of the control zone) arrive
	// through the instance-wired member input.
	intakeMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-intake", "main.tf")))
	for _, required := range []string{
		`writers = ["serviceAccount:${module.workload_identity.service_account_emails["fetcher"]}"]`,
		`readers = var.additional_reader_members`,
	} {
		if !strings.Contains(intakeMain, required) {
			t.Fatalf("stacks/dep-intake/main.tf does not bind the matrix form %q", required)
		}
	}
	intakeVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-intake", "variables.tf")))
	if !strings.Contains(intakeVariables, `variable "additional_reader_members" {`) {
		t.Fatal("stacks/dep-intake/variables.tf does not carry the additional_reader_members input")
	}

	// Evidence: the writer appends; the matrix writers (canonically the
	// admission, revalidation and revocation controllers of the control zone)
	// and the matrix reader (the approved promoter) arrive through the member
	// inputs.
	evidenceMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-evidence", "main.tf")))
	for _, required := range []string{
		`["serviceAccount:${module.workload_identity.service_account_emails["writer"]}"]`,
		`tolist(var.additional_writer_members)`,
		`tolist(var.additional_auditor_members)`,
	} {
		if !strings.Contains(evidenceMain, required) {
			t.Fatalf("stacks/dep-evidence/main.tf does not bind the matrix form %q", required)
		}
	}
	evidenceVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-evidence", "variables.tf")))
	if !strings.Contains(evidenceVariables, `variable "additional_writer_members" {`) {
		t.Fatal("stacks/dep-evidence/variables.tf does not carry the additional_writer_members input")
	}
	if !strings.Contains(evidenceVariables, "intake fetcher") {
		t.Fatal("stacks/dep-evidence/variables.tf does not name the intake fetcher as a canonical evidence writer; the intake use case writes its candidate records into the evidence repository")
	}

	// Approved: no zone-local workload identity; the promotion and revocation
	// writes and the revalidation read are control-zone members bound through
	// the member inputs.
	approvedVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-approved", "variables.tf")))
	if strings.Contains(approvedVariables, `variable "promoter" {`) {
		t.Fatal("stacks/dep-approved/variables.tf still declares a zone-local promoter identity; the approved promoter is a control-zone identity")
	}
	start := strings.Index(approvedVariables, `variable "identities" {`)
	if start < 0 {
		t.Fatal("stacks/dep-approved/variables.tf does not carry the optional identities input")
	}
	segment := approvedVariables[start:]
	if next := strings.Index(segment, ` variable "`); next > 0 {
		segment = segment[:next]
	}
	if !strings.Contains(segment, "default = {}") {
		t.Fatal("stacks/dep-approved/variables.tf must default identities to the empty map; the matrix binds no zone-local identity")
	}
	for _, required := range []string{
		`variable "promoter_member" {`,
		`variable "revocation_member" {`,
		`variable "revalidation_reader_member" {`,
	} {
		if !strings.Contains(approvedVariables, required) {
			t.Fatalf("stacks/dep-approved/variables.tf does not carry %q", required)
		}
	}

	approvedMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-approved", "main.tf")))
	for _, required := range []string{
		`identities = var.identities`,
		`writers = [var.promoter_member, var.revocation_member]`,
		`readers = concat([var.revalidation_reader_member], tolist(var.consumer_members))`,
	} {
		if !strings.Contains(approvedMain, required) {
			t.Fatalf("stacks/dep-approved/main.tf does not bind the matrix form %q", required)
		}
	}
	if strings.Contains(approvedMain, `service_account_emails["promoter"]`) {
		t.Fatal("stacks/dep-approved/main.tf still references the removed zone-local promoter identity")
	}
	approvedOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-approved", "outputs.tf")))
	if strings.Contains(approvedOutputs, "promoter_service_account_email") {
		t.Fatal("stacks/dep-approved/outputs.tf still exports the removed zone-local promoter identity")
	}

	// Control: every zone lane identity reads the release-class workload image
	// registry (the four control-plane lanes directly, the other zones through
	// the cross-zone member input); no writer on either class; the staging
	// class carries no binding at all.
	controlMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	for _, required := range []string{
		`module "workload_image_registry_iam" {`,
		`repository = module.workload_image_registries["release"].id`,
		`[for lane in keys(var.controllers) : "serviceAccount:${module.workload_identity.service_account_emails[lane]}"]`,
		`tolist(var.cross_zone_workload_reader_members)`,
	} {
		if !strings.Contains(controlMain, required) {
			t.Fatalf("stacks/dep-control/main.tf does not bind the matrix form %q", required)
		}
	}
	if strings.Contains(controlMain, `workload_image_registries["staging"]`) {
		t.Fatal("the staging workload image registry must never carry an IAM binding; it is filled exclusively by the governed producer channel")
	}
	iamStart := strings.Index(controlMain, `module "workload_image_registry_iam" {`)
	if iamStart < 0 {
		t.Fatal("stacks/dep-control/main.tf does not declare the workload image registry IAM module")
	}
	iamSegment := controlMain[iamStart:]
	if next := strings.Index(iamSegment, ` module "`); next > 0 {
		iamSegment = iamSegment[:next]
	}
	if strings.Contains(iamSegment, "writers") {
		t.Fatal("the workload image registry IAM binds a writer; no identity ever receives a writer grant on either workload image registry class")
	}
	controlVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tf")))
	if !strings.Contains(controlVariables, `variable "cross_zone_workload_reader_members" {`) {
		t.Fatal("stacks/dep-control/variables.tf does not carry the cross_zone_workload_reader_members input")
	}

	// The break-glass recovery identity holds no data-plane grant anywhere.
	for _, stack := range stackNames {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "break-glass") {
			t.Fatalf("stacks/%s/main.tf references the break-glass recovery identity; the matrix grants it no data-plane role", stack)
		}
	}

	// The architecture decision record carries the canonical matrix including
	// its exclusions.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"canonical IAM target matrix",
		"dep-intake-fetcher intake writer on *-dependencies-intake and *-dependencies-evidence",
		"dep-admission-controller",
		"dep-approved-promoter",
		"dep-revalidation-controller",
		"dep-revocation-controller",
		"dep-evidence-writer",
		"dep-evidence-auditor",
		"dep-break-glass-recovery",
		"release-controller-images",
		"no data-plane grant",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the canonical IAM target matrix element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-11") {
		t.Fatal("TRACEABILITY.md does not contain DAI-11")
	}
}

func TestStacksDeclareTheInvokeOnlyTriggerRights(t *testing.T) {
	// The canonical trigger identities: exactly one invoke-only trigger
	// identity per lane operation, named after the canonical job.
	triggers := map[string]map[string]string{
		"dep-intake": {
			"dep-intake-fetch": "dep-intake-fetch-trigger",
		},
		"dep-control": {
			"dep-admission":    "dep-admission-trigger",
			"dep-promotion":    "dep-promotion-trigger",
			"dep-revalidation": "dep-revalidation-trigger",
			"dep-revocation":   "dep-revocation-trigger",
		},
		"dep-evidence": {
			"dep-evidence-write": "dep-evidence-write-trigger",
			"dep-evidence-audit": "dep-evidence-audit-trigger",
		},
	}

	declared := 0
	for stack, jobs := range triggers {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		for job, trigger := range jobs {
			if !strings.Contains(main, `"`+job+`" = {`) {
				t.Fatalf("stacks/%s/main.tf does not declare the canonical job %q", stack, job)
			}
			if !strings.Contains(main, `trigger_id = "`+trigger+`"`) {
				t.Fatalf("stacks/%s/main.tf does not bind the canonical trigger identity %q to the job %q", stack, trigger, job)
			}
			declared++
		}
		if !strings.Contains(main, `invoker_member = "serviceAccount:${module.workload_identity.trigger_service_account_emails[each.value.identity_key]}"`) {
			t.Fatalf("stacks/%s/main.tf does not bind the job invoker to the lane trigger identity", stack)
		}
		if strings.Count(main, "trigger_service_account_emails") != 1 {
			t.Fatalf("stacks/%s/main.tf references the trigger identities outside the job invoker binding", stack)
		}
	}
	if declared != 7 {
		t.Fatalf("the stacks declare %d trigger identities, want the complete canonical set of 7", declared)
	}

	// The identity wiring injects the canonical trigger identity into every
	// lane identity.
	intakeMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-intake", "main.tf")))
	if !strings.Contains(intakeMain, `fetcher = merge(var.fetcher, { trigger_service_account_id = local.workload_jobs["dep-intake-fetch"].trigger_id })`) {
		t.Fatal("stacks/dep-intake/main.tf does not inject the trigger identity into the fetcher identity")
	}
	controlMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	for _, required := range []string{
		`controller_triggers = { for job, spec in local.workload_jobs : spec.identity_key => spec.trigger_id }`,
		`for lane, controller in var.controllers : lane => merge(controller, { trigger_service_account_id = local.controller_triggers[lane] })`,
	} {
		if !strings.Contains(controlMain, required) {
			t.Fatalf("stacks/dep-control/main.tf does not bind %q", required)
		}
	}
	evidenceMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-evidence", "main.tf")))
	for _, required := range []string{
		`writer = merge(var.writer, { trigger_service_account_id = local.workload_jobs["dep-evidence-write"].trigger_id })`,
		`auditor = merge(var.auditor, { trigger_service_account_id = local.workload_jobs["dep-evidence-audit"].trigger_id })`,
	} {
		if !strings.Contains(evidenceMain, required) {
			t.Fatalf("stacks/dep-evidence/main.tf does not bind %q", required)
		}
	}

	// The workload-identity module owns the trigger identities: one dedicated
	// service account per lane, the principal-set binding on the trigger
	// identity and never on the execution identity, and no roles for the
	// trigger identity.
	identityMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "workload-identity", "main.tf")))
	if !strings.Contains(identityMain, `resource "google_service_account" "trigger"`) {
		t.Fatal("modules/workload-identity/main.tf does not create the dedicated trigger identities")
	}
	bindingStart := strings.Index(identityMain, `resource "google_service_account_iam_member" "workload_identity_user"`)
	if bindingStart < 0 {
		t.Fatal("modules/workload-identity/main.tf does not carry the principal-set binding")
	}
	bindingSegment := identityMain[bindingStart:]
	if next := strings.Index(bindingSegment, ` resource "`); next > 0 {
		bindingSegment = bindingSegment[:next]
	}
	if !strings.Contains(bindingSegment, `service_account_id = google_service_account.trigger[each.key].name`) {
		t.Fatal("the principal-set binding must federate the trigger identity")
	}
	if strings.Contains(bindingSegment, `google_service_account.this[`) {
		t.Fatal("the principal-set binding must never federate the execution identity")
	}
	rolesStart := strings.Index(identityMain, `resource "google_project_iam_member" "identity_roles"`)
	if rolesStart < 0 {
		t.Fatal("modules/workload-identity/main.tf does not carry the identity roles binding")
	}
	if strings.Contains(identityMain[rolesStart:], "trigger") {
		t.Fatal("the trigger identity must never receive a role")
	}

	identityVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "workload-identity", "variables.tf")))
	for _, required := range []string{
		`trigger_service_account_id = string`,
		`identity.trigger_service_account_id != identity.service_account_id`,
	} {
		if !strings.Contains(identityVariables, required) {
			t.Fatalf("modules/workload-identity/variables.tf does not bind %q", required)
		}
	}
	if strings.Contains(identityVariables, `trigger_service_account_id = optional(`) {
		t.Fatal("the trigger service account must be required, never optional")
	}

	identityOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "workload-identity", "outputs.tf")))
	if !strings.Contains(identityOutputs, `output "trigger_service_account_emails"`) {
		t.Fatal("modules/workload-identity/outputs.tf does not export the trigger identity emails")
	}

	// The cloud-run-job module binds the invoke-only grant on exactly the own
	// job with the proven role contents.
	jobMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "main.tf")))
	for _, required := range []string{
		`resource "google_cloud_run_v2_job_iam_member" "invoker"`,
		`resource "google_cloud_run_v2_job_iam_member" "invoker_readback"`,
		`role = "roles/run.jobsExecutorWithOverrides"`,
		`role = "roles/run.viewer"`,
		`member = var.invoker_member`,
	} {
		if !strings.Contains(jobMain, required) {
			t.Fatalf("modules/cloud-run-job/main.tf does not bind %q", required)
		}
	}
	if strings.Contains(jobMain, "roles/run.invoker") {
		t.Fatal("modules/cloud-run-job/main.tf still binds roles/run.invoker; the lane invocation is an override execution and requires roles/run.jobsExecutorWithOverrides")
	}
	jobVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "variables.tf")))
	if !strings.Contains(jobVariables, `variable "invoker_member"`) {
		t.Fatal("modules/cloud-run-job/variables.tf does not carry the invoker_member input")
	}

	// The invoke roles stay resource-scoped: no project-level Cloud Run grant
	// exists anywhere in the core. The single declared exception is the
	// forensics reader access class (the forensics-readers module): the
	// organization-owned forensics group holds roles/run.viewer project-scoped
	// for the execution status read-back — a standing read-only diagnostic
	// identity, never a lane trigger identity; its exact surface is pinned
	// fail-closed by TestStacksDeclareTheForensicsReaderAccessClass.
	for _, path := range repositoryFiles(t, []string{".tf"}) {
		if strings.HasSuffix(filepath.ToSlash(path), "modules/forensics-readers/main.tf") {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		flattened := normalizeWhitespace(string(content))
		if !strings.Contains(flattened, "google_project_iam_member") {
			continue
		}
		for _, role := range []string{"roles/run.jobsExecutorWithOverrides", "roles/run.viewer"} {
			if strings.Contains(flattened, role) {
				t.Fatalf("%s grants %s at project level; the trigger identity holds invoke resource-scoped on exactly its own job", path, role)
			}
		}
	}

	// No trigger identity ever receives a data-plane grant: no repository IAM
	// module block references the trigger identities.
	for _, stack := range stackNames {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		for _, segment := range strings.Split(main, `module "`) {
			if !strings.Contains(segment, "modules/repository-iam") {
				continue
			}
			if strings.Contains(segment, "trigger") {
				t.Fatalf("stacks/%s binds a trigger identity in a repository IAM module; trigger identities never hold data-plane grants", stack)
			}
		}
	}

	// Zone purity: the quarantine and approved zones never carry trigger
	// identities.
	for _, stack := range []string{"dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "trigger") {
			t.Fatalf("stacks/%s must never declare trigger identities; the topology is zone-pure", stack)
		}
	}

	// The pass-through identity surfaces of the job-free zones carry the same
	// required trigger field, so any future zone-local identity binds a
	// dedicated trigger identity fail-closed.
	for _, stack := range []string{"dep-approved", "dep-quarantine"} {
		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		start := strings.Index(variables, `variable "identities" {`)
		if start < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the optional identities input", stack)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if !strings.Contains(segment, `trigger_service_account_id = string`) {
			t.Fatalf("stacks/%s/variables.tf does not carry the required trigger identity field in the identities input", stack)
		}
	}

	// The stack outputs export the trigger identity emails for the instance
	// bindings.
	intakeOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-intake", "outputs.tf")))
	if !strings.Contains(intakeOutputs, `output "fetcher_trigger_service_account_email"`) {
		t.Fatal("stacks/dep-intake/outputs.tf does not export the trigger identity email")
	}
	controlOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "outputs.tf")))
	if !strings.Contains(controlOutputs, `output "controller_trigger_service_account_emails"`) {
		t.Fatal("stacks/dep-control/outputs.tf does not export the trigger identity emails")
	}
	evidenceOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-evidence", "outputs.tf")))
	if !strings.Contains(evidenceOutputs, `output "workload_trigger_service_account_emails"`) {
		t.Fatal("stacks/dep-evidence/outputs.tf does not export the trigger identity emails")
	}

	// The module documentation carries the trigger identity form.
	identityReadme := readRepositoryFile(t, filepath.Join("modules", "workload-identity", "README.md"))
	if !strings.Contains(identityReadme, "trigger_service_account_id") {
		t.Fatal("the workload-identity module README does not document the trigger identity")
	}
	jobReadme := readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "README.md"))
	for _, required := range []string{"invoker_member", "roles/run.jobsExecutorWithOverrides", "roles/run.viewer"} {
		if !strings.Contains(jobReadme, required) {
			t.Fatalf("the cloud-run-job module README does not document %q", required)
		}
	}

	// The architecture decision record carries the trigger identity decision
	// including its exclusions.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"invoke-only trigger identity",
		"dep-<operation>-trigger",
		"roles/run.jobsExecutorWithOverrides",
		"run.jobs.runWithOverrides",
		"roles/run.viewer",
		"never federated",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the trigger identity decision element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-12") {
		t.Fatal("TRACEABILITY.md does not contain DAI-12")
	}
}

func TestStacksDeclareTheWorkloadNetworkOrigin(t *testing.T) {
	// The network module carries the zone workload network origin surface:
	// exactly one VPC with one Private Google Access subnetwork in the job
	// region, the restricted-range DNS response policy and the egress firewall
	// pair ordered around priority 1000.
	networkMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "main.tf")))
	for _, required := range []string{
		`resource "google_compute_network" "workload"`,
		`auto_create_subnetworks = false`,
		`resource "google_compute_subnetwork" "workload"`,
		`private_ip_google_access = true`,
		`resource "google_dns_response_policy" "workload"`,
		`resource "google_dns_response_policy_rule" "restricted_googleapis"`,
		`dns_name = "*.googleapis.com."`,
		`rrdatas = ["199.36.153.4", "199.36.153.5", "199.36.153.6", "199.36.153.7"]`,
		`resource "google_compute_firewall" "allow_restricted_googleapis_egress"`,
		`resource "google_compute_firewall" "deny_all_egress"`,
		`destination_ranges = ["199.36.153.4/30"]`,
		`priority = 999`,
		`priority = 1001`,
		`direction = "EGRESS"`,
	} {
		if !strings.Contains(networkMain, required) {
			t.Fatalf("modules/network/main.tf does not declare the workload network origin element %q", required)
		}
	}

	networkVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "variables.tf")))
	if !strings.Contains(networkVariables, `variable "workload_network" {`) {
		t.Fatal("modules/network/variables.tf does not carry the workload_network input")
	}
	if !strings.Contains(networkVariables, `default = null`) {
		t.Fatal("modules/network/variables.tf must default workload_network to null; the job-free zones declare no workload network")
	}

	networkOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "outputs.tf")))
	for _, required := range []string{
		`output "workload_network_id"`,
		`output "workload_subnetwork_id"`,
	} {
		if !strings.Contains(networkOutputs, required) {
			t.Fatalf("modules/network/outputs.tf does not export %q", required)
		}
	}

	networkReadme := readRepositoryFile(t, filepath.Join("modules", "network", "README.md"))
	for _, required := range []string{"workload network origin", "Private Google Access", "restricted.googleapis.com", "Direct VPC egress"} {
		if !strings.Contains(networkReadme, required) {
			t.Fatalf("the network module README does not document %q", required)
		}
	}

	// The three job-zone stacks declare their zone network through the network
	// module, wire it into their workload jobs and enforce the form through the
	// Cloud Run organization policies by default.
	for _, stack := range []string{"dep-intake", "dep-control", "dep-evidence"} {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		for _, required := range []string{
			`module "network" {`,
			`source = "../../modules/network"`,
			`workload_network = merge(var.workload_network, { region = var.location })`,
			`network = module.network.workload_network_id`,
			`subnetwork = module.network.workload_subnetwork_id`,
			`cloud_run_vpc_egress_all_traffic_only = var.policy_constraints.cloud_run_vpc_egress_all_traffic_only`,
			`cloud_run_ingress_internal_only = var.policy_constraints.cloud_run_ingress_internal_only`,
		} {
			if !strings.Contains(main, required) {
				t.Fatalf("stacks/%s/main.tf does not declare the workload network origin element %q", stack, required)
			}
		}

		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		start := strings.Index(variables, `variable "workload_network" {`)
		if start < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the workload_network input", stack)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if strings.Contains(segment, "default") {
			t.Fatalf("stacks/%s/variables.tf carries a default for workload_network; the zone network is an instance binding, never a stack default", stack)
		}
		for _, required := range []string{
			`cloud_run_vpc_egress_all_traffic_only = optional(bool, true)`,
			`cloud_run_ingress_internal_only = optional(bool, true)`,
		} {
			if !strings.Contains(variables, required) {
				t.Fatalf("stacks/%s/variables.tf does not default the Cloud Run enforcement %q to the enforced posture", stack, required)
			}
		}

		outputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "outputs.tf")))
		for _, required := range []string{
			`output "workload_network_id"`,
			`output "workload_subnetwork_id"`,
		} {
			if !strings.Contains(outputs, required) {
				t.Fatalf("stacks/%s/outputs.tf does not export %q", stack, required)
			}
		}
	}

	// Zone purity: the job-free zones never declare a workload network or the
	// Cloud Run enforcement surface.
	for _, stack := range []string{"dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "workload_network") {
			t.Fatalf("stacks/%s must never declare a workload network; the topology is zone-pure", stack)
		}
		variables := readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf"))
		for _, forbidden := range []string{`variable "workload_network"`, "cloud_run_vpc_egress", "cloud_run_ingress"} {
			if strings.Contains(variables, forbidden) {
				t.Fatalf("stacks/%s/variables.tf must never carry %q; the job-free zones enforce no Cloud Run form", stack, forbidden)
			}
		}
	}

	// The policy bindings carry the opt-in Cloud Run enforcement surface.
	policyMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("policy-bindings", "main.tf")))
	for _, required := range []string{
		`resource "google_org_policy_policy" "cloud_run_vpc_egress"`,
		`resource "google_org_policy_policy" "cloud_run_ingress"`,
		`"projects/${var.project_id}/policies/run.allowedVPCEgress"`,
		`"projects/${var.project_id}/policies/run.allowedIngress"`,
		`allowed_values = ["all-traffic"]`,
		`allowed_values = ["internal"]`,
	} {
		if !strings.Contains(policyMain, required) {
			t.Fatalf("policy-bindings/main.tf does not declare the Cloud Run enforcement element %q", required)
		}
	}
	policyVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("policy-bindings", "variables.tf")))
	for _, required := range []string{
		`variable "cloud_run_vpc_egress_all_traffic_only" {`,
		`variable "cloud_run_ingress_internal_only" {`,
	} {
		if !strings.Contains(policyVariables, required) {
			t.Fatalf("policy-bindings/variables.tf does not carry %q", required)
		}
	}
	if strings.Count(policyVariables, "default = false") != 2 {
		t.Fatal("the Cloud Run enforcement constraints are opt-in and default to not-enforced")
	}

	// The architecture decision record carries the network origin decision.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"workload network origin",
		"Direct VPC egress",
		"all-traffic",
		"199.36.153.4/30",
		"run.allowedVPCEgress",
		"run.allowedIngress",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the workload network origin element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-13") {
		t.Fatal("TRACEABILITY.md does not contain DAI-13")
	}
}

func TestStacksDeclareTheForensicsReaderAccessClass(t *testing.T) {
	// The forensics-readers module owns the read-only diagnostic access class:
	// exactly the two project-scoped read-only roles for the instance-bound
	// forensics group and the once-declared perimeter ingress rule.
	moduleMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "forensics-readers", "main.tf")))
	for _, required := range []string{
		`resource "google_project_iam_member" "log_reader"`,
		`resource "google_project_iam_member" "execution_reader"`,
		`role = "roles/logging.viewer"`,
		`role = "roles/run.viewer"`,
		`member = var.forensics_group`,
		`resource "google_access_context_manager_service_perimeter_ingress_policy" "forensics"`,
		`count = var.perimeter_ingress == null ? 0 : 1`,
		`perimeter = var.perimeter_ingress.perimeter_name`,
		`identities = [var.forensics_group]`,
		`access_level = "*"`,
		`resources = tolist(var.perimeter_ingress.zone_projects)`,
		`service_name = "logging.googleapis.com"`,
		`permission = "logging.logEntries.list"`,
	} {
		if !strings.Contains(moduleMain, required) {
			t.Fatalf("modules/forensics-readers/main.tf does not declare the forensics reader access class element %q", required)
		}
	}

	// The class is structurally read-only: the module grants exactly the two
	// diagnostic roles and never creates identities, never binds at
	// organization or folder level and never owns the perimeter itself.
	if count := strings.Count(moduleMain, "roles/"); count != 2 {
		t.Fatalf("modules/forensics-readers/main.tf carries %d role references, want exactly the two read-only diagnostic roles", count)
	}
	for _, forbidden := range []string{
		"google_organization_iam",
		"google_folder_iam",
		"google_service_account",
		`google_access_context_manager_service_perimeter"`,
	} {
		if strings.Contains(moduleMain, forbidden) {
			t.Fatalf("modules/forensics-readers/main.tf must never contain %q; the class creates no identities, holds no organization- or folder-level grant and never owns the perimeter itself", forbidden)
		}
	}

	moduleVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "forensics-readers", "variables.tf")))
	for _, required := range []string{
		`variable "forensics_group" {`,
		`variable "perimeter_ingress" {`,
		`default = null`,
		`^group:dep-forensics-readers@`,
		`^accessPolicies/[0-9]+/servicePerimeters/[A-Za-z0-9_]+$`,
		`^projects/[0-9]+$`,
	} {
		if !strings.Contains(moduleVariables, required) {
			t.Fatalf("modules/forensics-readers/variables.tf does not bind %q", required)
		}
	}
	// The forensics group is an instance binding, never a module-assigned value.
	groupStart := strings.Index(moduleVariables, `variable "forensics_group" {`)
	if groupStart < 0 {
		t.Fatal("modules/forensics-readers/variables.tf does not carry the forensics_group input")
	}
	groupSegment := moduleVariables[groupStart:]
	if next := strings.Index(groupSegment, ` variable "`); next > 0 {
		groupSegment = groupSegment[:next]
	}
	if strings.Contains(groupSegment, "default") {
		t.Fatal("modules/forensics-readers/variables.tf carries a default for forensics_group; the forensics group is an instance binding, never a module-assigned value")
	}

	moduleOutputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "forensics-readers", "outputs.tf")))
	for _, required := range []string{
		`output "log_reader_binding_id"`,
		`output "execution_reader_binding_id"`,
		`output "ingress_policy_id"`,
	} {
		if !strings.Contains(moduleOutputs, required) {
			t.Fatalf("modules/forensics-readers/outputs.tf does not export %q", required)
		}
	}

	moduleReadme := readRepositoryFile(t, filepath.Join("modules", "forensics-readers", "README.md"))
	for _, required := range []string{"forensics reader access class", "roles/logging.viewer", "roles/run.viewer", "logging.logEntries.list", "never carries the forensics identity"} {
		if !strings.Contains(moduleReadme, required) {
			t.Fatalf("the forensics-readers module README does not document %q", required)
		}
	}

	// Every zone stack binds the instance-supplied forensics group through the
	// module; the group is required everywhere and never a stack-assigned value.
	for _, stack := range stackNames {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		for _, required := range []string{
			`module "forensics_readers" {`,
			`source = "../../modules/forensics-readers"`,
			`forensics_group = var.forensics_group`,
		} {
			if !strings.Contains(main, required) {
				t.Fatalf("stacks/%s/main.tf does not declare the forensics reader access class element %q", stack, required)
			}
		}

		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		start := strings.Index(variables, `variable "forensics_group" {`)
		if start < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the forensics_group input", stack)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if strings.Contains(segment, "default") {
			t.Fatalf("stacks/%s/variables.tf carries a default for forensics_group; the forensics group is an instance binding, never a stack-assigned value", stack)
		}
	}

	// The perimeter ingress rule is declared exactly once: the control-zone
	// stack binds it as a required instance input; every other stack is pure
	// and never carries the rule surface.
	controlMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	if !strings.Contains(controlMain, `perimeter_ingress = var.perimeter_ingress`) {
		t.Fatal("stacks/dep-control/main.tf does not wire the forensics perimeter ingress rule")
	}
	controlVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tf")))
	ingressStart := strings.Index(controlVariables, `variable "perimeter_ingress" {`)
	if ingressStart < 0 {
		t.Fatal("stacks/dep-control/variables.tf does not carry the perimeter_ingress input")
	}
	ingressSegment := controlVariables[ingressStart:]
	if strings.Contains(ingressSegment, "default") {
		t.Fatal("stacks/dep-control/variables.tf carries a default for perimeter_ingress; the rule is an instance binding, never a stack-assigned value")
	}
	for _, stack := range []string{"dep-intake", "dep-evidence", "dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "perimeter_ingress") {
			t.Fatalf("stacks/%s must never declare the forensics perimeter ingress rule; the boundary-level binding lives exactly once in dep-control", stack)
		}
		variables := readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf"))
		if strings.Contains(variables, "perimeter_ingress") {
			t.Fatalf("stacks/%s/variables.tf must never carry perimeter_ingress; the boundary-level binding lives exactly once in dep-control", stack)
		}
	}

	// The architecture decision record carries the forensics reader access
	// class decision including its exclusions.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"forensics reader access class",
		"dep-forensics-readers",
		"roles/logging.viewer",
		"roles/run.viewer",
		"logging.logEntries.list",
		"never carries the forensics identity",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the forensics reader access class element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-14") {
		t.Fatal("TRACEABILITY.md does not contain DAI-14")
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
