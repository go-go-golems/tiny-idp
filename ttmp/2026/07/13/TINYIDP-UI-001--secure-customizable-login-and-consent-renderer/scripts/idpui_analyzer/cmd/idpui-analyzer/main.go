package main

import (
	"github.com/go-go-golems/tiny-idp/ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/scripts/idpui_analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(idpuianalyzer.Analyzer)
}
