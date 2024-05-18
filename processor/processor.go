package processor

import (
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"

	"github.com/nikolaydubina/go-instrument/instrument"
	"golang.org/x/tools/go/ast/astutil"
)

// Instrumenter supplies ast of Go code that will be inserted and required dependencies.
type Instrumenter interface {
	Imports() []*types.Package
	PrefixStatements(spanName string, hasError bool) []ast.Stmt
}

// FunctionSelector tells if function has to be instrumented.
type FunctionSelector interface {
	AcceptFunction(functionName string) bool
}

type Processor interface {
	Process(fileName, app string, overwrite, defaultSelect, skipGenerated bool) error
}

func NewSerialProcessor(contextName, contextPackage, contextType, errorName, errorType string) *SerialProcessor {
	return &SerialProcessor{
		SpanName:       BasicSpanName,
		ContextName:    contextName,
		ContextPackage: contextPackage,
		ContextType:    contextType,
		ErrorName:      errorName,
		ErrorType:      errorType,
	}
}

// Processor traverses AST, collects details on functions and methods, and invokes Instrumenter
type SerialProcessor struct {
	Instrumenter     Instrumenter
	FunctionSelector FunctionSelector
	SpanName         func(receiver, function string) string
	ContextName      string
	ContextPackage   string
	ContextType      string
	ErrorName        string
	ErrorType        string
}

func (p *SerialProcessor) process(fset *token.FileSet, file *ast.File) error {
	var patches []patch

	astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
		if c == nil {
			return true
		}

		var receiver, fname string
		var fnType *ast.FuncType
		var fnBody *ast.BlockStmt

		switch fn := c.Node().(type) {
		case *ast.FuncLit:
			fnType, fnBody = fn.Type, fn.Body
			fname = "anonymous"
		case *ast.FuncDecl:
			fnType, fnBody = fn.Type, fn.Body
			fname = functionName(fn)
			receiver = methodReceiverTypeName(fn)
		default:
			return true
		}

		if !p.FunctionSelector.AcceptFunction(fname) {
			return true
		}

		if functionHasContext(fnType, p.ContextName, p.ContextPackage, p.ContextType) {
			ps := p.Instrumenter.PrefixStatements(p.SpanName(receiver, fname), functionHasError(fnType, p.ErrorName, p.ErrorType))
			patches = append(patches, patch{pos: fnBody.Pos(), stmts: ps})
		}

		return true
	})

	if len(patches) > 0 {
		if err := patchFile(fset, file, patches...); err != nil {
			return err
		}
		for _, pkg := range p.Instrumenter.Imports() {
			astutil.AddNamedImport(fset, file, pkg.Name(), pkg.Path())
		}
	}

	return nil
}

func (p *SerialProcessor) Process(fileName, app string, overwrite, defaultSelect, skipGenerated bool) error {
	if fileName == "" {
		return errors.New("missing arg: file name")
	}

	src, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	formattedSrc, err := format.Source(src)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, fileName, formattedSrc, parser.ParseComments)
	if err != nil {
		return err
	}
	if skipGenerated && ast.IsGenerated(file) {
		return errors.New("skipping generated file")
	}

	directives := GoBuildDirectivesFromFile(*file)
	for _, q := range directives {
		if q.SkipFile() {
			return nil
		}
	}

	commands, err := CommandsFromFile(*file)
	if err != nil {
		return err
	}

	p.FunctionSelector = NewMapFunctionSelectorFromCommands(defaultSelect, commands)

	p.Instrumenter = &instrument.OpenTelemetry{
		TracerName:  app,
		ContextName: "ctx",
		ErrorName:   "err",
	}

	if err := p.process(fset, file); err != nil {
		return err
	}

	var out io.Writer = io.Discard
	if overwrite {
		outf, err := os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer outf.Close()
		out = outf
	}

	return format.Node(out, fset, file)
}

// BasicSpanName is common notation of <class>.<method> or <pkg>.<func>
func BasicSpanName(receiver, function string) string {
	if receiver == "" {
		return function
	}
	return receiver + "." + function
}

func methodReceiverTypeName(fn *ast.FuncDecl) string {
	// function
	if fn == nil || fn.Recv == nil {
		return ""
	}
	// method
	for _, v := range fn.Recv.List {
		if v == nil {
			continue
		}
		t := v.Type
		// pointer receiver
		if v, ok := v.Type.(*ast.StarExpr); ok {
			t = v.X
		}
		// value/pointer receiver
		if v, ok := t.(*ast.Ident); ok {
			return v.Name
		}
	}
	return ""
}

func functionName(fn *ast.FuncDecl) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	return fn.Name.Name
}

func isContext(e *ast.Field, contextName, contextPackage, contextType string) bool {
	// anonymous arg
	// multiple symbols
	// strange symbol
	if e == nil || len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != contextName {
		return false
	}

	pkg := ""
	sym := ""

	if se, ok := e.Type.(*ast.SelectorExpr); ok && se != nil {
		if v, ok := se.X.(*ast.Ident); ok && v != nil {
			pkg = v.Name
		}
		if v := se.Sel; v != nil {
			sym = v.Name
		}
	}

	return pkg == contextPackage && sym == contextType
}

func isError(e *ast.Field, errorName, errorType string) bool {
	if e == nil {
		return false
	}
	// anonymous arg
	// multiple symbols
	// strange symbol
	if len(e.Names) != 1 || e.Names[0] == nil {
		return false
	}
	if e.Names[0].Name != errorName {
		return false
	}

	if v, ok := e.Type.(*ast.Ident); ok && v != nil {
		return v.Name == errorType
	}

	return false
}

func functionHasContext(fnType *ast.FuncType, contextName, contextPackage, contextType string) bool {
	if fnType == nil {
		return false
	}

	if ps := fnType.Params; ps != nil {
		for _, q := range ps.List {
			if isContext(q, contextName, contextPackage, contextType) {
				return true
			}
		}
	}

	return false
}

func functionHasError(fnType *ast.FuncType, errorName, errorType string) bool {
	if fnType == nil {
		return false
	}

	if rs := fnType.Results; rs != nil {
		for _, q := range rs.List {
			if isError(q, errorName, errorType) {
				return true
			}
		}
	}

	return false
}
