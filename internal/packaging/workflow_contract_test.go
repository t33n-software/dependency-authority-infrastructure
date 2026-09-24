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
		"window.imports.tf",
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
			"dep-admission":             "admission",
			"dep-promotion":             "promotion",
			"dep-revalidation":          "revalidation",
			"dep-revocation":            "revocation",
			"dep-consumer-verification": "consumer-verification",
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
			`for_each = { for job, spec in local.workload_jobs : job => spec if contains(var.enabled_workload_jobs, job) }`,
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
	if declared != 8 {
		t.Fatalf("the stacks declare %d workload jobs, want the complete canonical topology of 8", declared)
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
	if !strings.Contains(traceability, "DAI-20") {
		t.Fatal("TRACEABILITY.md does not contain DAI-20")
	}
}

func TestStacksBindTheWorkloadJobActivationGate(t *testing.T) {
	// The engine-active surface composition of the operating model: every
	// job-owning stack consumes the instance-bound activation set and filters
	// the declared topology through it, so a declared-but-planned job never
	// enters the plan until its provisioning window activates it. The set
	// always carries every bound job; the declaration binds the consistency
	// fail-closed.
	for _, stack := range []string{"dep-intake", "dep-control", "dep-evidence"} {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		if !strings.Contains(main, `for_each = { for job, spec in local.workload_jobs : job => spec if contains(var.enabled_workload_jobs, job) }`) {
			t.Fatalf("stacks/%s/main.tf does not filter the workload job topology through the instance-bound activation set", stack)
		}

		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		start := strings.Index(variables, `variable "enabled_workload_jobs" {`)
		if start < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the enabled_workload_jobs input", stack)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if strings.Contains(segment, "default") {
			t.Fatalf("stacks/%s/variables.tf carries a default for enabled_workload_jobs; the activation set is an instance binding, never a stack default", stack)
		}
		if !strings.Contains(segment, `length(setsubtract(var.enabled_workload_jobs, keys(local.workload_jobs))) == 0`) {
			t.Fatalf("stacks/%s/variables.tf does not bind the activation set to the declared topology fail-closed", stack)
		}
	}

	// Zone purity: the job-free zones never carry the activation surface.
	for _, stack := range []string{"dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "enabled_workload_jobs") {
			t.Fatalf("stacks/%s must never carry the activation set; the topology is zone-pure", stack)
		}
		variables := readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf"))
		if strings.Contains(variables, "enabled_workload_jobs") {
			t.Fatalf("stacks/%s/variables.tf must never carry the activation set; the topology is zone-pure", stack)
		}
	}

	// Every job-owning stack carries the behavioral proof of the gate beside
	// the code: the acceptance run and the rejection run of an unknown
	// enabled job.
	for _, stack := range []string{"dep-intake", "dep-control", "dep-evidence"} {
		fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tofutest.hcl")))
		if !strings.Contains(fixture, `run "accepts_the_activation_set"`) {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the acceptance run of the activation set", stack)
		}
		if !strings.Contains(fixture, "expect_failures = [var.enabled_workload_jobs]") {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the rejection run of an unknown enabled job", stack)
		}
	}

	// The module documentation carries the activation form.
	jobReadme := readRepositoryFile(t, filepath.Join("modules", "cloud-run-job", "README.md"))
	for _, required := range []string{"activation", "enabled_workload_jobs", "engine-active"} {
		if !strings.Contains(jobReadme, required) {
			t.Fatalf("the cloud-run-job module README does not document %q of the activation form", required)
		}
	}

	// The architecture decision record carries the activation decision.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"activation set",
		"enabled_workload_jobs",
		"engine-active",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the activation gate element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-24") {
		t.Fatal("TRACEABILITY.md does not contain DAI-24")
	}
}

func TestEveryStackDeclaresTheBreakGlassRecovery(t *testing.T) {
	// Every trust zone declares its own break-glass recovery identity through
	// the recovery module, bound to the zone's own project: the dedicated,
	// dormant identity whose elevated project role exists only under the
	// mandatory time-bound IAM condition, with the role and the end time as
	// approved instance decisions.
	for _, stack := range stackNames {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		if count := strings.Count(main, `module "recovery" {`); count != 1 {
			t.Fatalf("stacks/%s/main.tf declares the recovery binding %d times, want exactly once", stack, count)
		}
		for _, required := range []string{
			`source = "../../modules/recovery"`,
			`project_id = var.project_id`,
			`role = var.break_glass_recovery.role`,
			`condition_end_time = var.break_glass_recovery.condition_end_time`,
		} {
			if !strings.Contains(main, required) {
				t.Fatalf("stacks/%s/main.tf does not declare the recovery binding element %q", stack, required)
			}
		}

		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		start := strings.Index(variables, `variable "break_glass_recovery" {`)
		if start < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the break_glass_recovery input", stack)
		}
		segment := variables[start:]
		if next := strings.Index(segment, ` variable "`); next > 0 {
			segment = segment[:next]
		}
		if strings.Contains(segment, "default") {
			t.Fatalf("stacks/%s/variables.tf carries a default for break_glass_recovery; the role and the end time are approved instance decisions, never stack defaults", stack)
		}
		for _, required := range []string{
			`role = string`,
			`condition_end_time = string`,
			`can(regex("^roles/[A-Za-z][A-Za-z0-9._]+$", var.break_glass_recovery.role))`,
			`can(timecmp(var.break_glass_recovery.condition_end_time, "1970-01-01T00:00:00Z"))`,
		} {
			if !strings.Contains(segment, required) {
				t.Fatalf("stacks/%s/variables.tf does not bind the recovery element %q", stack, required)
			}
		}

		outputs := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "outputs.tf")))
		if !strings.Contains(outputs, `output "break_glass_recovery_service_account_email"`) {
			t.Fatalf("stacks/%s/outputs.tf does not export the recovery identity email", stack)
		}
		if !strings.Contains(outputs, `value = module.recovery.service_account_email`) {
			t.Fatalf("stacks/%s/outputs.tf does not wire the recovery identity email to the module", stack)
		}

		// The behavioral proof of the recovery binding lives beside the code.
		fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tofutest.hcl")))
		if !strings.Contains(fixture, "break_glass_recovery = {") {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the synthetic recovery binding", stack)
		}
		if !strings.Contains(fixture, "expect_failures = [var.break_glass_recovery]") {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the rejection run of the recovery binding", stack)
		}

		readme := readRepositoryFile(t, filepath.Join("stacks", stack, "README.md"))
		if !strings.Contains(readme, "break_glass_recovery") {
			t.Fatalf("the %s stack README does not document the break_glass_recovery input", stack)
		}
		if !strings.Contains(readme, "recovery identity") {
			t.Fatalf("the %s stack README does not document the recovery identity boundary", stack)
		}
	}

	// The architecture decision record carries the per-zone decision.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"Every trust zone",
		"break-glass recovery identity",
		"one per zone project",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the per-zone recovery decision element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-26") {
		t.Fatal("TRACEABILITY.md does not contain DAI-26")
	}
	if !strings.Contains(traceability, "DAI-24") {
		t.Fatal("TRACEABILITY.md does not contain DAI-24")
	}
}

