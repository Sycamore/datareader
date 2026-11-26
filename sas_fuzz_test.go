package datareader

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func FuzzNewSAS7BDATReader(f *testing.F) {
	for _, pattern := range []string{
		"test_files/data/*.sas7bdat", "test_files/regression/*.sas7bdat",
		"/Users/kannanramakrishnan/projects/sycamoreinformatics/s-caf/testdata/sas7bdat_files/*.sas7bdat",
	} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			f.Fatal(err)
		}
		for _, path := range paths {
			file, err := os.Open(path)
			if err != nil {
				f.Fatal(err)
			}
			data, err := io.ReadAll(io.LimitReader(file, 64<<10))
			closeErr := file.Close()
			if err != nil {
				f.Fatal(err)
			}
			if closeErr != nil {
				f.Fatal(closeErr)
			}
			f.Add(data)
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := NewSAS7BDATReader(bytes.NewReader(data))
		if err != nil {
			return
		}
		r.TrimStrings = true
		r.ConvertDates = true
		for {
			cols, err := r.Read(1024)
			if err != nil {
				if err != io.EOF {
					return
				}
				return
			}
			if len(cols) == 0 || cols[0].Length() == 0 {
				return
			}
		}
	})
}
