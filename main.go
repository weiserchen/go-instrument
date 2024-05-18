package main

import (
	"flag"
	"os"

	"github.com/nikolaydubina/go-instrument/processor"
)

func main() {
	var (
		fileName      string
		overwrite     bool
		app           string
		defaultSelect bool
		skipGenerated bool
	)
	flag.StringVar(&fileName, "filename", "", "go file to instrument")
	flag.StringVar(&app, "app", "app", "name of application")
	flag.BoolVar(&overwrite, "w", false, "overwrite original file")
	flag.BoolVar(&defaultSelect, "all", true, "instrument all by default")
	flag.BoolVar(&skipGenerated, "skip-generated", false, "skip generated files")
	flag.Parse()

	pattern := &processor.TracePattern{
		ContextName:    "ctx",
		ContextPackage: "context",
		ContextType:    "Context",
		ErrorName:      "err",
		ErrorType:      "error",
	}
	p := processor.NewTraceProcessor(pattern)
	config := processor.TraceConfig{
		App:           app,
		Overwrite:     overwrite,
		DefaultSelect: defaultSelect,
		SkipGenerated: skipGenerated,
	}

	if err := p.Process(fileName, config); err != nil {
		os.Stderr.WriteString(err.Error())
		os.Exit(1)
	}
}
