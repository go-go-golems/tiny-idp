package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestInternalAPIAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), internalAPIAnalyzer, "fixture/pkg/public")
}

func TestIgnoredRandAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ignoredRandAnalyzer, "fixture/checks/randcheck")
}

func TestHTTPServerAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), httpServerAnalyzer, "fixture/checks/httpcheck")
}

func TestUnusedConfigAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), unusedConfigAnalyzer, "fixture/pkg/config")
}

func TestAtomicityAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), atomicityAnalyzer, "fixture/internal/admin")
}

func TestBearerTransportAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), bearerTransportAnalyzer, "fixture/checks/bearercheck")
}

func TestSecurityClockAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), securityClockAnalyzer, "fixture/checks/clockcheck")
}

func TestStrictSecurityParseAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), strictSecurityParseAnalyzer, "fixture/checks/strictparsecheck")
}

func TestInteractionContinuationAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), interactionContinuationAnalyzer, "fixture/checks/continuationcheck")
}

func TestProtocolLifecycleAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), protocolLifecycleAnalyzer, "fixture/checks/lifecyclecheck")
}

func TestIgnoredSecurityErrorAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), ignoredSecurityErrorAnalyzer, "fixture/checks/securityerrorcheck")
}
