// Package idpuitest provides reusable security and accessibility checks for
// InteractionRenderer implementations.
package idpuitest

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/manuel/tinyidp/pkg/idpui"
	"golang.org/x/net/html"
)

// Violation is one deterministic renderer-conformance finding.
type Violation struct {
	Rule   string
	Detail string
}

func (v Violation) String() string { return v.Rule + ": " + v.Detail }

// RenderAndCheck renders a page and checks the resulting complete HTML
// document against tiny-idp's presentation trust boundary.
func RenderAndCheck(ctx context.Context, renderer idpui.InteractionRenderer, page idpui.InteractionPage) ([]byte, []Violation, error) {
	if renderer == nil {
		return nil, nil, fmt.Errorf("renderer is required")
	}
	if err := page.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validate page: %w", err)
	}
	var output bytes.Buffer
	if err := renderer.RenderInteraction(ctx, &output, page.Clone()); err != nil {
		return nil, nil, fmt.Errorf("render interaction: %w", err)
	}
	violations, err := Check(output.Bytes(), page)
	if err != nil {
		return output.Bytes(), nil, err
	}
	return output.Bytes(), violations, nil
}

// Check parses a rendered document and reports active-content, origin,
// protocol-field, and baseline accessibility violations.
func Check(document []byte, page idpui.InteractionPage) ([]Violation, error) {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, fmt.Errorf("parse rendered HTML: %w", err)
	}
	checker := documentChecker{
		page:       page,
		ids:        map[string]struct{}{},
		labelsFor:  map[string]struct{}{},
		hiddenSeen: map[string][]string{},
		actions:    map[idpui.Action]int{},
	}
	checker.walk(root)
	checker.finish()
	sort.SliceStable(checker.violations, func(i, j int) bool {
		if checker.violations[i].Rule == checker.violations[j].Rule {
			return checker.violations[i].Detail < checker.violations[j].Detail
		}
		return checker.violations[i].Rule < checker.violations[j].Rule
	})
	return checker.violations, nil
}

type documentChecker struct {
	page          idpui.InteractionPage
	violations    []Violation
	ids           map[string]struct{}
	labelsFor     map[string]struct{}
	hiddenSeen    map[string][]string
	actions       map[idpui.Action]int
	credentialIDs []string
	forms         int
	passwordInput int
	loginInput    int
	alerts        int
}

func (c *documentChecker) walk(node *html.Node) {
	if node.Type == html.ElementNode {
		c.checkElement(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		c.walk(child)
	}
}

func (c *documentChecker) checkElement(node *html.Node) {
	tag := strings.ToLower(node.Data)
	attrs := attributeMap(node.Attr)
	if id := attrs["id"]; id != "" {
		c.ids[id] = struct{}{}
	}
	if tag == "label" && attrs["for"] != "" {
		c.labelsFor[attrs["for"]] = struct{}{}
	}
	if attrs["role"] == "alert" {
		c.alerts++
	}
	if forbiddenElements[tag] {
		c.add("active-content", "forbidden <"+tag+"> element")
	}
	if tag == "meta" && strings.EqualFold(attrs["http-equiv"], "refresh") {
		c.add("active-content", "meta refresh is forbidden")
	}
	for name, value := range attrs {
		lowerName := strings.ToLower(name)
		if strings.HasPrefix(lowerName, "on") {
			c.add("event-handler", fmt.Sprintf("<%s> has %s", tag, lowerName))
		}
		if lowerName == "style" {
			c.add("inline-style", fmt.Sprintf("<%s> has a style attribute", tag))
		}
		if urlAttributes[lowerName] {
			c.checkURL(tag, lowerName, value)
		}
	}
	switch tag {
	case "form":
		c.forms++
		if !strings.EqualFold(attrs["method"], "post") {
			c.add("form-contract", "interaction form method must be POST")
		}
		if attrs["action"] != c.page.Form.ActionURL {
			c.add("form-contract", "interaction form action differs from the provider value")
		}
	case "input":
		c.checkInput(attrs)
	case "button":
		if attrs["name"] == c.page.Form.ActionField {
			action := idpui.Action(attrs["value"])
			c.actions[action]++
			if !action.Valid() {
				c.add("action-contract", "button submits an unknown action")
			}
			if action.SkipsConstraintValidation() {
				if _, ok := attrs["formnovalidate"]; !ok {
					c.add("action-contract", "deny action must use formnovalidate")
				}
			}
		}
	}
}

func (c *documentChecker) checkInput(attrs map[string]string) {
	inputType := strings.ToLower(attrs["type"])
	name := attrs["name"]
	if inputType == "hidden" {
		c.hiddenSeen[name] = append(c.hiddenSeen[name], attrs["value"])
		if name != c.page.Form.InteractionField && name != c.page.Form.CSRFField {
			c.add("protocol-field", fmt.Sprintf("unexpected hidden field %q", name))
		}
	}
	if inputType == "password" {
		c.passwordInput++
		if _, ok := attrs["value"]; ok {
			c.add("password-retention", "password input must not have a value attribute")
		}
		if attrs["autocomplete"] != "current-password" {
			c.add("autocomplete", "password input must use current-password")
		}
	}
	if name == idpui.LoginFieldName {
		c.loginInput++
		if attrs["autocomplete"] != "username" {
			c.add("autocomplete", "login input must use username")
		}
	}
	if (inputType == "password" || name == idpui.LoginFieldName) && attrs["id"] == "" {
		c.add("input-label", fmt.Sprintf("input %q must have an id", name))
	} else if inputType == "password" || name == idpui.LoginFieldName {
		c.credentialIDs = append(c.credentialIDs, attrs["id"])
	}
}

func (c *documentChecker) checkURL(tag, name, raw string) {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)
	for _, scheme := range []string{"javascript:", "data:", "vbscript:"} {
		if strings.HasPrefix(lower, scheme) {
			c.add("dangerous-url", fmt.Sprintf("<%s> %s uses %s", tag, name, scheme))
			return
		}
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		c.add("invalid-url", fmt.Sprintf("<%s> %s is invalid", tag, name))
		return
	}
	if parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(trimmed, "//") {
		if tag == "form" && name == "action" && trimmed == c.page.Form.ActionURL {
			return
		}
		c.add("external-origin", fmt.Sprintf("<%s> %s references an external origin", tag, name))
	}
}

