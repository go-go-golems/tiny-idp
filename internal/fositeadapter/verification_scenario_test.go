package fositeadapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/internal/assurance"
	"github.com/go-go-golems/tiny-idp/internal/gojaverify"
	"github.com/go-go-golems/tiny-idp/pkg/verifyplan"
)

type securityClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *securityClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *securityClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

type strictScenarioDriver struct {
	fixture *interactionFixture
	clock   *securityClock
	form    url.Values
}

var _ verifyplan.Driver = (*strictScenarioDriver)(nil)

type beginAuthorizationParameters struct {
	Prompt string `json:"prompt,omitempty"`
	MaxAge string `json:"maxAge,omitempty"`
	State  string `json:"state,omitempty"`
}

type submitInteractionParameters struct {
	Login     string              `json:"login,omitempty"`
	Action    string              `json:"action,omitempty"`
	Mutations map[string][]string `json:"mutations,omitempty"`
}

type advanceClockParameters struct {
	Duration string `json:"duration"`
}

func (d *strictScenarioDriver) Execute(_ context.Context, step verifyplan.Step) (verifyplan.Observation, error) {
	switch step.Kind {
	case string(assurance.StepSessionLogin):
		d.fixture.login()
		return verifyplan.Observation{Kind: "session.established"}, nil
	case string(assurance.StepInteractionCreate):
		var parameters beginAuthorizationParameters
		if err := decodeStepParameters(step.Parameters, &parameters); err != nil {
			return verifyplan.Observation{}, fmt.Errorf("decode interaction.create: %w", err)
		}
		request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		request.Del("login")
		if parameters.Prompt != "" {
			request.Set("prompt", parameters.Prompt)
		}
		if parameters.MaxAge != "" {
			request.Set("max_age", parameters.MaxAge)
		}
		if parameters.State != "" {
			request.Set("state", parameters.State)
		}
		form, body, status := d.fixture.begin(request)
		d.form = form
		return verifyplan.Observation{Kind: "authorize.response", Data: map[string]any{
			"status":            status,
			"credentialForm":    strings.Contains(body, `name="password"`),
			"opaqueInteraction": form.Get("interaction") != "",
		}}, nil
	case string(assurance.StepInteractionApprove):
		var parameters submitInteractionParameters
		if err := decodeStepParameters(step.Parameters, &parameters); err != nil {
			return verifyplan.Observation{}, fmt.Errorf("decode interaction.approve: %w", err)
		}
		if d.form == nil {
			return verifyplan.Observation{}, fmt.Errorf("interaction.approve requires interaction.create")
		}
		form := cloneScenarioValues(d.form)
		if parameters.Login != "" {
			form.Set("login", parameters.Login)
		}
		action := parameters.Action
		if action == "" {
			action = "approve"
		}
		form.Set("action", action)
		for key, values := range parameters.Mutations {
			form.Del(key)
			for _, value := range values {
				form.Add(key, value)
			}
		}
		response := d.fixture.submit(form)
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return verifyplan.Observation{}, fmt.Errorf("read interaction response: %w", err)
		}
		location, _ := url.Parse(response.Header.Get("Location"))
		return verifyplan.Observation{Kind: "interaction.response", Data: map[string]any{
			"status": response.StatusCode,
			"code":   location.Query().Get("code") != "",
			"error":  location.Query().Get("error"),
			"state":  location.Query().Get("state"),
			"body":   string(body),
		}}, nil
	case string(assurance.StepClockAdvance):
		var parameters advanceClockParameters
		if err := decodeStepParameters(step.Parameters, &parameters); err != nil {
			return verifyplan.Observation{}, fmt.Errorf("decode clock.advance: %w", err)
		}
		duration, err := time.ParseDuration(parameters.Duration)
		if err != nil || duration < 0 {
			return verifyplan.Observation{}, fmt.Errorf("invalid non-negative clock duration %q", parameters.Duration)
		}
		d.clock.Advance(duration)
		return verifyplan.Observation{Kind: "clock.advanced", Data: map[string]any{"duration": duration.String()}}, nil
	default:
		return verifyplan.Observation{}, fmt.Errorf("unknown strict-provider action %q", step.Kind)
	}
}

func decodeStepParameters(raw json.RawMessage, destination any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("parameters contain trailing JSON")
	}
	return nil
}

func cloneScenarioValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, entries := range values {
		cloned[key] = append([]string(nil), entries...)
	}
	return cloned
}

func TestVerificationPlanRunsAgainstStrictProvider(t *testing.T) {
	clock := &securityClock{now: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)}
	fixture := newInteractionFixtureWithClock(t, nil, clock.Now)
	driver := &strictScenarioDriver{fixture: fixture, clock: clock}
	plan, err := gojaverify.Compile(context.Background(), `
const V = require("tinyidp/verify").v1;
module.exports = V.plan({suites: [{name: "fresh authentication", scenarios: [{
  name: "blank forced-login submit cannot reuse an old session",
  steps: [
    {kind: "session.login@v1"},
    {kind: "interaction.create@v1", parameters: {prompt: "login", state: "forced-state"}},
    {kind: "interaction.approve@v1"}
  ],
  assertions: [
    {id: "credentialFormShown", version: "v1"},
    {id: "noAuthorizationCode", version: "v1"}
  ]
}]}]});`, gojaverify.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	runner := verifyplan.Runner{Driver: driver, Steps: strictScenarioSteps(), Assertions: strictScenarioAssertions()}
	results, err := runner.Run(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results=%#v", results)
	}
}

