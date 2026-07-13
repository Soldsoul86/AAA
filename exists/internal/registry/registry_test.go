package registry

import "testing"

// These are real network integration tests against the live npm and PyPI
// registries — deliberate. Mocking the HTTP layer would only test that the
// mock behaves as assumed, not whether the real registries actually behave
// the way this package assumes; the entire point of "exists" is whether a
// package is real, so the test needs to ask the real registry the same
// question the tool does.

func TestExistsNPMRealPackage(t *testing.T) {
	exists, ok := Exists(NPM, "lodash")
	if !ok {
		t.Fatal("checkedOK = false, want true (registry should be reachable)")
	}
	if !exists {
		t.Error("exists = false for a well-known real package (lodash)")
	}
}

func TestExistsNPMScopedRealPackage(t *testing.T) {
	exists, ok := Exists(NPM, "@babel/core")
	if !ok {
		t.Fatal("checkedOK = false, want true")
	}
	if !exists {
		t.Error("exists = false for a well-known real scoped package (@babel/core)")
	}
}

func TestExistsNPMFakePackage(t *testing.T) {
	exists, ok := Exists(NPM, "this-package-definitely-does-not-exist-xyz123-actually-tool")
	if !ok {
		t.Fatal("checkedOK = false, want true")
	}
	if exists {
		t.Error("exists = true for a package name that should not exist")
	}
}

func TestExistsPyPIRealPackage(t *testing.T) {
	exists, ok := Exists(PyPI, "requests")
	if !ok {
		t.Fatal("checkedOK = false, want true")
	}
	if !exists {
		t.Error("exists = false for a well-known real package (requests)")
	}
}

func TestExistsPyPIFakePackage(t *testing.T) {
	exists, ok := Exists(PyPI, "this-package-definitely-does-not-exist-xyz123-actually-tool")
	if !ok {
		t.Fatal("checkedOK = false, want true")
	}
	if exists {
		t.Error("exists = true for a package name that should not exist")
	}
}

func TestExistsUnknownEcosystem(t *testing.T) {
	_, ok := Exists(Ecosystem("cobol-cpan"), "whatever")
	if ok {
		t.Error("checkedOK = true for an unsupported ecosystem, want false")
	}
}
