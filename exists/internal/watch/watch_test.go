package watch

import "testing"

// Real network calls, deliberately — same reasoning as registry's own tests.

func TestCheckRealPackageProducesNoFinding(t *testing.T) {
	findings := Check("npm install lodash")
	if findings != nil {
		t.Fatalf("got %v, want nil for a real, existing package", findings)
	}
}

func TestCheckHallucinatedPackageIsFlaggedMissing(t *testing.T) {
	findings := Check("pip install this-package-definitely-does-not-exist-xyz123-actually-tool")
	if len(findings) != 1 || findings[0].Status != Missing {
		t.Fatalf("got %+v, want one Missing finding", findings)
	}
}

func TestCheckNonInstallCommandProducesNoFinding(t *testing.T) {
	if findings := Check("go build ./..."); findings != nil {
		t.Fatalf("got %v, want nil for a non-install command", findings)
	}
}

func TestCheckMultiplePackagesOnlyFlagsTheMissingOne(t *testing.T) {
	findings := Check("pip install requests this-package-definitely-does-not-exist-xyz123-actually-tool")
	if len(findings) != 1 || findings[0].Package != "this-package-definitely-does-not-exist-xyz123-actually-tool" {
		t.Fatalf("got %+v, want exactly one finding for the fake package only", findings)
	}
}
