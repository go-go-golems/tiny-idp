package server

import (
	_ "embed"
	"html/template"
)

// loginHTML is the synthetic-user login form. It is embedded into the binary
// at build time so the server stays a single artifact with no external file
// dependencies. Editing static/login.html and rebuilding is all that's
// needed to change the page.
//
//go:embed static/login.html
var loginHTML string

// loginPage is the parsed login template. Parsed once at package init; the
// template is safe for concurrent execution.
var loginPage = template.Must(template.New("login").Parse(loginHTML))

// hiddenField is a hidden form input echoed back from the authorize request
// so the POST /authorize (login submit) reconstructs the original OAuth/OIDC
// request verbatim.
type hiddenField struct {
	Name  string
	Value string
}

// scenarioGroup is a labeled collection of selectable scenarios on the login
// page. It mirrors scenario.CategoryGroup but uses the template-facing
// scenarioItem shape.
type scenarioGroup struct {
	Label string
	Items []scenarioItem
}

type scenarioItem struct {
	Name        string
	Description string
}

// loginPageData is the template model.
type loginPageData struct {
	Hidden    []hiddenField
	Scenarios []scenarioGroup
}

// scenarioGroups converts the scenario registry's category groups into the
// template-facing shape. It is the bridge between internal/scenario and the
// embedded login template.
func (s *Server) scenarioGroups() []scenarioGroup {
	in := s.registry.Grouped()
	out := make([]scenarioGroup, 0, len(in))
	for _, g := range in {
		items := make([]scenarioItem, 0, len(g.Items))
		for _, sc := range g.Items {
			items = append(items, scenarioItem{Name: sc.Name, Description: sc.Description})
		}
		out = append(out, scenarioGroup{Label: g.Label, Items: items})
	}
	return out
}
