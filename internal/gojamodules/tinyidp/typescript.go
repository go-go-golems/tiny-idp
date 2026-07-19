package tinyidp

import (
	"github.com/go-go-golems/go-go-goja/modules"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
)

var _ modules.TypeScriptDeclarer = (*module)(nil)

// TypeScriptModule describes the deliberately small Phase 0 authoring API.
// Runtime invocation capabilities live on ctx.cap and are declared as unknown
// because their precise shape is derived from the program's host catalog.
func (*module) TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name:        Name,
		Description: "Compile bounded Tiny-IDP workflow lambdas without ambient host authority.",
		RawDTS: []string{
			`export type OutcomeKind = "continue" | "present" | "challenge" | "commit" | "complete" | "deny" | "skip" | "error";`,
			`export type EffectKind = "read" | "createLocalIdentity" | "attachPasswordCredential" | "consumeInvitation" | "establishBrowserSession" | "establishVirtualIdentity" | "sendEmailChallenge";`,
			`export interface PresentationSpec { title: string; resume: string; fields: FieldHandle[]; actions: ActionHandle[]; carry: unknown; expiresInSeconds: number; values?: Record<string, string>; errors?: Array<{ field: FieldHandle; code: "required" | "invalid" | "mismatch" | "rejected" }>; }`,
			`export interface PresentationBuilders { form(spec: PresentationSpec): Outcome; }`,
			`export interface SecretHandle { readonly __tinyIdpSecret?: never; }`,
			`export interface SecretContext { readonly [name: string]: SecretHandle | undefined; }`,
			`export interface SignupCommitSpec { login: string; displayName: string; password: SecretHandle; passwordConfirmation: SecretHandle; }`,
			`export interface CommitBuilders { signup(spec: SignupCommitSpec): Outcome; }`,
			`export interface EmailChallengeSpec { email: string; template: string; resume: string; expiresInSeconds: number; maximumAttempts: number; maximumResends: number; carry?: unknown; } export interface ChallengeBuilders { emailCode(spec: EmailChallengeSpec): Outcome; }`,
			`export interface InvocationContext<I = unknown, C = Record<string, unknown>> { readonly input: Readonly<I>; readonly cap: C; readonly present: PresentationBuilders; readonly challenge: ChallengeBuilders; readonly secret: SecretContext; readonly evidence: Readonly<Record<string, unknown>>; readonly commit: CommitBuilders; }`,
			`export interface Outcome { readonly kind: OutcomeKind; readonly code?: string; readonly nextHandler?: string; readonly value?: unknown; }`,
			`export interface LambdaSpec<I = unknown, C = Record<string, unknown>> { kind?: "workflow" | "provider"; input: string; output: string; outcomes: OutcomeKind[]; effects?: EffectKind[]; capabilities?: string[]; timeoutMs: number; maxCapabilityCalls: number; maxOutputBytes: number; run(ctx: InvocationContext<I, C>): Outcome | Promise<Outcome>; }`,
			`export interface LambdaHandle { readonly __tinyIdpLambda?: never; }`,
			`export interface FieldHandle { readonly __tinyIdpField?: never; }`,
			`export interface ActionHandle { readonly __tinyIdpAction?: never; }`,
			`export interface FieldBuilders { displayName(): FieldHandle; email(): FieldHandle; password(): FieldHandle; passwordConfirmation(): FieldHandle; inviteCode(): FieldHandle; emailCode(): FieldHandle; }`,
			`export interface ActionBuilders { submit(): ActionHandle; deny(): ActionHandle; }`,
			`export interface CapabilityRequirement { version: number; }`,
			`export interface WorkflowSpec { version: number; entry: string; handlers: Record<string, LambdaHandle>; edges?: Array<{ from: string; outcome: "continue" | "present" | "challenge"; to: string; input: string }>; }`,
			`export type ProviderKind = "identity" | "invitation"; export type ProviderState = "virtual" | "durable"; export type ReplayProtection = "none" | "expiry" | "one_time"; export type RevocationMode = "none" | "key_rollover" | "durable";`,
			`export interface ProviderSpec { version: number; state: ProviderState; replayProtection: ReplayProtection; revocation: RevocationMode; handlers: Record<string, LambdaHandle>; }`,
			`export interface ProgramTestSpec { lambda: string; input: unknown; expectedKind: OutcomeKind; fakes?: Record<string, unknown>; } export interface ProgramBuilder { capabilities(requirements: Record<string, CapabilityRequirement>): void; lambda<I = unknown, C = Record<string, unknown>>(id: string, spec: LambdaSpec<I, C>): LambdaHandle; workflow(id: string, spec: WorkflowSpec): void; provider(kind: ProviderKind, name: string, spec: ProviderSpec): void; test(id: string, spec: ProgramTestSpec): void; }`,
			`export interface ResultBuilders { continue(handler: string): Outcome; present(spec: { handler: string; carry?: unknown; expiresInSeconds: number }): Outcome; challenge(spec: { handler: string; carry?: unknown; expiresInSeconds: number }): Outcome; commit(effects: unknown[]): Outcome; complete(value?: unknown): Outcome; deny(code: string): Outcome; skip(code?: string): Outcome; error(code: string): Outcome; }`,
			`export interface TinyIdpV1 { lambda<I = unknown, C = Record<string, unknown>>(id: string, spec: LambdaSpec<I, C>): LambdaHandle; program(name: string, define: (program: ProgramBuilder) => void): unknown; readonly result: ResultBuilders; readonly field: FieldBuilders; readonly action: ActionBuilders; }`,
			`export const v1: TinyIdpV1;`,
		},
	}
}