func TestNormalizedModelCounterexampleReplaysWithStableRegisteredSteps(t *testing.T) {
	clock := &securityClock{now: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)}
	driver := &strictScenarioDriver{fixture: newInteractionFixtureWithClock(t, nil, clock.Now), clock: clock}
	counterexample := assurance.NormalizedCounterexample{
		SchemaVersion: assurance.NormalizedCounterexampleSchemaVersion,
		Name:          "duplicate approved terminal is rejected",
		Steps: []assurance.ScenarioStep{
			{Step: assurance.StepInteractionCreate, Parameters: json.RawMessage(`{}`)},
			{Step: assurance.StepInteractionApprove, Parameters: json.RawMessage(`{"login":"alice"}`)},
			{Step: assurance.StepInteractionApprove, Parameters: json.RawMessage(`{}`)},
		},
	}
	plan, err := counterexample.VerificationPlan(assurance.CurrentAuthorizationCatalog())
	if err != nil {
		t.Fatal(err)
	}
	results, err := (verifyplan.Runner{Driver: driver, Steps: strictScenarioSteps(), Assertions: strictScenarioAssertions()}).Run(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Passed || len(results[0].Observations) != 3 {
		t.Fatalf("results=%#v", results)
	}
	first, second := results[0].Observations[1].Data, results[0].Observations[2].Data
	if first["code"] != true || second["code"] != false || second["status"] != http.StatusBadRequest {
		t.Fatalf("counterexample replay observations=%#v", results[0].Observations)
	}
}

func strictScenarioAssertions() map[string]verifyplan.AssertionFunc {
	return map[string]verifyplan.AssertionFunc{
		"credentialFormShown@v1": func(_ context.Context, _ json.RawMessage, observations []verifyplan.Observation) error {
			for _, observation := range observations {
				if observation.Kind == "authorize.response" && observation.Data["credentialForm"] == true {
					return nil
				}
			}
			return fmt.Errorf("authorization did not require a credential form")
		},
		"noAuthorizationCode@v1": func(_ context.Context, _ json.RawMessage, observations []verifyplan.Observation) error {
			for index := len(observations) - 1; index >= 0; index-- {
				if observations[index].Kind == "interaction.response" {
					if observations[index].Data["code"] == true {
						return fmt.Errorf("interaction issued an authorization code")
					}
					return nil
				}
			}
			return fmt.Errorf("scenario has no interaction response")
		},
	}
}

func strictScenarioSteps() verifyplan.StepRegistry {
	return verifyplan.StepRegistry{
		string(assurance.StepSessionLogin): verifyplan.ExactObjectValidator,
		string(assurance.StepInteractionCreate): func(raw json.RawMessage) error {
			var parameters beginAuthorizationParameters
			return decodeStepParameters(raw, &parameters)
		},
		string(assurance.StepInteractionApprove): func(raw json.RawMessage) error {
			var parameters submitInteractionParameters
			return decodeStepParameters(raw, &parameters)
		},
		string(assurance.StepClockAdvance): func(raw json.RawMessage) error {
			var parameters advanceClockParameters
			if err := decodeStepParameters(raw, &parameters); err != nil {
				return err
			}
			duration, err := time.ParseDuration(parameters.Duration)
			if err != nil || duration < 0 {
				return fmt.Errorf("invalid non-negative clock duration %q", parameters.Duration)
			}
			return nil
		},
	}
}

func TestStrictScenarioDriverRejectsUnknownFieldsAndActions(t *testing.T) {
	clock := &securityClock{now: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)}
	driver := &strictScenarioDriver{fixture: newInteractionFixtureWithClock(t, nil, clock.Now), clock: clock}
	if _, err := driver.Execute(context.Background(), verifyplan.Step{Kind: string(assurance.StepInteractionCreate), Parameters: json.RawMessage(`{"unknown":true}`)}); err == nil {
		t.Fatal("unknown security action parameter was accepted")
	}
	if _, err := driver.Execute(context.Background(), verifyplan.Step{Kind: "provider.raw"}); err == nil {
		t.Fatal("unknown action was accepted")
	}
}

func TestAuthorizationMetamorphicIrrelevantUILocalesPreservesOutcome(t *testing.T) {
	for _, uiLocales := range []string{"", "en", "en fr"} {
		t.Run(strings.ReplaceAll(uiLocales, " ", "_"), func(t *testing.T) {
			fixture := newInteractionFixture(t, nil)
			request := authorizeForm("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			request.Del("login")
			request.Set("state", "metamorphic-state")
			if uiLocales != "" {
				request.Set("ui_locales", uiLocales)
			}
			form, _, status := fixture.begin(request)
			if status != http.StatusOK {
				t.Fatalf("begin status=%d", status)
			}
			form.Set("login", "alice")
			response := fixture.submit(form)
			defer response.Body.Close()
			location, _ := url.Parse(response.Header.Get("Location"))
			if location.Query().Get("code") == "" || location.Query().Get("state") != "metamorphic-state" {
				t.Fatalf("ui_locales=%q changed authorization relation: %s", uiLocales, location.String())
			}
		})
	}
}
