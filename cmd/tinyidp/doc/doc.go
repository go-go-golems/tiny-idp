// Package doc embeds tinyidp's help documentation (Markdown pages with Glazed
// frontmatter) and registers it with the Glazed help system at root-command
// initialization. Pages live in the pages/ subdirectory so the .go source
// file is not itself treated as a help page.
package doc

import (
	"embed"

	"github.com/go-go-golems/glazed/pkg/help"
)

//go:embed all:pages
var docFS embed.FS

// AddDocToHelpSystem loads every embedded help page into the given help
// system. Called once from main() after the help system is created and
// before help_cmd.SetupCobraRootCommand wires `tinyidp help`.
func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
	return helpSystem.LoadSectionsFromFS(docFS, "pages")
}
