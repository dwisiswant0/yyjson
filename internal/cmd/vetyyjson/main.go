package main

import (
	"go.dw1.io/yyjson/internal/cmd/vetyyjson/internal/unsafeptrpolicy"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
)

var analyzers = []*analysis.Analyzer{
	handwritten(appends.Analyzer),
	handwritten(asmdecl.Analyzer),
	handwritten(assign.Analyzer),
	handwritten(atomic.Analyzer),
	handwritten(bools.Analyzer),
	handwritten(buildtag.Analyzer),
	handwritten(cgocall.Analyzer),
	handwritten(composite.Analyzer),
	handwritten(copylock.Analyzer),
	handwritten(deepequalerrors.Analyzer),
	handwritten(defers.Analyzer),
	handwritten(directive.Analyzer),
	handwritten(errorsas.Analyzer),
	handwritten(framepointer.Analyzer),
	handwritten(hostport.Analyzer),
	handwritten(httpmux.Analyzer),
	handwritten(httpresponse.Analyzer),
	handwritten(ifaceassert.Analyzer),
	handwritten(loopclosure.Analyzer),
	handwritten(lostcancel.Analyzer),
	handwritten(nilfunc.Analyzer),
	handwritten(printf.Analyzer),
	handwritten(shift.Analyzer),
	handwritten(sigchanyzer.Analyzer),
	handwritten(slog.Analyzer),
	handwritten(sortslice.Analyzer),
	handwritten(stdmethods.Analyzer),
	handwritten(stdversion.Analyzer),
	handwritten(stringintconv.Analyzer),
	handwritten(structtag.Analyzer),
	handwritten(testinggoroutine.Analyzer),
	handwritten(tests.Analyzer),
	handwritten(timeformat.Analyzer),
	handwritten(unmarshal.Analyzer),
	handwritten(unreachable.Analyzer),
	handwritten(unusedresult.Analyzer),
	handwritten(unusedwrite.Analyzer),
	handwritten(waitgroup.Analyzer),
	unsafeptrpolicy.Analyzer,
}

func main() {
	multichecker.Main(analyzers...)
}

func handwritten(analyzer *analysis.Analyzer) *analysis.Analyzer {
	wrapped := *analyzer
	wrapped.Run = func(pass *analysis.Pass) (any, error) {
		if unsafeptrpolicy.IsGeneratedPackage(pass.Pkg.Path()) {
			generatedPass := *pass
			generatedPass.Report = func(analysis.Diagnostic) {}
			return analyzer.Run(&generatedPass)
		}
		return analyzer.Run(pass)
	}
	return &wrapped
}
