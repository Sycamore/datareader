package datareader

import (
	"os"
	"testing"
)

func TestFlaggedPageTypes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		rows, cols int
		first      string
	}{
		{"comp_deleted.sas7bdat", 3, 4, ""},
		{"data_page_with_deleted.sas7bdat", 997, 3, "93"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open("test_files/regression/" + tc.name)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			r, err := NewSAS7BDATReader(f)
			if err != nil {
				t.Fatal(err)
			}
			r.TrimStrings = true
			if r.RowCount() != tc.rows || len(r.ColumnNames()) != tc.cols {
				t.Fatalf("metadata: %d rows %d cols", r.RowCount(), len(r.ColumnNames()))
			}
			cols, err := r.Read(-1)
			if err != nil {
				t.Fatal(err)
			}
			if cols[0].Length() != tc.rows {
				t.Fatalf("read %d rows", cols[0].Length())
			}
			if tc.first != "" {
				if got := cols[0].Data().([]string)[130]; got != tc.first {
					t.Fatalf("row 131: %q", got)
				}
			}
		})
	}
}
