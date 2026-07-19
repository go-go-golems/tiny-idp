package main

const (
	applicationName = "tinyidp-message-app"
	clientID        = "tinyidp-message-app"
	issuerPath      = "/idp"
	callbackPath    = "/auth/callback"
	appCookieName   = "tinymsg_app_session"
	registerCookie  = "tinymsg_registration"
)

var routeContract = []string{
	"GET /",
	"GET /auth/login",
	"GET /auth/register",
	"GET /auth/callback",
	"POST /auth/logout",
	"GET /api/session",
	"GET /api/registration",
	"POST /api/accounts",
	"GET /api/messages",
	"POST /api/messages",
	"GET /healthz",
	"GET /readyz",
}

var securityInvariantTests = []string{
	"TestExternalImportBoundary",
	"TestStateRootPermissions",
	"TestMigrationChecksumMismatchFailsStartup",
	"TestLoginAttemptConsumesOnce",
	"TestRegistrationAttemptConsumesOnce",
	"TestSessionStoresOnlyTokenHash",
	"TestCallbackRejectsReplay",
	"TestCallbackValidatesNonce",
	"TestReturnToRejectsExternalAndAmbiguousPaths",
	"TestRegistrationRequiresPreSessionCSRF",
	"TestMessageCreateRequiresSessionCSRF",
	"TestMessageAuthorComesFromVerifiedSession",
	"TestMessageTextIsNeverRenderedAsMarkup",
	"TestUnsafeRequestsRejectForeignOrigin",
	"TestSensitiveResponsesAreNoStore",
}
