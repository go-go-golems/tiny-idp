package pluginapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type testDefinition struct {
	descriptor Descriptor
	slug       string
	prefix     string
}

func (d testDefinition) Descriptor() Descriptor { return d.descriptor }
func (d testDefinition) Section() (schema.Section, error) {
	return schema.NewSection(d.slug, d.slug, schema.WithPrefix(d.prefix))
}
func (d testDefinition) Prepare(context.Context, *values.Values) (Prepared, error) {
	return nil, nil
}

var _ Definition = testDefinition{}
var _ http.Handler = http.HandlerFunc(nil)

func TestRegistryIsValidatedSortedAndImmutable(t *testing.T) {
	first := testDefinition{descriptor: Descriptor{ID: "alpha", APIVersion: APIVersion, Summary: "alpha"}, slug: "plugin-alpha", prefix: "alpha-"}
	second := testDefinition{descriptor: Descriptor{ID: "zulu", APIVersion: APIVersion, Summary: "zulu"}, slug: "plugin-zulu", prefix: "zulu-"}
	registry, err := NewRegistry(second, first)
	if err != nil {
		t.Fatal(err)
	}
	definitions := registry.Definitions()
	if len(definitions) != 2 || definitions[0].Descriptor().ID != "alpha" {
		t.Fatalf("definitions = %#v", definitions)
	}
	definitions[0] = second
	again := registry.Definitions()
	if again[0].Descriptor().ID != "alpha" {
		t.Fatal("registry definitions were mutable through returned slice")
	}
	if got, ok := registry.Definition("zulu"); !ok || got.Descriptor().ID != "zulu" {
		t.Fatalf("definition lookup = %#v, %v", got, ok)
	}
}

func TestRegistryRejectsInvalidAndCollidingDefinitions(t *testing.T) {
	valid := testDefinition{descriptor: Descriptor{ID: "alpha", APIVersion: APIVersion, Summary: "alpha"}, slug: "plugin-alpha", prefix: "alpha-"}
	cases := []struct {
		name        string
		definitions []Definition
	}{
		{name: "nil", definitions: []Definition{nil}},
		{name: "invalid id", definitions: []Definition{testDefinition{descriptor: Descriptor{ID: "Alpha", APIVersion: APIVersion, Summary: "alpha"}, slug: "a", prefix: "a-"}}},
		{name: "version", definitions: []Definition{testDefinition{descriptor: Descriptor{ID: "alpha", APIVersion: 99, Summary: "alpha"}, slug: "a", prefix: "a-"}}},
		{name: "duplicate id", definitions: []Definition{valid, valid}},
		{name: "duplicate slug", definitions: []Definition{valid, testDefinition{descriptor: Descriptor{ID: "beta", APIVersion: APIVersion, Summary: "beta"}, slug: valid.slug, prefix: "beta-"}}},
		{name: "duplicate prefix", definitions: []Definition{valid, testDefinition{descriptor: Descriptor{ID: "beta", APIVersion: APIVersion, Summary: "beta"}, slug: "plugin-beta", prefix: valid.prefix}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRegistry(test.definitions...); err == nil {
				t.Fatal("invalid registry accepted")
			}
		})
	}
}
