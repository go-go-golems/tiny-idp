package jitsi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/pluginapi"
	"github.com/go-go-golems/tiny-idp/pkg/idpscript"
)

const policySourcePrefix = `const A = require("tinyidp").v1;
module.exports = A.program("jitsi-policy", p => {
`

const policySourceSuffix = `
  p.provider("authorization", "jitsi", {
    version: 1, state: "virtual", replayProtection: "none", revocation: "none",
    handlers: {decide},
  });
});`

func policySource(run string, capabilities string) string {
	return policySourcePrefix + capabilities + `
  const decide = A.lambda("integration.jitsi.authorize@v1", {
    kind: "provider",
    input: "integration.jitsi.authorize.input.v1",
    output: "integration.jitsi.authorize.output.v1",
    outcomes: ["complete", "deny"],
    effects: [],
    capabilities: ` + capabilityList(capabilities) + `,
    timeoutMs: 50,
    maxCapabilityCalls: 1,
    maxOutputBytes: 4096,
    run: ` + run + `,
  });` + policySourceSuffix
}

func capabilityList(declaration string) string {
	if declaration == "" {
		return "[]"
	}
	return `["meeting.membership.lookup"]`
}

func testInput() PolicyInput {
	return PolicyInput{
		IntegrationID: "jitsi", Room: "engineering", Tenant: "meet.example.test",
		Identity: PolicyIdentity{
			Subject: "user-123", DisplayName: "Test User", PreferredUsername: "test",
			Email: "user@example.test", EmailVerified: true,
			Roles: []string{"meeting-organizer"}, Groups: []string{"staff"}, AuthTime: "2026-07-23T12:00:00Z",
		},
	}
}

func TestPolicyAllowsAndDeniesWithTypedResults(t *testing.T) {
	source := policySource(`ctx => {
    if (!ctx.input.identity.emailVerified) return A.result.deny("verified_email_required");
    return A.result.complete({kind:"complete", claims:{
      displayName:ctx.input.identity.displayName,
      includeEmail:true,
      moderator:ctx.input.identity.roles.includes("meeting-organizer"),
    }});
  }`, "")
	executor, err := NewPolicyExecutor(context.Background(), source, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close(context.Background())
	allowed, err := executor.Authorize(context.Background(), testInput())
	if err != nil || !allowed.Allowed || !allowed.Moderator || allowed.DisplayName != "Test User" {
		t.Fatalf("allowed = %#v, %v", allowed, err)
	}
	deniedInput := testInput()
	deniedInput.Identity.EmailVerified = false
	denied, err := executor.Authorize(context.Background(), deniedInput)
	if err != nil || denied.Allowed || denied.DiagnosticID != "verified_email_required" {
		t.Fatalf("denied = %#v, %v", denied, err)
	}
	stats := executor.Stats()
	if stats.Invocations != 2 || stats.Allowed != 1 || stats.Denied != 1 || stats.Failures != 0 || !executor.Ready() {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestPolicyInputFromIdentityUsesEmptyJSONArrays(t *testing.T) {
	input := PolicyInputFromIdentity(pluginapi.Identity{
		Subject: "user-123",
		Name:    "Test User",
	}, "jitsi", "engineering", "meet.example.test")
	if input.Identity.Roles == nil || input.Identity.Groups == nil {
		t.Fatalf("empty collections must be arrays: roles=%#v groups=%#v", input.Identity.Roles, input.Identity.Groups)
	}

	source := policySource(`ctx => A.result.complete({kind:"complete", claims:{
      displayName:ctx.input.identity.displayName,
      includeEmail:false,
      moderator:ctx.input.identity.roles.includes("meeting-organizer"),
    }})`, "")
	executor, err := NewPolicyExecutor(context.Background(), source, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close(context.Background())
	decision, err := executor.Authorize(context.Background(), input)
	if err != nil || !decision.Allowed || decision.DisplayName != "Test User" {
		t.Fatalf("decision = %#v, %v", decision, err)
	}
}

func TestPolicyRejectsMalformedOutputAndCapabilities(t *testing.T) {
	malformed := policySource(`_ => A.result.complete({kind:"complete", claims:{displayName:"", includeEmail:true, moderator:false}})`, "")
	executor, err := NewPolicyExecutor(context.Background(), malformed, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close(context.Background())
	if _, err := executor.Authorize(context.Background(), testInput()); err == nil {
		t.Fatal("malformed completion accepted")
	}
	capabilityDeclaration := `p.capabilities({"meeting.membership.lookup": {version:1}});`
	withCapability := policySource(`async ctx => {
    await ctx.cap.meeting.membership.lookup({subject:ctx.input.identity.subject});
    return A.result.deny("meeting_access_denied");
  }`, capabilityDeclaration)
	if _, err := NewPolicyExecutor(context.Background(), withCapability, 1); err == nil || !strings.Contains(err.Error(), "does not permit capabilities") {
		t.Fatalf("capability error = %v", err)
	}
}

func TestPolicyTimeoutDiscardsWorkerAndRecovers(t *testing.T) {
	source := policySource(`_ => { while (true) {} }`, "")
	executor, err := NewPolicyExecutor(context.Background(), source, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close(context.Background())
	_, err = executor.Authorize(context.Background(), testInput())
	if !errors.Is(err, idpscript.ErrInvocationTimeout) {
		t.Fatalf("timeout error = %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for executor.Stats().Pool.Discarded == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if executor.Stats().Pool.Discarded == 0 || !executor.Ready() {
		t.Fatalf("pool did not replace interrupted worker: %#v", executor.Stats())
	}
}

func TestPolicyPoolFailsClosedWhenSaturated(t *testing.T) {
	source := policySource(`_ => { while (true) {} }`, "")
	executor, err := NewPolicyExecutor(context.Background(), source, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close(context.Background())
	firstDone := make(chan error, 1)
	go func() {
		_, invokeErr := executor.Authorize(context.Background(), testInput())
		firstDone <- invokeErr
	}()
	deadline := time.Now().Add(time.Second)
	for executor.Stats().Pool.Active != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if executor.Stats().Pool.Active != 1 {
		t.Fatal("first invocation did not acquire the only worker")
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if _, err := executor.Authorize(waitCtx, testInput()); !errors.Is(err, idpscript.ErrRuntimeSaturated) {
		t.Fatalf("saturation error = %v", err)
	}
	if err := <-firstDone; !errors.Is(err, idpscript.ErrInvocationTimeout) {
		t.Fatalf("first invocation error = %v", err)
	}
}

func TestPolicyTypeScriptDeclaresVersionedContract(t *testing.T) {
	for _, fragment := range []string{
		"interface JitsiAuthorizeInput", "roles: string[]", `kind: "complete"`, `kind: "deny"`,
		"verified_email_required", "moderator: boolean",
	} {
		if !strings.Contains(PolicyTypeScript, fragment) {
			t.Fatalf("TypeScript declaration missing %q", fragment)
		}
	}
}
