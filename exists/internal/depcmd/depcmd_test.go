package depcmd

import (
	"reflect"
	"testing"
)

func TestPackagesNPM(t *testing.T) {
	cases := []struct {
		cmd  string
		want []string
	}{
		{"npm install lodash", []string{"lodash"}},
		{"npm i lodash", []string{"lodash"}},
		{"npm install lodash@4.17.21", []string{"lodash"}},
		{"npm install @babel/core", []string{"@babel/core"}},
		{"npm install @babel/core@7.20.0", []string{"@babel/core"}},
		{"npm install --save-dev typescript", []string{"typescript"}},
		{"npm install -g typescript", []string{"typescript"}},
		{"yarn add axios", []string{"axios"}},
		{"pnpm add -D vitest", []string{"vitest"}},
		{"npm install lodash axios", []string{"lodash", "axios"}},
	}
	for _, c := range cases {
		eco, got, ok := Packages(c.cmd)
		if !ok {
			t.Errorf("Packages(%q) ok=false, want true", c.cmd)
			continue
		}
		if eco != NPM {
			t.Errorf("Packages(%q) eco=%v, want NPM", c.cmd, eco)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Packages(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestPackagesPyPI(t *testing.T) {
	cases := []struct {
		cmd  string
		want []string
	}{
		{"pip install requests", []string{"requests"}},
		{"pip3 install requests", []string{"requests"}},
		{"pip install requests==2.31.0", []string{"requests"}},
		{"pip install requests>=2.0", []string{"requests"}},
		{"pip install requests[security]", []string{"requests"}},
		{"poetry add fastapi", []string{"fastapi"}},
		{"uv add pydantic", []string{"pydantic"}},
		{"pip install requests flask", []string{"requests", "flask"}},
	}
	for _, c := range cases {
		eco, got, ok := Packages(c.cmd)
		if !ok {
			t.Errorf("Packages(%q) ok=false, want true", c.cmd)
			continue
		}
		if eco != PyPI {
			t.Errorf("Packages(%q) eco=%v, want PyPI", c.cmd, eco)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Packages(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

// The real bug caught during development: a value-taking flag's argument
// getting misparsed as a second package name.
func TestPackagesValueFlagArgumentNotMisparsedAsPackage(t *testing.T) {
	cases := []struct {
		cmd  string
		want []string
	}{
		{"poetry add --group dev fastapi", []string{"fastapi"}},
		{"pip install --index-url https://example.com/simple requests", []string{"requests"}},
		{"npm install --registry https://registry.example.com lodash", []string{"lodash"}},
	}
	for _, c := range cases {
		_, got, ok := Packages(c.cmd)
		if !ok {
			t.Errorf("Packages(%q) ok=false, want true", c.cmd)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Packages(%q) = %v, want %v (flag value must not be treated as a package)", c.cmd, got, c.want)
		}
	}
}

func TestPackagesNoNewPackage(t *testing.T) {
	cases := []string{
		"npm install",
		"npm ci",
		"npm run test",
		"pip install -r requirements.txt",
		"pip install -e .",
		"yarn install",
		"go get github.com/foo/bar",
		"git commit -m 'add package'",
		"echo npm install lodash", // not actually an install command
	}
	for _, c := range cases {
		_, got, ok := Packages(c)
		if ok {
			t.Errorf("Packages(%q) = %v, ok=true, want ok=false (no new package being added)", c, got)
		}
	}
}