func TestStacksDeclareTheCanonicalIAMTargetMatrix(t *testing.T) {
	// Intake: the fetcher is the only writer; the matrix readers (canonically
	// the admission, revalidation and promotion controllers of the control
	// zone) arrive through the instance-wired member input.
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
	// admission, revalidation, revocation, promotion and consumer verification
	// controllers of the control zone) arrive through the member inputs.
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
	if !strings.Contains(evidenceVariables, "admission, revalidation, revocation and promotion controllers") {
		t.Fatal("stacks/dep-evidence/variables.tf does not name the promotion controller as a canonical evidence writer; the promotion writes its approved record into the evidence repository")
	}
	if !strings.Contains(evidenceVariables, "consumer verification controller") {
		t.Fatal("stacks/dep-evidence/variables.tf does not name the consumer verification controller as a canonical evidence writer; the consumer verification writes its lane evidence into the evidence repository")
	}
	// The retired form: the promoter was the canonical additional evidence
	// reader; the promotion now writes its approved record itself, so the
	// auditor input never names the promoter as the canonical reader.
	auditorStart := strings.Index(evidenceVariables, `variable "additional_auditor_members" {`)
	if auditorStart < 0 {
		t.Fatal("stacks/dep-evidence/variables.tf does not carry the additional_auditor_members input")
	}
	auditorSegment := evidenceVariables[auditorStart:]
	if next := strings.Index(auditorSegment, ` variable "`); next > 0 {
		auditorSegment = auditorSegment[:next]
	}
	if strings.Contains(auditorSegment, "promoter") {
		t.Fatal("stacks/dep-evidence/variables.tf still names the approved promoter as a canonical evidence reader; the promotion writes its approved record into the evidence repository through the writer grant")
	}

	// Approved: no zone-local workload identity; the promotion and revocation
	// writes and the revalidation and consumer verification reads are
	// control-zone members bound through the member inputs.
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
	if !strings.Contains(approvedVariables, "consumer verification controller") {
		t.Fatal("stacks/dep-approved/variables.tf does not name the consumer verification controller as a canonical consumer reader of the approved repositories")
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
		"dep-revalidation-controller control reader on *-dependencies-intake",
		"dep-revocation-controller",
		"dep-evidence-writer",
		"dep-evidence-auditor",
		"dep-break-glass-recovery",
		"release-controller-images",
		"no data-plane grant",
		"dep-approved-promoter control reader on *-dependencies-intake; writer on *-dependencies-approved and *-dependencies-evidence (the promotion writes its approved record into the evidence repository); reader on release-controller-images",
		"dep-consumer-verifier",
		"dep-consumer-verifier control reader on *-dependencies-approved",
		"writer on *-dependencies-evidence (the consumer verification writes its lane evidence into the evidence repository); reader on release-controller-images",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the canonical IAM target matrix element %q", required)
		}
	}
	if strings.Contains(adr, "dep-approved-promoter control reader on *-dependencies-intake and *-dependencies-evidence") {
		t.Fatal("ADR-0001 carries the retired promoter reader form on the evidence repository; the promotion writes its approved record into the evidence repository")
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-11") {
		t.Fatal("TRACEABILITY.md does not contain DAI-11")
	}
	if !strings.Contains(traceability, "DAI-20") {
		t.Fatal("TRACEABILITY.md does not contain DAI-20")
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
			"dep-admission":             "dep-admission-trigger",
			"dep-promotion":             "dep-promotion-trigger",
			"dep-revalidation":          "dep-revalidation-trigger",
			"dep-revocation":            "dep-revocation-trigger",
			"dep-consumer-verification": "dep-consumer-verifier-trigger",
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
	if declared != 8 {
		t.Fatalf("the stacks declare %d trigger identities, want the complete canonical set of 8", declared)
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
	if !strings.Contains(traceability, "DAI-20") {
		t.Fatal("TRACEABILITY.md does not contain DAI-20")
	}
}

func TestStacksDeclareTheWorkloadNetworkOrigin(t *testing.T) {
	// The network module carries the zone workload network origin surface:
	// exactly one VPC with one Private Google Access subnetwork in the job
	// region, the restricted-range DNS response policy (covering the Google
	// API calls and the Artifact Registry data plane pkg.dev) and the egress
	// firewall pair ordered around priority 1000.
	networkMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "main.tf")))
	for _, required := range []string{
		`resource "google_compute_network" "workload"`,
		`auto_create_subnetworks = false`,
		`resource "google_compute_subnetwork" "workload"`,
		`private_ip_google_access = true`,
		`resource "google_dns_response_policy" "workload"`,
		`resource "google_dns_response_policy_rule" "restricted_googleapis"`,
		`dns_name = "*.googleapis.com."`,
		`resource "google_dns_response_policy_rule" "restricted_pkg_dev"`,
		`dns_name = "*.pkg.dev."`,
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
	// The module declares exactly the two restricted-range rules — the Google
	// API form and the Artifact Registry data plane — and both carry the same
	// restricted VIP record set.
	if count := strings.Count(networkMain, `resource "google_dns_response_policy_rule"`); count != 2 {
		t.Fatalf("modules/network/main.tf declares %d DNS response policy rules, want exactly 2 (restricted_googleapis and restricted_pkg_dev)", count)
	}
	if count := strings.Count(networkMain, `rrdatas = ["199.36.153.4", "199.36.153.5", "199.36.153.6", "199.36.153.7"]`); count != 2 {
		t.Fatalf("modules/network/main.tf carries %d restricted-range record sets, want exactly 2 (the googleapis rule and the pkg.dev data-plane rule)", count)
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
	for _, required := range []string{"workload network origin", "Private Google Access", "restricted.googleapis.com", "Direct VPC egress", "pkg.dev"} {
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
		`"projects/${var.project_number}/policies/run.allowedVPCEgress"`,
		`"projects/${var.project_number}/policies/run.allowedIngress"`,
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
		"pkg.dev",
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
	if !strings.Contains(traceability, "DAI-16") {
		t.Fatal("TRACEABILITY.md does not contain DAI-16")
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
		`method = "LoggingServiceV2.ListLogEntries"`,
	} {
		if !strings.Contains(moduleMain, required) {
			t.Fatalf("modules/forensics-readers/main.tf does not declare the forensics reader access class element %q", required)
		}
	}

	// Regression guard (DAI-15): for Cloud Logging the platform supports only
	// the method form in method selectors; the permission form is rejected
	// fail-closed and must never return to the declaration.
	if strings.Contains(moduleMain, `permission = "logging.logEntries.list"`) {
		t.Fatal("modules/forensics-readers/main.tf carries the permission form of the logging read scope; the platform supports only the method form LoggingServiceV2.ListLogEntries for Cloud Logging")
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
	for _, required := range []string{"forensics reader access class", "roles/logging.viewer", "roles/run.viewer", "LoggingServiceV2.ListLogEntries", "never carries the forensics identity"} {
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
		"LoggingServiceV2.ListLogEntries",
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

func TestStacksDeclareTheVpcscUpstreamAllowance(t *testing.T) {
	// The artifact-registry module owns the remote upstream allowance: a
	// null-gated opt-in surface binding the zone-level registry-platform
	// singleton that permits a remote repository's upstream fetch inside the
	// perimeter. The pinned GA provider carries no resource for this surface,
	// so the declaration binds the exactly pinned google-beta provider.
	moduleMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "main.tf")))
	for _, required := range []string{
		`resource "google_artifact_registry_vpcsc_config" "this"`,
		`count = var.vpcsc_upstream_allowance ? 1 : 0`,
		`provider = google-beta`,
		`project = var.project_id`,
		`location = var.location`,
		`vpcsc_policy = "ALLOW"`,
	} {
		if !strings.Contains(moduleMain, required) {
			t.Fatalf("modules/artifact-registry/main.tf does not declare the remote upstream allowance element %q", required)
		}
	}

	moduleVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "variables.tf")))
	for _, required := range []string{
		`variable "vpcsc_upstream_allowance" {`,
		`default = false`,
		`!var.vpcsc_upstream_allowance || var.mode == "REMOTE_REPOSITORY"`,
	} {
		if !strings.Contains(moduleVariables, required) {
			t.Fatalf("modules/artifact-registry/variables.tf does not bind the remote upstream allowance element %q", required)
		}
	}

	moduleVersions := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "versions.tf")))
	for _, required := range []string{
		`source = "hashicorp/google-beta"`,
		`version = "= 7.44.0"`,
	} {
		if !strings.Contains(moduleVersions, required) {
			t.Fatalf("modules/artifact-registry/versions.tf does not carry the exact beta provider pin %q", required)
		}
	}

	// The beta provider scope is fail-closed: the only beta-provider resource
	// declaration in the core is the remote upstream allowance, and the beta
	// pin lives exactly in the owning module and the five zone stacks that
	// call it — nowhere else.
	for _, path := range repositoryFiles(t, []string{".tf"}) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		content := normalizeWhitespace(string(raw))
		slashed := filepath.ToSlash(path)
		if strings.Contains(content, "provider = google-beta") && !strings.HasSuffix(slashed, "modules/artifact-registry/main.tf") {
			t.Fatalf("%s declares a resource against the beta provider; the beta provider scope is exactly the remote upstream allowance of the artifact-registry module", slashed)
		}
		if strings.Contains(content, `hashicorp/google-beta`) &&
			!strings.HasSuffix(slashed, "modules/artifact-registry/versions.tf") &&
			!strings.Contains(slashed, "/stacks/") {
			t.Fatalf("%s carries the beta provider pin outside the owning module and the zone stacks", slashed)
		}
	}

	// Every zone stack pins and configures the beta provider uniformly; only
	// the intake stack opts in to the allowance, and every other stack never
	// wires it (zone purity).
	for _, stack := range stackNames {
		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		if !strings.Contains(main, `provider "google-beta" {`) {
			t.Fatalf("stacks/%s/main.tf does not configure the beta provider", stack)
		}
		versions := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "versions.tf")))
		if !strings.Contains(versions, `source = "hashicorp/google-beta"`) {
			t.Fatalf("stacks/%s/versions.tf does not pin the beta provider", stack)
		}
		lock := readRepositoryFile(t, filepath.Join("stacks", stack, ".terraform.lock.hcl"))
		if !strings.Contains(lock, "hashicorp/google-beta") {
			t.Fatalf("stacks/%s/.terraform.lock.hcl does not record the beta provider", stack)
		}
	}

	intakeMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-intake", "main.tf")))
	if !strings.Contains(intakeMain, "vpcsc_upstream_allowance = true") {
		t.Fatal("stacks/dep-intake/main.tf does not opt in to the remote upstream allowance; the intake zone carries the remote repositories")
	}
	for _, stack := range []string{"dep-control", "dep-evidence", "dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "vpcsc_upstream_allowance") {
			t.Fatalf("stacks/%s must never wire the remote upstream allowance; only zones with remote repositories inside the perimeter opt in", stack)
		}
	}

	// The documentation surfaces carry the boundary: the module README, the
	// intake stack README, the provider-binding convention, the architecture
	// decision record and the register.
	moduleReadme := readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "README.md"))
	for _, required := range []string{"vpcsc_upstream_allowance", "google_artifact_registry_vpcsc_config", "google-beta", "never a perimeter egress rule"} {
		if !strings.Contains(moduleReadme, required) {
			t.Fatalf("the artifact-registry module README does not document %q", required)
		}
	}

	intakeReadme := readRepositoryFile(t, filepath.Join("stacks", "dep-intake", "README.md"))
	for _, required := range []string{"vpcsc_upstream_allowance", "upstream allowance"} {
		if !strings.Contains(intakeReadme, required) {
			t.Fatalf("the dep-intake stack README does not document %q", required)
		}
	}

	convention := readRepositoryFile(t, filepath.Join("docs", "conventions", "provider-binding", "beta-stage-resources.md"))
	for _, required := range []string{"google_artifact_registry_vpcsc_config", "hashicorp/google-beta", "tofu providers schema", "never a perimeter egress rule", "vpcsc_upstream_allowance"} {
		if !strings.Contains(convention, required) {
			t.Fatalf("docs/conventions/provider-binding/beta-stage-resources.md does not carry %q", required)
		}
	}

	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{"hashicorp/google-beta", "google_artifact_registry_vpcsc_config", "vpcsc_upstream_allowance", "never a perimeter egress rule", "docs/conventions/provider-binding/beta-stage-resources.md"} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the dual-provider decision element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-17") {
		t.Fatal("TRACEABILITY.md does not contain DAI-17")
	}
}

