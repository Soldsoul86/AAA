// Package watch ties depcmd and registry together: given one shell
// command, determine whether it installs a dependency and, if so, whether
// every package it names actually exists.
package watch

import (
	"github.com/Soldsoul86/AAA/exists/internal/depcmd"
	"github.com/Soldsoul86/AAA/exists/internal/registry"
)

type Status string

const (
	Missing      Status = "missing"      // checked the real registry; it doesn't exist
	Unverifiable Status = "unverifiable" // the check itself failed (network, timeout, etc.)
)

type Finding struct {
	Ecosystem depcmd.Ecosystem
	Package   string
	Status    Status
}

// Check inspects one shell command. It returns nil if the command isn't a
// recognized dependency-install command, or one Finding per package that
// isn't confirmed to exist. A package that's confirmed to exist produces
// no finding — Check only speaks up when something doesn't add up, same
// posture as actually's claim verification.
func Check(command string) []Finding {
	eco, packages, ok := depcmd.Packages(command)
	if !ok {
		return nil
	}
	var findings []Finding
	for _, pkg := range packages {
		exists, checkedOK := registry.Exists(eco, pkg)
		switch {
		case checkedOK && !exists:
			findings = append(findings, Finding{Ecosystem: eco, Package: pkg, Status: Missing})
		case !checkedOK:
			findings = append(findings, Finding{Ecosystem: eco, Package: pkg, Status: Unverifiable})
		}
	}
	return findings
}
