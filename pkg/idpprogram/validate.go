package idpprogram

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func containsCapability(requirements []CapabilityRequirement, candidate string) bool {
	for _, requirement := range requirements {
		if requirement.ID == candidate {
			return true
		}
	}
	return false
}

const (
	APIVersionV1        = "tinyidp/v1"
	maxIdentifierBytes  = 128
	maxCarrySchemaBytes = 64 << 10
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`)

// Validate deterministically checks all runtime-independent program invariants.
func Validate(program Program) Diagnostics {
	var diagnostics Diagnostics
	add := func(id, path, message string) {
		diagnostics = append(diagnostics, Diagnostic{ID: id, Severity: SeverityError, Path: path, Message: message})
	}

	if program.APIVersion != APIVersionV1 {
		add("program.api_version", "apiVersion", fmt.Sprintf("must equal %q", APIVersionV1))
	}
	validateIdentifier(&diagnostics, "program.name", "name", program.Name)

	for key, schema := range program.Schemas {
		path := "schemas." + key
		validateMapIdentity(&diagnostics, "schema.id_mismatch", path, key, schema.ID)
		if !schema.Kind.Valid() {
			add("schema.kind", path+".kind", fmt.Sprintf("unsupported schema kind %q", schema.Kind))
		}
		if schema.MaxBytes <= 0 || schema.MaxBytes > maxCarrySchemaBytes {
			add("schema.max_bytes", path+".maxBytes", fmt.Sprintf("must be between 1 and %d", maxCarrySchemaBytes))
		}
		if schema.Kind == SchemaKindObject {
			for fieldName, field := range schema.Fields {
				fieldPath := path + ".fields." + fieldName
				validateIdentifier(&diagnostics, "schema.field_id", fieldPath, fieldName)
				if field.Ref == "" || program.Schemas[field.Ref].ID == "" {
					add("schema.field_ref", fieldPath+".ref", fmt.Sprintf("unknown schema %q", field.Ref))
				}
			}
		} else if len(schema.Fields) != 0 {
			add("schema.scalar_fields", path+".fields", "scalar schema must not define fields")
		}
	}
	validateSchemaCycles(&diagnostics, program.Schemas)

	for key, capability := range program.Capabilities {
		path := "capabilities." + key
		validateMapIdentity(&diagnostics, "capability.id_mismatch", path, key, capability.ID)
		if capability.Version == 0 {
			add("capability.version", path+".version", "must be greater than zero")
		}
	}

	for key, lambda := range program.Lambdas {
		path := "lambdas." + key
		validateMapIdentity(&diagnostics, "lambda.id_mismatch", path, key, lambda.ID)
		if !lambda.Kind.Valid() {
			add("lambda.kind", path+".kind", fmt.Sprintf("unsupported lambda kind %q", lambda.Kind))
		}
		if _, ok := program.Schemas[lambda.InputSchema]; !ok {
			add("lambda.input_schema", path+".inputSchema", fmt.Sprintf("unknown schema %q", lambda.InputSchema))
		}
		if _, ok := program.Schemas[lambda.OutputSchema]; !ok {
			add("lambda.output_schema", path+".outputSchema", fmt.Sprintf("unknown schema %q", lambda.OutputSchema))
		}
		if len(lambda.AllowedOutcomes) == 0 {
			add("lambda.outcomes_empty", path+".allowedOutcomes", "must declare at least one outcome")
		}
		seenOutcomes := map[OutcomeKind]bool{}
		for i, outcome := range lambda.AllowedOutcomes {
			outcomePath := fmt.Sprintf("%s.allowedOutcomes.%d", path, i)
			if !outcome.Valid() {
				add("lambda.outcome", outcomePath, fmt.Sprintf("unsupported outcome %q", outcome))
			}
			if seenOutcomes[outcome] {
				add("lambda.outcome_duplicate", outcomePath, fmt.Sprintf("duplicate outcome %q", outcome))
			}
			seenOutcomes[outcome] = true
		}
		seenCapabilities := map[string]bool{}
		for i, requirement := range lambda.RequiredCapabilities {
			requirementPath := fmt.Sprintf("%s.requiredCapabilities.%d", path, i)
			declared, ok := program.Capabilities[requirement.ID]
			if !ok {
				add("lambda.capability", requirementPath, fmt.Sprintf("undeclared capability %q", requirement.ID))
			} else if declared.Version != requirement.Version {
				add("lambda.capability_version", requirementPath, fmt.Sprintf("capability %q requires version %d, program declares %d", requirement.ID, requirement.Version, declared.Version))
			}
			if seenCapabilities[requirement.ID] {
				add("lambda.capability_duplicate", requirementPath, fmt.Sprintf("duplicate capability %q", requirement.ID))
			}
			seenCapabilities[requirement.ID] = true
		}
		seenEffects := map[EffectKind]bool{}
		for i, effect := range lambda.AllowedEffects {
			effectPath := fmt.Sprintf("%s.allowedEffects.%d", path, i)
			if !effect.Valid() {
				add("lambda.effect", effectPath, fmt.Sprintf("unsupported effect %q", effect))
			}
			if seenEffects[effect] {
				add("lambda.effect_duplicate", effectPath, fmt.Sprintf("duplicate effect %q", effect))
			}
			seenEffects[effect] = true
		}
		if lambda.Budget.Timeout <= 0 {
			add("lambda.timeout", path+".budget.timeoutNanos", "must be greater than zero")
		}
		if lambda.Budget.MaxCapabilityCalls < 0 {
			add("lambda.capability_budget", path+".budget.maxCapabilityCalls", "must not be negative")
		}
		if lambda.Budget.MaxOutputBytes <= 0 {
			add("lambda.output_budget", path+".budget.maxOutputBytes", "must be greater than zero")
		}
	}

	seenTests := map[string]bool{}
	for index, test := range program.Tests {
		path := fmt.Sprintf("tests.%d", index)
		if test.ID == "" || seenTests[test.ID] {
			add("test.id", path+".id", "must be nonempty and unique")
		}
		seenTests[test.ID] = true
		lambda, ok := program.Lambdas[test.LambdaID]
		if !ok {
			add("test.lambda", path+".lambdaId", fmt.Sprintf("unknown lambda %q", test.LambdaID))
			continue
		}
		if !test.ExpectedKind.Valid() || !containsOutcome(lambda.AllowedOutcomes, test.ExpectedKind) {
			add("test.expected_outcome", path+".expectedKind", "must be a declared lambda outcome")
		}
		if err := ValidateJSON(program.Schemas, lambda.InputSchema, test.Input); err != nil {
			add("test.input", path+".input", err.Error())
		}
		for capabilityID, output := range test.Fakes {
			if !containsCapability(lambda.RequiredCapabilities, capabilityID) {
				add("test.fake_capability", path+".fakes."+capabilityID, "must name a capability required by the test lambda")
			}
			if len(output) == 0 || !json.Valid(output) || len(output) > maxCarrySchemaBytes {
				add("test.fake_output", path+".fakes."+capabilityID, "must be valid bounded JSON")
			}
		}
	}

	for key, workflow := range program.Workflows {
		path := "workflows." + key
		validateMapIdentity(&diagnostics, "workflow.id_mismatch", path, key, workflow.ID)
		if workflow.Version == 0 {
			add("workflow.version", path+".version", "must be greater than zero")
		}
		if _, ok := workflow.Handlers[workflow.EntryHandler]; !ok {
			add("workflow.entry", path+".entryHandler", fmt.Sprintf("unknown handler %q", workflow.EntryHandler))
		}
		for handlerKey, handler := range workflow.Handlers {
			handlerPath := path + ".handlers." + handlerKey
			validateMapIdentity(&diagnostics, "handler.id_mismatch", handlerPath, handlerKey, handler.ID)
			lambda, ok := program.Lambdas[handler.LambdaID]
			if !ok {
				add("handler.lambda", handlerPath+".lambdaId", fmt.Sprintf("unknown lambda %q", handler.LambdaID))
			} else if lambda.Kind != LambdaKindWorkflow {
				add("handler.lambda_kind", handlerPath+".lambdaId", fmt.Sprintf("lambda %q is not a workflow lambda", handler.LambdaID))
			}
			seenEdges := map[string]bool{}
			for i, edge := range handler.ContinuationEdges {
				edgePath := fmt.Sprintf("%s.continuationEdges.%d", handlerPath, i)
				if edge.OutcomeKind != OutcomeContinue && edge.OutcomeKind != OutcomePresent && edge.OutcomeKind != OutcomeChallenge {
					add("edge.outcome", edgePath+".outcomeKind", "edge must use continue, present, or challenge")
				}
				if ok && !containsOutcome(lambda.AllowedOutcomes, edge.OutcomeKind) {
					add("edge.undeclared_outcome", edgePath+".outcomeKind", fmt.Sprintf("lambda %q does not allow %q", lambda.ID, edge.OutcomeKind))
				}
				target, targetOK := workflow.Handlers[edge.HandlerID]
				if !targetOK {
					add("edge.handler", edgePath+".handlerId", fmt.Sprintf("unknown handler %q", edge.HandlerID))
				} else if targetLambda, targetLambdaOK := program.Lambdas[target.LambdaID]; targetLambdaOK && targetLambda.InputSchema != edge.InputSchema {
					add("edge.schema", edgePath+".inputSchema", fmt.Sprintf("target handler requires %q, edge declares %q", targetLambda.InputSchema, edge.InputSchema))
				}
				edgeKey := string(edge.OutcomeKind) + "\x00" + edge.HandlerID
				if seenEdges[edgeKey] {
					add("edge.duplicate", edgePath, fmt.Sprintf("duplicate %s edge to %q", edge.OutcomeKind, edge.HandlerID))
				}
				seenEdges[edgeKey] = true
			}
		}
		validateReachability(&diagnostics, path, workflow)
	}

	for key, provider := range program.Providers {
		path := "providers." + key
		validateMapIdentity(&diagnostics, "provider.id_mismatch", path, key, provider.ID)
		if !provider.Kind.Valid() {
			add("provider.kind", path+".kind", fmt.Sprintf("unsupported provider kind %q", provider.Kind))
		}
		if provider.Version == 0 {
			add("provider.version", path+".version", "must be greater than zero")
		}
		if !provider.State.Valid() {
			add("provider.state", path+".state", fmt.Sprintf("unsupported provider state %q", provider.State))
		}
		if !provider.ReplayProtection.Valid() {
			add("provider.replay", path+".replayProtection", fmt.Sprintf("unsupported replay protection %q", provider.ReplayProtection))
		}
		if !provider.Revocation.Valid() {
			add("provider.revocation", path+".revocation", fmt.Sprintf("unsupported revocation mode %q", provider.Revocation))
		}
		if provider.State == ProviderStateVirtual && provider.ReplayProtection == ReplayProtectionOneTime {
			add("provider.one_time_state", path+".replayProtection", "one-time replay protection requires durable provider state")
		}
		if provider.Revocation == RevocationDurable && provider.State != ProviderStateDurable {
			add("provider.durable_revocation_state", path+".revocation", "durable revocation requires durable provider state")
		}
		requiredHandler := requiredProviderHandler(provider.Kind)
		if _, ok := provider.Handlers[requiredHandler]; !ok {
			add("provider.required_handler", path+".handlers", fmt.Sprintf("%s provider requires handler %q", provider.Kind, requiredHandler))
		}
		for handlerID, handler := range provider.Handlers {
			handlerPath := path + ".handlers." + handlerID
			validateMapIdentity(&diagnostics, "provider.handler_id_mismatch", handlerPath, handlerID, handler.ID)
			lambda, ok := program.Lambdas[handler.LambdaID]
			if !ok {
				add("provider.handler_lambda", handlerPath+".lambdaId", fmt.Sprintf("unknown lambda %q", handler.LambdaID))
				continue
			}
			if lambda.Kind != LambdaKindProvider {
				add("provider.handler_lambda_kind", handlerPath+".lambdaId", fmt.Sprintf("lambda %q is not a provider lambda", lambda.ID))
			}
			if lambda.InputSchema != handler.InputSchema {
				add("provider.handler_input_schema", handlerPath+".inputSchema", fmt.Sprintf("lambda %q requires %q, provider declares %q", lambda.ID, lambda.InputSchema, handler.InputSchema))
			}
			if lambda.OutputSchema != handler.OutputSchema {
				add("provider.handler_output_schema", handlerPath+".outputSchema", fmt.Sprintf("lambda %q returns %q, provider declares %q", lambda.ID, lambda.OutputSchema, handler.OutputSchema))
			}
			if _, ok := program.Schemas[handler.InputSchema]; !ok {
				add("provider.handler_input_schema_unknown", handlerPath+".inputSchema", fmt.Sprintf("unknown schema %q", handler.InputSchema))
			}
			if _, ok := program.Schemas[handler.OutputSchema]; !ok {
				add("provider.handler_output_schema_unknown", handlerPath+".outputSchema", fmt.Sprintf("unknown schema %q", handler.OutputSchema))
			}
		}
	}

	return diagnostics.sorted()
}

func requiredProviderHandler(kind ProviderKind) string {
	switch kind {
	case ProviderKindIdentity:
		return IdentityEstablishHandler
	case ProviderKindAuthorization:
		return AuthorizationDecideHandler
	case ProviderKindClaims:
		return ClaimsAdditionalHandler
	case ProviderKindInvitation:
		return InvitationValidateHandler
	default:
		return ""
	}
}

func validateSchemaCycles(diagnostics *Diagnostics, schemas map[string]Schema) {
	state := map[string]uint8{}
	var visit func(string, []string)
	visit = func(id string, stack []string) {
		switch state[id] {
		case 1:
			*diagnostics = append(*diagnostics, Diagnostic{
				ID:       "schema.reference_cycle",
				Severity: SeverityError,
				Path:     "schemas." + id,
				Message:  "schema reference cycle: " + strings.Join(append(stack, id), " -> "),
			})
			return
		case 2:
			return
		}
		state[id] = 1
		schema, ok := schemas[id]
		if ok {
			fieldNames := make([]string, 0, len(schema.Fields))
			for fieldName := range schema.Fields {
				fieldNames = append(fieldNames, fieldName)
			}
			sort.Strings(fieldNames)
			for _, fieldName := range fieldNames {
				ref := schema.Fields[fieldName].Ref
				if _, exists := schemas[ref]; exists {
					visit(ref, append(stack, id))
				}
			}
		}
		state[id] = 2
	}

	ids := make([]string, 0, len(schemas))
	for id := range schemas {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		visit(id, nil)
	}
}

func validateReachability(diagnostics *Diagnostics, path string, workflow Workflow) {
	if _, ok := workflow.Handlers[workflow.EntryHandler]; !ok {
		return
	}
	reachable := map[string]bool{}
	queue := []string{workflow.EntryHandler}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if reachable[current] {
			continue
		}
		reachable[current] = true
		for _, edge := range workflow.Handlers[current].ContinuationEdges {
			if _, ok := workflow.Handlers[edge.HandlerID]; ok && !reachable[edge.HandlerID] {
				queue = append(queue, edge.HandlerID)
			}
		}
	}
	for handlerID := range workflow.Handlers {
		if !reachable[handlerID] {
			*diagnostics = append(*diagnostics, Diagnostic{
				ID:       "handler.unreachable",
				Severity: SeverityError,
				Path:     path + ".handlers." + handlerID,
				Message:  fmt.Sprintf("handler %q is unreachable from entry %q", handlerID, workflow.EntryHandler),
			})
		}
	}
}

func validateMapIdentity(diagnostics *Diagnostics, id, path, key, value string) {
	validateIdentifier(diagnostics, id, path, key)
	if value != key {
		*diagnostics = append(*diagnostics, Diagnostic{ID: id, Severity: SeverityError, Path: path + ".id", Message: fmt.Sprintf("map key %q does not match id %q", key, value)})
	}
}

func validateIdentifier(diagnostics *Diagnostics, id, path, value string) {
	if len(value) == 0 || len(value) > maxIdentifierBytes || !identifierPattern.MatchString(value) || strings.Contains(value, "..") {
		*diagnostics = append(*diagnostics, Diagnostic{ID: id, Severity: SeverityError, Path: path, Message: fmt.Sprintf("invalid identifier %q", value)})
	}
}