func TestZoneStacksBindTheFoundationProvisionedStateHome(t *testing.T) {
	// Every zone stack consumes its state home — the dedicated state bucket
	// of the zone, provisioned by the converged foundation, never by the
	// zone's own roots and never by hand: the backend binding is final from
	// birth, and the engine layer of the dual fortress form encrypts every
	// state and plan artifact of the root client-side, fail-closed enforced.
	for _, stack := range stackNames {
		versions := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "versions.tf")))
		for _, required := range []string{
			`backend "gcs" {`,
			`bucket = var.state_bucket_name`,
			`prefix = "` + stack + `"`,
			`key_provider "gcp_kms" "main"`,
			`kms_encryption_key = var.state_encryption_key`,
			`key_length = 32`,
			`encrypted_metadata_alias = "state-encryption"`,
			`method "aes_gcm" "main"`,
			`keys = key_provider.gcp_kms.main`,
			`state { method = method.aes_gcm.main enforced = true`,
			`plan { method = method.aes_gcm.main enforced = true`,
			`remote_state_data_sources { default { method = method.aes_gcm.main`,
		} {
			if !strings.Contains(versions, required) {
				t.Fatalf("stacks/%s/versions.tf does not bind the foundation-provisioned state-home form %q", stack, required)
			}
		}
		if count := strings.Count(versions, "enforced = true"); count != 2 {
			t.Fatalf("stacks/%s/versions.tf carries %d fail-closed enforcements, want exactly 2 (state and plan)", stack, count)
		}

		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		for _, required := range []string{
			`variable "state_bucket_name" {`,
			`variable "state_encryption_key" {`,
			`^projects/[^/]+/locations/[^/]+/keyRings/[^/]+/cryptoKeys/[^/]+$`,
			`^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$`,
		} {
			if !strings.Contains(variables, required) {
				t.Fatalf("stacks/%s/variables.tf does not bind the state-home consumption input %q", stack, required)
			}
		}
		// The bucket name and the engine key reference are instance bindings
		// without defaults.
		for _, name := range []string{"state_bucket_name", "state_encryption_key"} {
			start := strings.Index(variables, `variable "`+name+`" {`)
			if start < 0 {
				t.Fatalf("stacks/%s/variables.tf does not carry the %s input", stack, name)
			}
			segment := variables[start:]
			if next := strings.Index(segment, ` variable "`); next > 0 {
				segment = segment[:next]
			}
			if strings.Contains(segment, "default") {
				t.Fatalf("stacks/%s/variables.tf carries a default for %s; the value is an instance binding, never a default", stack, name)
			}
		}
	}

	// The zone self-provisioning form is retired: no state-home root and no
	// state-home module exist anymore — the foundation provisions every zone
	// state home through its own converged engine path.
	for _, root := range []string{"stacks/dep-control-state", "modules/state-home"} {
		if _, err := os.Stat(repositoryPath(filepath.FromSlash(root))); !os.IsNotExist(err) {
			t.Fatalf("%s still exists; the foundation provisions every zone state home, and the zone self-provisioning form is retired", root)
		}
	}

	// The architecture decision record carries the foundation-provisioned
	// state-home form.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"state home",
		"foundation",
		"dual fortress",
		"never by the zone",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the foundation-provisioned state-home element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-22") {
		t.Fatal("TRACEABILITY.md does not contain DAI-22")
	}
}