func (c *documentChecker) finish() {
	if c.forms != 1 {
		c.add("form-contract", fmt.Sprintf("document contains %d forms; want 1", c.forms))
	}
	c.requireHidden(c.page.Form.InteractionField, c.page.Form.Interaction)
	c.requireHidden(c.page.Form.CSRFField, c.page.Form.CSRFToken)
	for _, expected := range c.page.Form.Actions {
		if c.actions[expected] != 1 {
			c.add("action-contract", fmt.Sprintf("action %q occurs %d times; want 1", expected, c.actions[expected]))
		}
	}
	for action, count := range c.actions {
		if !containsAction(c.page.Form.Actions, action) {
			c.add("action-contract", fmt.Sprintf("unexpected action %q occurs %d times", action, count))
		}
	}
	if c.page.Login != nil {
		if c.loginInput != 1 || c.passwordInput != 1 {
			c.add("credential-contract", fmt.Sprintf("login/password input counts are %d/%d; want 1/1", c.loginInput, c.passwordInput))
		}
		for _, id := range c.credentialIDs {
			if _, labeled := c.labelsFor[id]; !labeled {
				c.add("input-label", fmt.Sprintf("input %q has no explicit label", id))
			}
		}
	} else if c.loginInput != 0 || c.passwordInput != 0 {
		c.add("credential-contract", "credential inputs exist on a page without a login prompt")
	}
	if c.page.Error != nil && c.alerts == 0 {
		c.add("error-identification", "public error is not exposed through an alert")
	}
}

func (c *documentChecker) requireHidden(name, value string) {
	values := c.hiddenSeen[name]
	if len(values) != 1 || values[0] != value {
		c.add("protocol-field", fmt.Sprintf("hidden field %q does not exactly match the provider value", name))
	}
}

func (c *documentChecker) add(rule, detail string) {
	c.violations = append(c.violations, Violation{Rule: rule, Detail: detail})
}

func attributeMap(attrs []html.Attribute) map[string]string {
	result := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		result[strings.ToLower(attr.Key)] = attr.Val
	}
	return result
}

func containsAction(actions []idpui.Action, candidate idpui.Action) bool {
	for _, action := range actions {
		if action == candidate {
			return true
		}
	}
	return false
}

var forbiddenElements = map[string]bool{
	"script": true, "style": true, "iframe": true, "frame": true,
	"object": true, "embed": true, "img": true, "svg": true,
	"math": true, "audio": true, "video": true, "source": true,
}

var urlAttributes = map[string]bool{
	"action": true, "formaction": true, "href": true, "src": true,
	"srcset": true, "poster": true, "data": true, "background": true,
}
