package assurance

import (
	"sort"

	"github.com/pkg/errors"

	"github.com/go-go-golems/tiny-idp/pkg/idpprogram"
)

const LocalWebGraphSchemaVersion = "tinyidp.assurance.local-web/v1"

// NativeBlockDescriptor is a host-owned selection point. It identifies a
// native implementation already represented by the transition catalog; it is
// not a callback or an instruction for JavaScript to execute a transition.
type NativeBlockDescriptor struct {
	ID         string
	WorkflowID string
	Step       StepID
}

// PolicyDescriptor identifies one typed policy family which a compiled Goja
// program may select. The provider contract still controls the callback schema
// and native code still owns where it is invoked.
type PolicyDescriptor struct {
	ID   string
	Kind idpprogram.ProviderKind
}

type LocalWebRegistry struct {
	Blocks   []NativeBlockDescriptor
	Policies []PolicyDescriptor
}

// DefaultLocalWebRegistry is intentionally small: local signup and the three
// post-validation/decoration policy families implemented by the current host.
func DefaultLocalWebRegistry() LocalWebRegistry {
	return LocalWebRegistry{
		Blocks: []NativeBlockDescriptor{{ID: "block.signup.local@v1", WorkflowID: "signup", Step: StepInteractionCreate}},
		Policies: []PolicyDescriptor{
			{ID: "policy.authorization@v1", Kind: idpprogram.ProviderKindAuthorization},
			{ID: "policy.claims@v1", Kind: idpprogram.ProviderKindClaims},
			{ID: "policy.presentation@v1", Kind: idpprogram.ProviderKindPresentation},
		},
	}
}

// LocalWebGraph is the materialized, serializable selection graph for the
// local-web host profile. It contains no Goja functions, HTTP objects, stores,
// keys, credentials, capabilities, or protocol artifacts.
type LocalWebGraph struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Configuration ConfigurationReference `json:"configuration"`
	Blocks        []LocalWebBlock        `json:"blocks"`
	Policies      []LocalWebPolicy       `json:"policies,omitempty"`
}

type LocalWebBlock struct {
	DescriptorID string `json:"descriptorId"`
	WorkflowID   string `json:"workflowId"`
	Step         StepID `json:"step"`
}

type LocalWebPolicy struct {
	DescriptorID string                  `json:"descriptorId"`
	ProviderID   string                  `json:"providerId"`
	Kind         idpprogram.ProviderKind `json:"kind"`
}

// MaterializeLocalWebGraph turns an already compiled Goja program into a
// local-web selection graph. Every selected workflow or provider must resolve
// to a registry descriptor; program contents never create new native blocks,
// policy families, protocol routes, or execution authority.
func MaterializeLocalWebGraph(program idpprogram.Program, fingerprints idpprogram.Fingerprints, registry LocalWebRegistry) (LocalWebGraph, error) {
	if diagnostics := idpprogram.Validate(program); diagnostics.HasErrors() {
		return LocalWebGraph{}, errors.New("compiled program is invalid")
	}
	configuration := ConfigurationReference{SchemaVersion: ConfigurationSchemaVersion, ProgramFingerprint: fingerprints.Program, SourceFingerprint: fingerprints.Source}
	if err := configuration.Validate(); err != nil {
		return LocalWebGraph{}, errors.Wrap(err, "validate compiled program fingerprints")
	}
	blocks := map[string]NativeBlockDescriptor{}
	for _, descriptor := range registry.Blocks {
		if descriptor.ID == "" || descriptor.WorkflowID == "" || !ValidStableID(string(descriptor.Step)) || blocks[descriptor.WorkflowID].ID != "" {
			return LocalWebGraph{}, errors.New("local-web native block registry is invalid")
		}
		blocks[descriptor.WorkflowID] = descriptor
	}
	policies := map[idpprogram.ProviderKind]PolicyDescriptor{}
	for _, descriptor := range registry.Policies {
		if descriptor.ID == "" || !descriptor.Kind.Valid() || policies[descriptor.Kind].ID != "" {
			return LocalWebGraph{}, errors.New("local-web policy registry is invalid")
		}
		policies[descriptor.Kind] = descriptor
	}
	graph := LocalWebGraph{SchemaVersion: LocalWebGraphSchemaVersion, Configuration: configuration}
	for workflowID := range program.Workflows {
		descriptor, ok := blocks[workflowID]
		if !ok {
			return LocalWebGraph{}, errors.Errorf("workflow %q is not a registered local-web block", workflowID)
		}
		graph.Blocks = append(graph.Blocks, LocalWebBlock{DescriptorID: descriptor.ID, WorkflowID: workflowID, Step: descriptor.Step})
	}
	for providerID, provider := range program.Providers {
		descriptor, ok := policies[provider.Kind]
		if !ok {
			return LocalWebGraph{}, errors.Errorf("provider %q kind %q is not a registered local-web policy", providerID, provider.Kind)
		}
		graph.Policies = append(graph.Policies, LocalWebPolicy{DescriptorID: descriptor.ID, ProviderID: providerID, Kind: provider.Kind})
	}
	sort.Slice(graph.Blocks, func(i, j int) bool { return graph.Blocks[i].WorkflowID < graph.Blocks[j].WorkflowID })
	sort.Slice(graph.Policies, func(i, j int) bool { return graph.Policies[i].ProviderID < graph.Policies[j].ProviderID })
	return graph, nil
}