func TestZoneStacksBindTheProvenStateBucketNameValidation(t *testing.T) {
	// Regression guard (DAI-23): the state_bucket_name validation of every
	// zone stack binds the documented string-matching form of the OpenTofu
	// quality-gates reference. The retired form bound the membership test
	// contains against the string variable, which errors only when a value is
	// evaluated and therefore passed format, initialization and validation
	// silently — the proven defect class. Every corrected root also carries
	// its behavioral proof beside the code: the acceptance run and the four
	// rejection runs of the naming rules, executed in the governed execution
	// window because the encryption-carrying roots never initialize offline.
	for _, stack := range stackNames {
		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		if !strings.Contains(variables, `!can(regex("google", var.state_bucket_name))`) {
			t.Fatalf("stacks/%s/variables.tf does not bind the documented string-matching form for the google-spelling clause of state_bucket_name", stack)
		}
		if strings.Contains(variables, "contains(var.state_bucket_name") {
			t.Fatalf("stacks/%s/variables.tf still carries the membership test on the state_bucket_name string; contains requires a list, tuple or set as its first argument and errors at evaluation on a string", stack)
		}

		fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tofutest.hcl")))
		if count := strings.Count(fixture, "expect_failures = [var.state_bucket_name]"); count != 4 {
			t.Fatalf("stacks/%s/variables.tofutest.hcl carries %d rejection runs of state_bucket_name, want exactly 4 (invalid characters, IP form, goog prefix, google substring)", stack, count)
		}
		if !strings.Contains(fixture, `run "accepts_a_valid_state_bucket_name"`) {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the acceptance run of state_bucket_name", stack)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-23") {
		t.Fatal("TRACEABILITY.md does not contain DAI-23")
	}
}

func TestControlStackBindsTheWorkloadImageCleanupLifecycle(t *testing.T) {
	// The workload image lifecycle and retention convention: the declared,
	// platform-executed cleanup policies of the two workload image registries.
	// The module owns the provider-schema-proven surface and binds the class
	// rule fail-closed; the control stack binds the instance-supplied values
	// through the required input; every other stack is pure.
	moduleVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "variables.tf")))
	for _, required := range []string{
		`variable "cleanup_policies" {`,
		`variable "cleanup_policy_dry_run" {`,
		`default = true`,
		`length(var.cleanup_policies) == 0 || var.format == "DOCKER"`,
		`append-only supply-chain records`,
	} {
		if !strings.Contains(moduleVariables, required) {
			t.Fatalf("modules/artifact-registry/variables.tf does not bind the workload image cleanup lifecycle element %q", required)
		}
	}

	moduleMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "main.tf")))
	for _, required := range []string{
		`dynamic "cleanup_policies" {`,
		`for_each = var.cleanup_policies`,
		`id = cleanup_policies.key`,
		`cleanup_policy_dry_run = var.cleanup_policy_dry_run`,
	} {
		if !strings.Contains(moduleMain, required) {
			t.Fatalf("modules/artifact-registry/main.tf does not bind the cleanup policy surface element %q", required)
		}
	}

	// The behavioral proof of the class rule lives beside the module and
	// carries the acceptance and rejection runs.
	moduleFixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "variables.tofutest.hcl")))
	for _, required := range []string{
		`run "accepts_docker_cleanup_policies"`,
		`run "rejects_cleanup_policies_on_a_dependency_repository"`,
		`run "rejects_cleanup_policies_on_an_evidence_repository"`,
		`expect_failures = [var.cleanup_policies]`,
	} {
		if !strings.Contains(moduleFixture, required) {
			t.Fatalf("modules/artifact-registry/variables.tofutest.hcl does not carry the class-rule proof element %q", required)
		}
	}

	// The control stack binds the instance-supplied lifecycle values through
	// the required input — never a stack default — and proves the
	// convention's structural rules fail-closed.
	controlVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tf")))
	start := strings.Index(controlVariables, `variable "workload_image_cleanup" {`)
	if start < 0 {
		t.Fatal("stacks/dep-control/variables.tf does not carry the workload_image_cleanup input")
	}
	segment := controlVariables[start:]
	if strings.Contains(segment, "default") {
		t.Fatal("stacks/dep-control/variables.tf carries a default for workload_image_cleanup; the lifecycle binding is an instance binding, never a stack default")
	}
	for _, required := range []string{
		`dry_run = bool`,
		`policies = map(object(`,
		`alltrue([for _, policy in var.workload_image_cleanup.release.policies : policy.action == "KEEP"])`,
		`anytrue([for _, policy in var.workload_image_cleanup.staging.policies : policy.action == "DELETE" && policy.condition != null && policy.condition.older_than != null])`,
		`anytrue([for _, policy in var.workload_image_cleanup.staging.policies : policy.action == "KEEP" && policy.most_recent_versions != null && policy.most_recent_versions.keep_count != null])`,
	} {
		if !strings.Contains(segment, required) {
			t.Fatalf("stacks/dep-control/variables.tf does not bind the workload image cleanup lifecycle element %q", required)
		}
	}

	controlMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	for _, required := range []string{
		`cleanup_policies = var.workload_image_cleanup[each.key].policies`,
		`cleanup_policy_dry_run = var.workload_image_cleanup[each.key].dry_run`,
	} {
		if !strings.Contains(controlMain, required) {
			t.Fatalf("stacks/dep-control/main.tf does not wire the workload image cleanup lifecycle element %q", required)
		}
	}

	// Zone purity: no other stack ever carries the lifecycle binding.
	for _, stack := range []string{"dep-intake", "dep-evidence", "dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "cleanup_policies") || strings.Contains(main, "workload_image_cleanup") {
			t.Fatalf("stacks/%s must never bind the workload image cleanup lifecycle; only the control zone carries the workload image registries", stack)
		}
		variables := readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf"))
		if strings.Contains(variables, "workload_image_cleanup") {
			t.Fatalf("stacks/%s/variables.tf must never carry the workload image cleanup binding; it lives exactly once in dep-control", stack)
		}
	}

	// The behavioral proofs of the structural rules live beside the stack.
	fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tofutest.hcl")))
	for _, required := range []string{
		`workload_image_cleanup = {`,
		`run "accepts_the_canonical_cleanup_binding"`,
		`run "rejects_a_delete_policy_on_the_release_class"`,
		`run "rejects_a_staging_binding_without_the_time_delete"`,
		`run "rejects_a_staging_binding_without_the_keep_floor"`,
		`expect_failures = [var.workload_image_cleanup]`,
	} {
		if !strings.Contains(fixture, required) {
			t.Fatalf("stacks/dep-control/variables.tofutest.hcl does not carry the cleanup lifecycle proof element %q", required)
		}
	}

	// The documentation surfaces carry the lifecycle form: the module README
	// carries the class rule and the provider-schema proof, the stack README
	// carries the instance-bound input, the architecture decision record
	// carries the declaration decision and the register carries the ticket.
	moduleReadme := readRepositoryFile(t, filepath.Join("modules", "artifact-registry", "README.md"))
	for _, required := range []string{"append-only supply-chain records", "tofu providers schema", "7.44.0"} {
		if !strings.Contains(moduleReadme, required) {
			t.Fatalf("the artifact-registry module README does not document %q of the cleanup lifecycle", required)
		}
	}

	stackReadme := readRepositoryFile(t, filepath.Join("stacks", "dep-control", "README.md"))
	for _, required := range []string{"workload_image_cleanup", "cleanup", "dry-run"} {
		if !strings.Contains(stackReadme, required) {
			t.Fatalf("the dep-control stack README does not document %q of the cleanup lifecycle", required)
		}
	}

	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"workload image lifecycle",
		"cleanup",
		"dry-run",
		"append-only supply-chain records",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the workload image cleanup lifecycle element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-25") {
		t.Fatal("TRACEABILITY.md does not contain DAI-25")
	}
}

