package assurance

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
	"github.com/go-go-golems/tiny-idp/pkg/idpsignup"
)

func TestMaterializeLocalWebGraphUsesOnlyRegisteredDescriptors(t *testing.T) {
	artifact, err := idpsignup.Compile(context.Background(), idpsignup.DefaultSource)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := MaterializeLocalWebGraph(artifact.Program(), artifact.Fingerprints(), DefaultLocalWebRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Blocks) != 1 || graph.Blocks[0].DescriptorID != "block.signup.local@v1" || graph.Blocks[0].Step != StepInteractionCreate {
		t.Fatalf("blocks=%#v", graph.Blocks)
	}
	if graph.Configuration.ProgramFingerprint == "" || graph.Configuration.SourceFingerprint == "" {
		t.Fatalf("configuration=%#v", graph.Configuration)
	}
}

func TestMaterializeLocalWebGraphRejectsUnregisteredSelections(t *testing.T) {
	artifact, err := idpsignup.Compile(context.Background(), idpsignup.DefaultSource)
	if err != nil {
		t.Fatal(err)
	}
	program := artifact.Program()
	unregistered := program.Workflows["signup"]
	unregistered.ID = "unregistered"
	program.Workflows["unregistered"] = unregistered
	if _, err := MaterializeLocalWebGraph(program, artifact.Fingerprints(), DefaultLocalWebRegistry()); err == nil || !strings.Contains(err.Error(), "not a registered local-web block") {
		t.Fatalf("unregistered workflow error=%v", err)
	}
	program = artifact.Program()
	program.Schemas["policyInput"] = idpprogram.Schema{ID: "policyInput", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024}
	program.Schemas["policyOutput"] = idpprogram.Schema{ID: "policyOutput", Kind: idpprogram.SchemaKindObject, MaxBytes: 1024}
	program.Lambdas["identity.establish"] = idpprogram.LambdaSpec{ID: "identity.establish", Kind: idpprogram.LambdaKindProvider, InputSchema: "policyInput", OutputSchema: "policyOutput", AllowedOutcomes: []idpprogram.OutcomeKind{idpprogram.OutcomeComplete}, Budget: idpprogram.InvocationBudget{Timeout: time.Second, MaxOutputBytes: 1024}}
	program.Providers = map[string]idpprogram.Provider{"identity.virtual": {ID: "identity.virtual", Kind: idpprogram.ProviderKindIdentity, Version: 1, State: idpprogram.ProviderStateVirtual, ReplayProtection: idpprogram.ReplayProtectionNone, Revocation: idpprogram.RevocationNone, Handlers: map[string]idpprogram.ProviderHandler{idpprogram.IdentityEstablishHandler: {ID: idpprogram.IdentityEstablishHandler, LambdaID: "identity.establish", InputSchema: "policyInput", OutputSchema: "policyOutput"}}}}
	if _, err := MaterializeLocalWebGraph(program, artifact.Fingerprints(), DefaultLocalWebRegistry()); err == nil || !strings.Contains(err.Error(), "not a registered local-web policy") {
		t.Fatalf("unregistered policy error=%v", err)
	}
}
