package gojaverify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/verifyplan"
)

type recordingDriver struct{}

func (recordingDriver) Execute(_ context.Context, step verifyplan.Step) (verifyplan.Observation, error) {
	return verifyplan.Observation{Kind: step.Kind}, nil
}

func TestCompileIsolatedVerificationPlan(t *testing.T) {
	source := `
const V = require("tinyidp/verify").v1;
module.exports = V.plan({
  suites: [{
    name: "authorization interaction",
    scenarios: [{
      name: "forced login",
      steps: [{kind: "authorize.begin", parameters: {prompt: "login"}}],
      assertions: [{id: "freshAuthenticationBeforeIssuance", version: "v1"}]
    }]
  }]
});`
	plan, err := Compile(context.Background(), source, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if plan.SchemaVersion != verifyplan.SchemaVersion || len(plan.Suites) != 1 || plan.SourceHash == "" {
		t.Fatalf("plan=%#v", plan)
	}
}

func TestCompiledPlanRunsWithNativeDriverAndAssertion(t *testing.T) {
	plan, err := Compile(context.Background(), `
const V = require("tinyidp/verify").v1;
module.exports = V.plan({suites: [{name: "native boundary", scenarios: [{
  name: "compiled data is interpreted by Go",
  steps: [{kind: "authorize.begin"}],
  assertions: [{id: "observedKind", version: "v1", config: {kind: "authorize.begin"}}]
}]}]});`, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	runner := verifyplan.Runner{Driver: recordingDriver{}, Steps: verifyplan.StepRegistry{"authorize.begin": verifyplan.ExactObjectValidator}, Assertions: map[string]verifyplan.AssertionFunc{
		"observedKind@v1": func(_ context.Context, config json.RawMessage, observations []verifyplan.Observation) error {
			var expected struct {
				Kind string `json:"kind"`
			}
			if err := json.Unmarshal(config, &expected); err != nil {
				return fmt.Errorf("decode expected observation: %w", err)
			}
			if len(observations) != 1 || observations[0].Kind != expected.Kind {
				return fmt.Errorf("observation does not match native assertion config")
			}
			return nil
		},
	}}
	results, err := runner.Run(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results=%#v", results)
	}
}

func TestCompileRejectsAmbientModules(t *testing.T) {
	_, err := Compile(context.Background(), `require("fs"); module.exports = {};`, DefaultOptions())
	if err == nil || !strings.Contains(err.Error(), "ambient module") {
		t.Fatalf("error=%v", err)
	}
}

func TestCompileInterruptsUnboundedJavaScript(t *testing.T) {
	options := DefaultOptions()
	options.Timeout = 10 * time.Millisecond
	_, err := Compile(context.Background(), `for (;;) {}`, options)
	if err == nil {
		t.Fatal("unbounded JavaScript was not interrupted")
	}
}