func TestControlStackBindsTheMandatoryDescriptionSurfaces(t *testing.T) {
	// The mandatory resource properties convention: every managed resource
	// declares the canonical human-readable description surface of its provider
	// schema. The network module exposes the five description surfaces of the
	// workload network origin as optional inputs (null when unbound); the
	// control stack binds them as required instance-supplied values — the
	// create-only VPC and subnetwork descriptions byte-exact to the live
	// values, the in-place surfaces as the canonical forms.
	moduleVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "variables.tf")))
	for _, required := range []string{
		`network_description = optional(string, null)`,
		`subnet_description = optional(string, null)`,
		`firewall_allow_description = optional(string, null)`,
		`firewall_deny_description = optional(string, null)`,
		`dns_policy_description = optional(string, null)`,
		`description == null || length(description) > 0`,
	} {
		if !strings.Contains(moduleVariables, required) {
			t.Fatalf("modules/network/variables.tf does not bind the mandatory description surface element %q", required)
		}
	}

	moduleMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "main.tf")))
	for _, required := range []string{
		`description = var.workload_network.network_description`,
		`description = var.workload_network.subnet_description`,
		`description = var.workload_network.firewall_allow_description`,
		`description = var.workload_network.firewall_deny_description`,
		`description = var.workload_network.dns_policy_description`,
	} {
		if !strings.Contains(moduleMain, required) {
			t.Fatalf("modules/network/main.tf does not wire the mandatory description surface %q", required)
		}
	}
	// Exactly the five managed surfaces of the workload network origin carry
	// the binding: the VPC, the subnetwork, the egress firewall pair and the
	// restricted-range DNS response policy; the DNS response policy rules
	// expose no description surface in the pinned provider schema (google
	// 7.44.0, proven through `tofu providers schema -json`), so the duty never
	// binds them.
	if count := strings.Count(moduleMain, "description = var.workload_network."); count != 5 {
		t.Fatalf("modules/network/main.tf carries %d description wirings of the workload network origin, want exactly 5", count)
	}

	// The behavioral proof of the description duty lives beside the module and
	// executes offline (the module carries no backend and no encryption block).
	moduleFixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "network", "variables.tofutest.hcl")))
	for _, required := range []string{
		`run "accepts_the_bound_description_surfaces"`,
		`run "rejects_an_empty_description"`,
		`expect_failures = [var.workload_network]`,
	} {
		if !strings.Contains(moduleFixture, required) {
			t.Fatalf("modules/network/variables.tofutest.hcl does not carry the description duty proof element %q", required)
		}
	}

	// The control stack binds the five surfaces as required instance-supplied
	// values — never optional, never a stack default — with the non-empty
	// validation bound fail-closed.
	controlVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tf")))
	start := strings.Index(controlVariables, `variable "workload_network" {`)
	if start < 0 {
		t.Fatal("stacks/dep-control/variables.tf does not carry the workload_network input")
	}
	segment := controlVariables[start:]
	if next := strings.Index(segment, ` variable "`); next > 0 {
		segment = segment[:next]
	}
	for _, required := range []string{
		`network_description = string`,
		`subnet_description = string`,
		`firewall_allow_description = string`,
		`firewall_deny_description = string`,
		`dns_policy_description = string`,
		`length(var.workload_network.network_description) > 0`,
		`length(var.workload_network.subnet_description) > 0`,
		`length(var.workload_network.firewall_allow_description) > 0`,
		`length(var.workload_network.firewall_deny_description) > 0`,
		`length(var.workload_network.dns_policy_description) > 0`,
	} {
		if !strings.Contains(segment, required) {
			t.Fatalf("stacks/dep-control/variables.tf does not bind the mandatory description surface element %q", required)
		}
	}
	if strings.Contains(segment, "optional(") {
		t.Fatal("stacks/dep-control/variables.tf carries an optional description surface of the workload network origin; the control zone binds every surface as a required instance-supplied value")
	}

	// The identity surfaces of the control zone: the trigger identity carries
	// its identity class name as the display name, and every controller
	// identity binds its display name and description as required values.
	identityMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "workload-identity", "main.tf")))
	triggerStart := strings.Index(identityMain, `resource "google_service_account" "trigger"`)
	if triggerStart < 0 {
		t.Fatal("modules/workload-identity/main.tf does not create the dedicated trigger identities")
	}
	triggerSegment := identityMain[triggerStart:]
	if next := strings.Index(triggerSegment, ` resource "`); next > 0 {
		triggerSegment = triggerSegment[:next]
	}
	if !strings.Contains(triggerSegment, `display_name = each.value.trigger_service_account_id`) {
		t.Fatal("modules/workload-identity/main.tf does not bind the trigger identity display name to its identity class name")
	}

	controllersStart := strings.Index(controlVariables, `variable "controllers" {`)
	if controllersStart < 0 {
		t.Fatal("stacks/dep-control/variables.tf does not carry the controllers input")
	}
	controllersSegment := controlVariables[controllersStart:]
	if next := strings.Index(controllersSegment, ` variable "`); next > 0 {
		controllersSegment = controllersSegment[:next]
	}
	for _, required := range []string{
		`display_name = string`,
		`description = string`,
		`length(controller.display_name) > 0 && length(controller.description) > 0`,
	} {
		if !strings.Contains(controllersSegment, required) {
			t.Fatalf("stacks/dep-control/variables.tf does not bind the controller description surface element %q", required)
		}
	}
	if strings.Contains(controllersSegment, `display_name = optional(`) || strings.Contains(controllersSegment, `description = optional(`) {
		t.Fatal("stacks/dep-control/variables.tf carries an optional controller display or description surface; every controller identity binds both as required values")
	}

	// The control stack wires the pool and audit sink description surfaces.
	controlMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	for _, required := range []string{
		`pool_display_name = var.pool_id`,
		`pool_description = local.workload_identity_pool_description`,
		`description = local.audit_sink_description`,
	} {
		if !strings.Contains(controlMain, required) {
			t.Fatalf("stacks/dep-control/main.tf does not wire the description surface %q", required)
		}
	}

	// The behavioral proofs of the stack bindings live beside the code.
	fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tofutest.hcl")))
	for _, required := range []string{
		`network_description = "Zone workload network origin."`,
		`run "accepts_the_description_surface_bindings"`,
		`run "rejects_an_empty_network_description"`,
		`run "rejects_an_identity_without_the_description_surfaces"`,
		`expect_failures = [var.controllers]`,
	} {
		if !strings.Contains(fixture, required) {
			t.Fatalf("stacks/dep-control/variables.tofutest.hcl does not carry the description duty proof element %q", required)
		}
	}

	// The documentation surfaces carry the duty: the module and stack READMEs,
	// the architecture decision record and the register.
	moduleReadme := readRepositoryFile(t, filepath.Join("modules", "network", "README.md"))
	for _, required := range []string{"mandatory resource properties convention", "create-only", "byte-exact"} {
		if !strings.Contains(moduleReadme, required) {
			t.Fatalf("the network module README does not document %q of the mandatory description duty", required)
		}
	}
	stackReadme := readRepositoryFile(t, filepath.Join("stacks", "dep-control", "README.md"))
	for _, required := range []string{"mandatory resource properties convention", "byte-exact", "display name and description"} {
		if !strings.Contains(stackReadme, required) {
			t.Fatalf("the dep-control stack README does not document %q of the mandatory description duty", required)
		}
	}

	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"mandatory resource properties convention",
		"create-only",
		"byte-exact",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the mandatory description duty element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-27") {
		t.Fatal("TRACEABILITY.md does not contain DAI-27")
	}
}

