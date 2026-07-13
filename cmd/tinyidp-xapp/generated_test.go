package main

import (
	"strings"
	"testing"

	"github.com/manuel/tinyidp/cmd/tinyidp-xapp/internal/xgojaruntime"
)

func TestGeneratedBundleContainsActorBoundProductSurface(t *testing.T) {
	bundle, err := xgojaruntime.NewBundle(xgojaruntime.Options{})
	if err != nil {
		t.Fatalf("NewBundle: %v", err)
	}
	declarations, err := bundle.TypeScriptDeclarations()
	if err != nil {
		t.Fatalf("TypeScriptDeclarations: %v", err)
	}
	for _, required := range []string{
		`declare module "durableobjects"`,
		"rpcForActor",
		"fetchForActor",
		"fetch(namespace",
		`declare module "express"`,
		`declare module "fs:assets"`,
	} {
		if !strings.Contains(declarations, required) {
			t.Errorf("generated declarations do not contain %q", required)
		}
	}
	plan, err := xgojaruntime.DecodeRuntimePlan()
	if err != nil {
		t.Fatalf("DecodeRuntimePlan: %v", err)
	}
	if plan.Name != "tinyidp-xapp" || len(plan.Runtime.Modules) != 4 {
		t.Fatalf("unexpected runtime plan: name=%q modules=%d", plan.Name, len(plan.Runtime.Modules))
	}
}
