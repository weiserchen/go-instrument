package cmd

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListFileName(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "SingleFile",
			args:     []string{"../internal/testdata/walk/abc.go"},
			expected: []string{"../internal/testdata/walk/abc.go"},
		},
		{
			name: "MultipleFiles1",
			args: []string{
				"../internal/testdata/walk/abc.go",
				"../internal/testdata/walk/suv.go",
			},
			expected: []string{
				"../internal/testdata/walk/abc.go",
				"../internal/testdata/walk/suv.go",
			},
		},
		{
			name: "MultipleFiles2",
			args: []string{
				"../internal/testdata/walk/suv.go",
				"../internal/testdata/walk/xyz.go",
			},
			expected: []string{
				"../internal/testdata/walk/suv.go",
				"../internal/testdata/walk/xyz.go",
			},
		},
		{
			name: "MixedFileDirectory1",
			args: []string{
				"../internal/testdata/walk/dir1",
				"../internal/testdata/walk/xyz.go",
			},
			expected: []string{
				"../internal/testdata/walk/dir1/abcdir1.go",
				"../internal/testdata/walk/dir1/dir3/abcdir3.go",
				"../internal/testdata/walk/dir1/dir4/abcdir4.go",
				"../internal/testdata/walk/dir1/dir4/dir5/abcdir5.go",
				"../internal/testdata/walk/xyz.go",
			},
		},
		{
			name: "MixedFileDirectory2",
			args: []string{
				"../internal/testdata/walk/dir2",
				"../internal/testdata/walk/dir1/dir4",
				"../internal/testdata/walk/suv.go",
			},
			expected: []string{
				"../internal/testdata/walk/dir1/dir4/abcdir4.go",
				"../internal/testdata/walk/dir1/dir4/dir5/abcdir5.go",
				"../internal/testdata/walk/dir2/abcdir2.go",
				"../internal/testdata/walk/suv.go",
			},
		},
		{
			name: "AllFilesInDirectory",
			args: []string{
				"../internal/testdata/walk",
			},
			expected: []string{
				"../internal/testdata/walk/abc.go",
				"../internal/testdata/walk/dir1/abcdir1.go",
				"../internal/testdata/walk/dir1/dir3/abcdir3.go",
				"../internal/testdata/walk/dir1/dir4/abcdir4.go",
				"../internal/testdata/walk/dir1/dir4/dir5/abcdir5.go",
				"../internal/testdata/walk/dir2/abcdir2.go",
				"../internal/testdata/walk/suv.go",
				"../internal/testdata/walk/xyz.go",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := listFileNames(tc.args)
			assert.Nil(t, err)
			sort.Strings(result)
			assert.Equal(t, tc.expected, result)
		})
	}
}
