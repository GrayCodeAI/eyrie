package eyrie

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GrayCodeAI/eyrie/client"
)

func TestVersionNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version should not be empty; check that the VERSION file is embedded")
	}
}

func TestVersionMatchesSemver(t *testing.T) {
	// Expect a semver-like pattern: major.minor.patch with optional pre-release/build metadata
	re := regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$`)
	if !re.MatchString(Version) {
		t.Errorf("Version %q does not match semver pattern", Version)
	}
}

func TestVersionIsTrimmed(t *testing.T) {
	if strings.TrimSpace(Version) != Version {
		t.Errorf("Version contains leading/trailing whitespace: %q", Version)
	}
}

func TestVersionFromEmbedFile(t *testing.T) {
	// The versionFile variable is the raw embedded content; Version should be the trimmed form
	raw := strings.TrimSpace(versionFile)
	if raw != Version {
		t.Errorf("Version (%q) should equal trimmed versionFile (%q)", Version, raw)
	}
}

func TestVersionFileNotEmpty(t *testing.T) {
	if strings.TrimSpace(versionFile) == "" {
		t.Fatal("Embedded versionFile is empty; the VERSION file may be missing")
	}
}

func TestVersionPropagatedToClient(t *testing.T) {
	// The init() in this package calls client.SetVersion(Version).
	// After package init, client.Version should match.
	if client.Version != Version {
		t.Errorf("client.Version = %q, want %q", client.Version, Version)
	}
}

func TestClientSetVersionDirectly(t *testing.T) {
	original := client.Version
	defer client.SetVersion(original)

	client.SetVersion("1.2.3-test")
	if client.Version != "1.2.3-test" {
		t.Errorf("client.Version = %q after SetVersion, want %q", client.Version, "1.2.3-test")
	}
}

func TestClientSetVersionEmpty(t *testing.T) {
	original := client.Version
	defer client.SetVersion(original)

	client.SetVersion("")
	if client.Version != "" {
		t.Errorf("client.Version = %q after SetVersion(empty), want empty", client.Version)
	}
}

func TestVersionStartsWithV0(t *testing.T) {
	// The initial version is 0.2.0; verify it starts with 0.
	if !strings.HasPrefix(Version, "0.") {
		t.Errorf("expected Version to start with '0.', got %q", Version)
	}
}