func TestControlStackBindsTheWorkloadJobEnvOwnership(t *testing.T) {
	// The workload configuration ownership convention: the declaration owns
	// every static, non-credential configuration value of a workload
	// completely. The control stack consumes the instance-bound
	// workload_job_env input — never a stack default — and wires it into the
	// job module; every other stack is pure.
	controlVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tf")))
	start := strings.Index(controlVariables, `variable "workload_job_env" {`)
	if start < 0 {
		t.Fatal("stacks/dep-control/variables.tf does not carry the workload_job_env input")
	}
	segment := controlVariables[start:]
	if next := strings.Index(segment, ` variable "`); next > 0 {
		segment = segment[:next]
	}
	if strings.Contains(segment, "default") {
		t.Fatal("stacks/dep-control/variables.tf carries a default for workload_job_env; the static configuration is an instance binding, never a stack default")
	}
	for _, required := range []string{
		`type = map(map(string))`,
		`length(setsubtract(keys(var.workload_job_env), keys(local.workload_jobs))) == 0`,
		`can(regex("^[A-Z][A-Z0-9_]*$", key))`,
		`!can(regex("(?i)(password|secret|token|credential|api[_-]?key|private[_-]?key)", key))`,
		`!can(regex("(?i)(password|secret|token|credential|api[_-]?key|private[_-]?key)", value))`,
	} {
		if !strings.Contains(segment, required) {
			t.Fatalf("stacks/dep-control/variables.tf does not bind the workload job env ownership element %q", required)
		}
	}

	controlMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "main.tf")))
	if !strings.Contains(controlMain, `env = lookup(var.workload_job_env, each.key, {})`) {
		t.Fatal("stacks/dep-control/main.tf does not wire the instance-bound env binding into the workload jobs")
	}

	// Zone purity: no other stack ever carries the binding.
	for _, stack := range []string{"dep-intake", "dep-evidence", "dep-approved", "dep-quarantine"} {
		main := readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf"))
		if strings.Contains(main, "workload_job_env") {
			t.Fatalf("stacks/%s must never bind the workload job env surface; it lives exactly once in dep-control", stack)
		}
		variables := readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf"))
		if strings.Contains(variables, "workload_job_env") {
			t.Fatalf("stacks/%s/variables.tf must never carry the workload_job_env binding; it lives exactly once in dep-control", stack)
		}
	}

	// The behavioral proofs live beside the stack.
	fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", "dep-control", "variables.tofutest.hcl")))
	for _, required := range []string{
		`workload_job_env = {`,
		`run "accepts_the_canonical_workload_job_env_binding"`,
		`run "rejects_an_unknown_job_env_binding"`,
		`run "rejects_a_credential_carrying_env_binding"`,
		`expect_failures = [var.workload_job_env]`,
	} {
		if !strings.Contains(fixture, required) {
			t.Fatalf("stacks/dep-control/variables.tofutest.hcl does not carry the workload job env proof element %q", required)
		}
	}

	// The documentation surfaces carry the ownership form: the stack README,
	// the architecture decision record and the register.
	stackReadme := readRepositoryFile(t, filepath.Join("stacks", "dep-control", "README.md"))
	for _, required := range []string{"workload_job_env", "configuration ownership", "never a stack default"} {
		if !strings.Contains(stackReadme, required) {
			t.Fatalf("the dep-control stack README does not document %q of the workload job env ownership", required)
		}
	}

	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"workload configuration ownership",
		"workload_job_env",
		"never a stack default",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the workload job env ownership element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-28") {
		t.Fatal("TRACEABILITY.md does not contain DAI-28")
	}
}

