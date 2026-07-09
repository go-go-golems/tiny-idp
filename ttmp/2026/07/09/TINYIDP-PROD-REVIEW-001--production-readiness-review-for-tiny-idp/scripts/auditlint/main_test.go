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
