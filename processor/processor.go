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
	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/ast/astutil"
)

var (
	// for testing purpose
	defaultOut io.Writer = os.Stdout
)

var (
	ErrInvalidConfigType = errors.New("invalid config type")
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
	Process(fileNames []string, config ...any) error
}

type Task struct {
	FileName string
	Config   TraceConfig
	ErrCh    chan error
}

func NewTraceProcessor(pattern TracePattern) *TraceProcessor {
	return &TraceProcessor{
		SpanName: BasicSpanName,
		Pattern:  pattern,
	}
}

// TraceProcessor traverses AST, collects details on functions and methods, and invokes Instrumenter
type TraceProcessor struct {
	Instrumenter     Instrumenter
	FunctionSelector FunctionSelector
	SpanName         func(receiver, function string) string
	Pattern          TracePattern
}

func (p *TraceProcessor) Process(fileName string, config ...any) error {
	conf, ok := config[0].(TraceConfig)
	if !ok {
		return ErrInvalidConfigType
	}

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
	if conf.SkipGenerated && ast.IsGenerated(file) {
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

	p.FunctionSelector = NewMapFunctionSelectorFromCommands(conf.DefaultSelect, commands)

	p.Instrumenter = &instrument.OpenTelemetry{
		TracerName:  conf.App,
		ContextName: "ctx",
		ErrorName:   "err",
	}

	if err := p.process(fset, file); err != nil {
		return err
	}

	var out io.Writer = defaultOut
	if conf.Overwrite {
		outf, err := os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC, 0)
		if err != nil {
			return err
		}
		defer outf.Close()
		out = outf
	}

	return format.Node(out, fset, file)
}

func (p *TraceProcessor) process(fset *token.FileSet, file *ast.File) error {
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

		if functionHasContext(fnType, p.Pattern.ContextName, p.Pattern.ContextPackage, p.Pattern.ContextType) {
			ps := p.Instrumenter.PrefixStatements(p.SpanName(receiver, fname), functionHasError(fnType, p.Pattern.ErrorName, p.Pattern.ErrorType))
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

func NewSerialTraceProcessor(pattern TracePattern) *SerialTraceProcessor {
	return &SerialTraceProcessor{
		Pattern: pattern,
	}
}

type SerialTraceProcessor struct {
	Pattern TracePattern
}

func (p *SerialTraceProcessor) Process(fileNames []string, config ...any) error {
	conf, ok := config[0].(TraceConfig)
	if !ok {
		return ErrInvalidConfigType
	}

	fp := NewTraceProcessor(p.Pattern)
	for _, fileName := range fileNames {
		err := fp.Process(fileName, conf)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewParallelTraceProcessor(worker int, pattern TracePattern) *ParallelTraceProcessor {
	taskCh := make(chan Task)
	doneCh := make(chan bool)

	for i := 0; i < worker; i++ {
		go func() {
			p := NewTraceProcessor(pattern)
			for {
				select {
				case task := <-taskCh:
					task.ErrCh <- p.Process(task.FileName, task.Config)
				case <-doneCh:
					return
				}
			}
		}()
	}

	return &ParallelTraceProcessor{
		Worker:  worker,
		Pattern: pattern,
		TaskCh:  taskCh,
	}
}

type ParallelTraceProcessor struct {
	Worker  int
	Pattern TracePattern
	TaskCh  chan Task
	DoneCh  chan bool
}

func (p *ParallelTraceProcessor) Process(fileNames []string, config ...any) error {
	conf, ok := config[0].(TraceConfig)
	if !ok {
		return ErrInvalidConfigType
	}

	run := func() error {
		var g errgroup.Group

		for _, fileName := range fileNames {
			g.Go(func() error {
				errCh := make(chan error)
				task := Task{
					FileName: fileName,
					Config:   conf,
					ErrCh:    errCh,
				}

				p.TaskCh <- task
				return <-errCh
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
		return nil
	}

	return run()
}