func TestEveryStackBindsTheInstanceBoundProjectNumber(t *testing.T) {
	// The project reference form discipline (DAI-30, completed by DAI-33):
	// every project-referencing resource binds the form the platform state
	// carries for its class — the number-addressed classes (the workload
	// identity pool, its providers and every organization policy) bind the
	// instance-bound numeric project number, and the ID-addressed classes keep
	// the project ID. Binding the project ID on a number-addressed class
	// forces a destroy-and-recreate of the live resource at the convergence
	// window. This guard pins the complete assignment matrix fail-closed, per
	// resource class and across every module.
	moduleVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "workload-identity", "variables.tf")))
	start := strings.Index(moduleVariables, `variable "project_number" {`)
	if start < 0 {
		t.Fatal("modules/workload-identity/variables.tf does not carry the project_number input")
	}
	segment := moduleVariables[start:]
	if next := strings.Index(segment, ` variable "`); next > 0 {
		segment = segment[:next]
	}
	if strings.Contains(segment, "default") {
		t.Fatal("modules/workload-identity/variables.tf carries a default for project_number; the value is an instance binding, never a default")
	}
	if !strings.Contains(segment, `can(regex("^[0-9]+$", var.project_number))`) {
		t.Fatal("modules/workload-identity/variables.tf does not bind the numeric validation of project_number")
	}

	// The workload-identity module matrix: the pool and the providers bind the
	// number; both service account families and the identity role bindings
	// keep the project ID.
	moduleMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("modules", "workload-identity", "main.tf")))
	for _, class := range []struct {
		marker  string
		binding string
	}{
		{`resource "google_iam_workload_identity_pool" "this"`, `project = var.project_number`},
		{`resource "google_iam_workload_identity_pool_provider" "this"`, `project = var.project_number`},
		{`resource "google_service_account" "this"`, `project = var.project_id`},
		{`resource "google_service_account" "trigger"`, `project = var.project_id`},
		{`resource "google_project_iam_member" "identity_roles"`, `project = var.project_id`},
	} {
		segment := segmentFromMarker(t, moduleMain, class.marker, ` resource "`, "modules/workload-identity/main.tf")
		if !strings.Contains(segment, class.binding) {
			t.Fatalf("modules/workload-identity/main.tf does not bind %q on %q", class.binding, class.marker)
		}
	}
	if count := strings.Count(moduleMain, `project = var.project_number`); count != 2 {
		t.Fatalf("modules/workload-identity/main.tf binds the project number %d times, want exactly 2 (the pool and the providers)", count)
	}
	if count := strings.Count(moduleMain, `project = var.project_id`); count != 3 {
		t.Fatalf("modules/workload-identity/main.tf keeps the project ID on %d resources, want exactly 3 (both service account families and the identity roles)", count)
	}

	// The policy-bindings module: every organization policy is
	// number-addressed, and the module never references the project ID.
	policyVariables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("policy-bindings", "variables.tf")))
	if strings.Contains(policyVariables, `variable "project_id"`) {
		t.Fatal("policy-bindings/variables.tf still carries the project ID input; every organization policy is number-addressed")
	}
	policyNumberInput := segmentFromMarker(t, policyVariables, `variable "project_number" {`, ` variable "`, "policy-bindings/variables.tf")
	if strings.Contains(policyNumberInput, "default") {
		t.Fatal("policy-bindings/variables.tf carries a default for project_number; the value is an instance binding, never a default")
	}
	if !strings.Contains(policyNumberInput, `can(regex("^[0-9]+$", var.project_number))`) {
		t.Fatal("policy-bindings/variables.tf does not bind the numeric validation of project_number")
	}

	policyMain := normalizeWhitespace(readRepositoryFile(t, filepath.Join("policy-bindings", "main.tf")))
	if strings.Contains(policyMain, "var.project_id") {
		t.Fatal("policy-bindings/main.tf still references the project ID; every organization policy binds the instance-bound project number")
	}
	if count := strings.Count(policyMain, "projects/${var.project_number}"); count != 6 {
		t.Fatalf("policy-bindings/main.tf binds the project number on %d surfaces, want exactly 6 (the name and parent of every organization policy resource)", count)
	}
	for _, marker := range []string{
		`resource "google_org_policy_policy" "this"`,
		`resource "google_org_policy_policy" "cloud_run_vpc_egress"`,
		`resource "google_org_policy_policy" "cloud_run_ingress"`,
	} {
		segment := segmentFromMarker(t, policyMain, marker, ` resource "`, "policy-bindings/main.tf")
		if !strings.Contains(segment, `name = "projects/${var.project_number}/policies/`) {
			t.Fatalf("policy-bindings/main.tf does not bind the number form in the name of %q", marker)
		}
		if !strings.Contains(segment, `parent = "projects/${var.project_number}"`) {
			t.Fatalf("policy-bindings/main.tf does not bind the number form in the parent of %q", marker)
		}
	}

	// The negative proof across every other module: the number-addressed
	// classes live exactly in the workload-identity and policy-bindings
	// modules; no other module ever references the project number.
	for _, module := range moduleNames {
		if module == "workload-identity" {
			continue
		}
		for _, file := range []string{"main.tf", "variables.tf"} {
			if content := readRepositoryFile(t, filepath.Join("modules", module, file)); strings.Contains(content, "project_number") {
				t.Fatalf("modules/%s/%s references the project number; the number-addressed classes live exactly in the workload-identity and policy-bindings modules", module, file)
			}
		}
	}

	for _, stack := range stackNames {
		variables := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tf")))
		stackStart := strings.Index(variables, `variable "project_number" {`)
		if stackStart < 0 {
			t.Fatalf("stacks/%s/variables.tf does not carry the project_number input", stack)
		}
		stackSegment := variables[stackStart:]
		if next := strings.Index(stackSegment, ` variable "`); next > 0 {
			stackSegment = stackSegment[:next]
		}
		if strings.Contains(stackSegment, "default") {
			t.Fatalf("stacks/%s/variables.tf carries a default for project_number; the value is an instance binding, never a default", stack)
		}
		if !strings.Contains(stackSegment, `can(regex("^[0-9]+$", var.project_number))`) {
			t.Fatalf("stacks/%s/variables.tf does not bind the numeric validation of project_number", stack)
		}

		main := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "main.tf")))
		moduleStart := strings.Index(main, `module "workload_identity" {`)
		if moduleStart < 0 {
			t.Fatalf("stacks/%s/main.tf does not call the workload-identity module", stack)
		}
		moduleSegment := main[moduleStart:]
		if next := strings.Index(moduleSegment, ` module "`); next > 0 {
			moduleSegment = moduleSegment[:next]
		}
		if !strings.Contains(moduleSegment, `project_number = var.project_number`) {
			t.Fatalf("stacks/%s/main.tf does not wire the instance-bound project number into the workload identity module", stack)
		}
		policySegment := segmentFromMarker(t, main, `module "policy_bindings" {`, ` module "`, "stacks/"+stack+"/main.tf")
		if !strings.Contains(policySegment, `project_number = var.project_number`) {
			t.Fatalf("stacks/%s/main.tf does not wire the instance-bound project number into the policy bindings module", stack)
		}
		if strings.Contains(policySegment, "project_id") {
			t.Fatalf("stacks/%s/main.tf still wires the project ID into the policy bindings module; every organization policy is number-addressed", stack)
		}
		if count := strings.Count(main, "project_number = var.project_number"); count != 2 {
			t.Fatalf("stacks/%s/main.tf wires the project number %d times, want exactly 2 (the workload identity and the policy bindings module calls)", stack, count)
		}

		// The behavioral proofs live beside the stack: the synthetic binding
		// and the rejection run of a non-numeric value.
		fixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("stacks", stack, "variables.tofutest.hcl")))
		if !strings.Contains(fixture, `project_number = "`) {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the synthetic project number binding", stack)
		}
		if !strings.Contains(fixture, `run "rejects_a_non_numeric_project_number"`) {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the rejection run of a non-numeric project number", stack)
		}
		if !strings.Contains(fixture, "expect_failures = [var.project_number]") {
			t.Fatalf("stacks/%s/variables.tofutest.hcl does not carry the rejection proof of the project number binding", stack)
		}

		readme := readRepositoryFile(t, filepath.Join("stacks", stack, "README.md"))
		if !strings.Contains(readme, "project_number") {
			t.Fatalf("the %s stack README does not document the project_number input", stack)
		}
	}

	// The module-level behavioral proof of the policy-bindings validation lives
	// beside the module and executes offline (the module carries no backend and
	// no encryption block).
	policyFixture := normalizeWhitespace(readRepositoryFile(t, filepath.Join("policy-bindings", "variables.tofutest.hcl")))
	for _, required := range []string{
		`run "accepts_a_numeric_project_number"`,
		`run "rejects_a_non_numeric_project_number"`,
		"expect_failures = [var.project_number]",
	} {
		if !strings.Contains(policyFixture, required) {
			t.Fatalf("policy-bindings/variables.tofutest.hcl does not carry the project number proof element %q", required)
		}
	}

	// The module documentation carries the binding form and its rationale.
	moduleReadme := readRepositoryFile(t, filepath.Join("modules", "workload-identity", "README.md"))
	for _, required := range []string{"project_number", "destroy", "project ID"} {
		if !strings.Contains(moduleReadme, required) {
			t.Fatalf("the workload-identity module README does not document %q of the project number binding", required)
		}
	}
	policyReadme := readRepositoryFile(t, filepath.Join("policy-bindings", "README.md"))
	if !strings.Contains(policyReadme, "project_number") {
		t.Fatal("the policy-bindings module README does not document the project number binding")
	}
	if strings.Contains(policyReadme, "project_id") {
		t.Fatal("the policy-bindings module README still references the project ID input; every organization policy is number-addressed")
	}

	// The architecture decision record carries the completed project number
	// decision including the rejected data-source form.
	adr := normalizeWhitespace(readRepositoryFile(t, filepath.Join("docs", "architecture", "ADR-0001-DEPENDENCY-AUTHORITY-INFRASTRUCTURE.md")))
	for _, required := range []string{
		"project number",
		"project_number",
		"destroy",
		`data "google_project"`,
		"organization policies",
	} {
		if !strings.Contains(adr, required) {
			t.Fatalf("ADR-0001 does not carry the project number decision element %q", required)
		}
	}

	traceability := readRepositoryFile(t, filepath.Join("docs", "TRACEABILITY.md"))
	if !strings.Contains(traceability, "DAI-30") {
		t.Fatal("TRACEABILITY.md does not contain DAI-30")
	}
	if !strings.Contains(traceability, "DAI-33") {
		t.Fatal("TRACEABILITY.md does not contain DAI-33")
	}
}

