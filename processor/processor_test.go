package processor

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
)

const BenchFileCount = 1

func BenchmarkProcessorFile(b *testing.B) {
	tempDir := setupFiles(b)

	pattern := fmt.Sprintf("%s/*.go", tempDir)
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// fmt.Println("abc")
		for _, fname := range filenames {
			p := NewSerialProcessor("ctx", "context", "Context", "err", "error")
			err := p.Process(fname, "test", false, true, false)
			if err != nil {
				b.Error(err)
			}
		}
	}
}

func setupFiles(b *testing.B) string {
	b.Helper()

	buf, err := os.ReadFile("../internal/testdata/basic.go")
	if err != nil {
		b.Fatal(err)
	}

	// fmt.Print(string(buf))

	tempDir := b.TempDir()
	for i := 0; i < BenchFileCount; i++ {
		filepath := path.Join(tempDir, strconv.Itoa(i)+".go")
		err := os.WriteFile(filepath, buf, 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	return tempDir
}
