package processor

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
)

const (
	BenchSerailCount    = 1
	BenchParallelCount  = 10000
	BenchParallelWorker = 32
)

func BenchmarkTraceProcessor(b *testing.B) {
	tempDir := setupFiles(b, BenchSerailCount)

	filePattern := fmt.Sprintf("%s/*.go", tempDir)
	filenames, err := filepath.Glob(filePattern)
	if err != nil {
		b.Fatal(err)
	}

	defaultOut = io.Discard
	defer func() {
		defaultOut = os.Stdout
	}()

	b.ResetTimer()
	tracePattern := TracePattern{
		ContextName:    "ctx",
		ContextPackage: "context",
		ContextType:    "Context",
		ErrorName:      "err",
		ErrorType:      "error",
	}
	p := NewTraceProcessor(tracePattern)
	for i := 0; i < b.N; i++ {
		for _, fname := range filenames {
			err := p.Process(fname, "test", false, true, false)
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkSerialProcessor(b *testing.B) {
	tempDir := setupFiles(b, BenchParallelCount)

	filePattern := fmt.Sprintf("%s/*.go", tempDir)
	filenames, err := filepath.Glob(filePattern)
	if err != nil {
		b.Fatal(err)
	}

	defaultOut = io.Discard
	defer func() {
		defaultOut = os.Stdout
	}()

	b.ResetTimer()
	tracePattern := TracePattern{
		ContextName:    "ctx",
		ContextPackage: "context",
		ContextType:    "Context",
		ErrorName:      "err",
		ErrorType:      "error",
	}
	p := NewSerialTraceProcessor(tracePattern)
	for i := 0; i < b.N; i++ {
		err := p.Process(filenames, "test", false, true, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkParallelProcessor(b *testing.B) {
	tempDir := setupFiles(b, BenchParallelCount)

	filePattern := fmt.Sprintf("%s/*.go", tempDir)
	filenames, err := filepath.Glob(filePattern)
	if err != nil {
		b.Fatal(err)
	}

	defaultOut = io.Discard
	defer func() {
		defaultOut = os.Stdout
	}()

	b.ResetTimer()
	tracePattern := TracePattern{
		ContextName:    "ctx",
		ContextPackage: "context",
		ContextType:    "Context",
		ErrorName:      "err",
		ErrorType:      "error",
	}
	p := NewParallelTraceProcessor(BenchParallelWorker, tracePattern)
	for i := 0; i < b.N; i++ {
		err := p.Process(filenames, "test", false, true, false)
		if err != nil {
			b.Error(err)
		}
	}
}

func setupFiles(b *testing.B, count int) string {
	b.Helper()

	buf, err := os.ReadFile("../internal/testdata/basic.go")
	if err != nil {
		b.Fatal(err)
	}

	tempDir := b.TempDir()
	for i := 0; i < count; i++ {
		filepath := path.Join(tempDir, strconv.Itoa(i)+".go")
		err := os.WriteFile(filepath, buf, 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	return tempDir
}
