// Package registry checks whether a package name actually exists in a
// public registry, via real HTTP calls — confirmed directly against both
// registries before writing this: a plain GET to
// https://registry.npmjs.org/<pkg> (unencoded "/" works fine for scoped
// names like "@babel/core") and https://pypi.org/pypi/<pkg>/json return a
// clean 200 for a real package and 404 for one that doesn't exist, no
// authentication or request body needed either way.
package registry

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Soldsoul86/AAA/exists/internal/depcmd"
)

type Ecosystem = depcmd.Ecosystem

const (
	NPM  = depcmd.NPM
	PyPI = depcmd.PyPI
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

// Exists checks whether pkg exists in the given ecosystem's public
// registry. checkedOK is false if the check itself failed — network error,
// timeout, or an unexpected status code — in which case exists should not
// be trusted either way; a failed check is deliberately not treated as
// "doesn't exist."
func Exists(eco Ecosystem, pkg string) (exists bool, checkedOK bool) {
	reqURL, ok := buildURL(eco, pkg)
	if !ok {
		return false, false
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return false, false
	}
	req.Header.Set("User-Agent", "exists (github.com/Soldsoul86/AAA/exists) - checking whether an AI agent hallucinated a package name")

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, true
	case http.StatusNotFound:
		return false, true
	default:
		return false, false
	}
}

func buildURL(eco Ecosystem, pkg string) (string, bool) {
	switch eco {
	case NPM:
		// Scoped packages ("@scope/name") need each segment escaped
		// separately — escaping the whole string would also encode the
		// "/" the registry expects literally.
		if scope, name, found := strings.Cut(pkg, "/"); found {
			return "https://registry.npmjs.org/" + url.PathEscape(scope) + "/" + url.PathEscape(name), true
		}
		return "https://registry.npmjs.org/" + url.PathEscape(pkg), true
	case PyPI:
		return "https://pypi.org/pypi/" + url.PathEscape(pkg) + "/json", true
	default:
		return "", false
	}
}
