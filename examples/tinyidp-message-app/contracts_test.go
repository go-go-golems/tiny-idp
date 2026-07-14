package main

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

func TestExternalImportBoundary(t *testing.T) {
	packages, err := parser.ParseDir(token.NewFileSet(), ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		for filename, file := range pkg.Files {
			for _, imported := range file.Imports {
				path, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					t.Fatalf("%s has malformed import %q", filename, imported.Path.Value)
				}
				if strings.HasPrefix(path, "github.com/manuel/tinyidp/internal/") {
					t.Errorf("%s imports forbidden internal package %q", filename, path)
				}
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
