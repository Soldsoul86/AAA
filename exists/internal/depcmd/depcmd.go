// Package depcmd detects shell commands that install a new dependency and
// extracts the package name(s) being added, for the ecosystems with the
// best-evidenced hallucinated-package harm: npm and PyPI. A precise,
// known-command list is used rather than a loose heuristic — same posture
// as testrun.IsTestCommand in the actually tool — so "npm install" with no
// arguments (installs from package.json, adds nothing new) or "pip install
// -r requirements.txt" (installs from a file, not a named package) are
// correctly not treated as adding a new dependency.
package depcmd

import "strings"

type Ecosystem string

const (
	NPM  Ecosystem = "npm"
	PyPI Ecosystem = "pypi"
)

// installVerbs maps each recognized "<tool> <verb>" prefix to the
// ecosystem it installs into.
var installVerbs = map[string]Ecosystem{
	"npm install":  NPM,
	"npm i ":       NPM, // trailing space: distinguishes from "npm install" handled above and avoids matching "npm init"
	"yarn add":     NPM,
	"pnpm add":     NPM,
	"pip install":  PyPI,
	"pip3 install": PyPI,
	"poetry add":   PyPI,
	"uv add":       PyPI,
}

// Packages extracts the base package name(s) a command installs, if it
// recognizably is an install command with at least one named package.
// Returns ok=false for anything else, including a recognized install verb
// with no package arguments (e.g. bare "npm install", which installs from
// the existing manifest rather than adding something new).
func Packages(command string) (eco Ecosystem, packages []string, ok bool) {
	trimmed := strings.TrimSpace(command)
	for verb, e := range installVerbs {
		var rest string
		if strings.HasPrefix(trimmed, verb) {
			rest = trimmed[len(verb):]
		} else {
			continue
		}
		pkgs := extractPackageArgs(rest, e)
		if len(pkgs) == 0 {
			return "", nil, false
		}
		return e, pkgs, true
	}
	return "", nil, false
}

// valueFlags lists flags that consume the *next* token as their value
// rather than being a standalone boolean — without this, something like
// "poetry add --group dev fastapi" would misparse "dev" as a second
// package name. Not exhaustive; uncommon flags outside this list are a
// known, documented limitation.
var valueFlags = map[Ecosystem]map[string]bool{
	NPM: {
		"--registry": true, "--tag": true, "--scope": true,
		"--workspace": true, "-w": true,
	},
	PyPI: {
		"--target": true, "-t": true, "--index-url": true, "-i": true,
		"--extra-index-url": true, "--group": true, "-G": true,
		"--python": true, "-p": true, "--find-links": true, "-f": true,
		"--source": true,
	},
}

// extractPackageArgs tokenizes the remainder of an install command and
// keeps only tokens that look like package names: not a flag (doesn't
// start with "-"), not the value belonging to a preceding value-taking
// flag, and not an obvious non-package argument like a requirements file
// path or a local directory.
func extractPackageArgs(rest string, eco Ecosystem) []string {
	var pkgs []string
	skipNext := false
	for _, tok := range strings.Fields(rest) {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(tok, "-") {
			if valueFlags[eco][tok] {
				skipNext = true
			}
			continue
		}
		if eco == PyPI && (strings.HasSuffix(tok, ".txt") || strings.HasSuffix(tok, ".toml") || strings.HasSuffix(tok, ".cfg")) {
			// e.g. "-r requirements.txt" or "-e ." style local/file installs.
			continue
		}
		if tok == "." || tok == ".." {
			continue
		}
		name := stripVersion(tok, eco)
		if name == "" {
			continue
		}
		pkgs = append(pkgs, name)
	}
	return pkgs
}

// stripVersion removes a trailing version/extras specifier so
// "lodash@4.17.21" -> "lodash", "@babel/core@7.20.0" -> "@babel/core", and
// "requests==2.31.0" -> "requests", while leaving a scoped npm package with
// no version ("@babel/core") untouched.
func stripVersion(tok string, eco Ecosystem) string {
	switch eco {
	case NPM:
		// A leading "@" is the scope marker, not a version separator —
		// only split on an "@" that isn't the first character.
		if i := strings.LastIndex(tok, "@"); i > 0 {
			return tok[:i]
		}
		return tok
	case PyPI:
		for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<"} {
			if i := strings.Index(tok, sep); i > 0 {
				return tok[:i]
			}
		}
		// Extras syntax: "requests[security]" -> "requests".
		if i := strings.Index(tok, "["); i > 0 {
			return tok[:i]
		}
		return tok
	default:
		return tok
	}
}
