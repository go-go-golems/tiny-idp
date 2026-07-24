package doc

import (
	"strings"
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
)

func TestPluginTutorialIsEmbeddedAndDiscoverable(t *testing.T) {
	t.Parallel()

	helpSystem := help.NewHelpSystem()
	if err := AddDocToHelpSystem(helpSystem); err != nil {
		t.Fatalf("load embedded TinyIDP help: %v", err)
	}

	section, err := helpSystem.GetSectionWithSlug("writing-and-deploying-plugins")
	if err != nil {
		t.Fatalf("resolve plugin tutorial by slug: %v", err)
	}
	if section.SectionType.String() != "Tutorial" {
		t.Fatalf("plugin help section type = %s, want Tutorial", section.SectionType)
	}
	for _, required := range []string{
		"compiled-in, first-party integrations",
		"Definition.Prepare(ctx, values)",
		"internal/plugins/jitsi",
		"Troubleshooting",
	} {
		if !strings.Contains(section.Content, required) {
			t.Errorf("plugin tutorial does not contain %q", required)
		}
	}
}