func TestRepositoryFilesSkipTheLocalWorkingForms(t *testing.T) {
	// The bound local working forms are never repository content: the
	// convergence-window import carrier (the git-ignored window.imports.tf)
	// and the engine override forms are local working surfaces, removed after
	// their proof — a paused convergence window never fails the guards.
	for _, name := range []string{"window.imports.tf", "override.tf", "override.tf.json", "example_override.tf", "example_override.tf.json"} {
		if !isLocalWorkingFile(name) {
			t.Fatalf("isLocalWorkingFile(%q) = false, want true (the bound local working form)", name)
		}
	}
	for _, name := range []string{"main.tf", "variables.tf", "outputs.tf", "versions.tf", "window.tf", "window.imports.tf.json"} {
		if isLocalWorkingFile(name) {
			t.Fatalf("isLocalWorkingFile(%q) = true, want false (bound repository content)", name)
		}
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

// segmentFromMarker returns the normalized content from the block marker up
// to the next top-level block marker of the same kind (or the end of the
// content), so a guard asserts on exactly one resource, module or variable
// block; a missing marker fails the test with the named surface.
func segmentFromMarker(t *testing.T, content string, marker string, nextMarker string, surface string) string {
	t.Helper()
	start := strings.Index(content, marker)
	if start < 0 {
		t.Fatalf("%s does not carry %q", surface, marker)
	}
	segment := content[start:]
	if next := strings.Index(segment, nextMarker); next > 0 {
		segment = segment[:next]
	}
	return segment
}

// isLocalWorkingFile reports the bound local working forms that are never
// repository content: the git-ignored convergence-window import carrier
// (window.imports.tf) and the engine's override forms (override.tf,
// *_override.tf and their JSON variants). The guards evaluate the bound
// repository content only.
func isLocalWorkingFile(name string) bool {
	switch name {
	case "window.imports.tf", "override.tf", "override.tf.json":
		return true
	}
	return strings.HasSuffix(name, "_override.tf") || strings.HasSuffix(name, "_override.tf.json")
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
		// Local working files are never repository content: the git-ignored
		// convergence-window import carrier and the engine's override forms
		// carry local working values and are removed after their proof.
		if isLocalWorkingFile(entry.Name()) {
			return nil
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
