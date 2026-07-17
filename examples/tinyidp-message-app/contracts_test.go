package main

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestExternalImportBoundary(t *testing.T) {
	loaded, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedImports}, ".")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(loaded) != 0 {
		t.Fatal("load example package imports")
	}
	for _, pkg := range loaded {
		for path := range pkg.Imports {
			if strings.HasPrefix(path, "github.com/go-go-golems/tiny-idp/internal/") {
				t.Errorf("%s imports forbidden internal package %q", pkg.PkgPath, path)
			}
		}
	}
}

func TestImplementationContractIsUnambiguous(t *testing.T) {
	assertUnique := func(label string, values []string) {
		t.Helper()
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				t.Errorf("%s contains an empty value", label)
			}
			if _, duplicate := seen[value]; duplicate {
				t.Errorf("%s contains duplicate %q", label, value)
			}
			seen[value] = struct{}{}
		}
	}
	assertUnique("route contract", routeContract)
	assertUnique("security invariant inventory", securityInvariantTests)
}
