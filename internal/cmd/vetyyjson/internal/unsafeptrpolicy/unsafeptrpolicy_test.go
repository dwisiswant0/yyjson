package unsafeptrpolicy_test

import (
	"testing"

	"go.dw1.io/yyjson/internal/cmd/vetyyjson/internal/unsafeptrpolicy"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), unsafeptrpolicy.Analyzer,
		"go.dw1.io/yyjson/internal/bind",
		"go.dw1.io/yyjson/internal/other",
		"go.dw1.io/yyjson/internal/yyjsonc",
	)
}
