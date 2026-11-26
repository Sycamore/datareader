package datareader

import (
	"os"
	"testing"
)

func TestSASCharacterEncoding(t *testing.T) {
	for _, tc := range []struct{ file, name, value string }{
		{"testbig5.sas7bdat", "VAR1", "§Ú·R§A"},
		{"chinese_column_works.sas7bdat", "测试", ""},
	} {
		t.Run(tc.file, func(t *testing.T) {
			f, e := os.Open("test_files/regression/" + tc.file)
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			r, e := NewSAS7BDATReader(f)
			if e != nil {
				t.Fatal(e)
			}
			if got := r.ColumnNames()[0]; got != tc.name {
				t.Fatalf("column name %q", got)
			}
			if tc.value != "" {
				r.TrimStrings = true
				c, e := r.Read(1)
				if e != nil {
					t.Fatal(e)
				}
				if got := c[0].Data().([]string)[0]; got != tc.value {
					t.Fatalf("value %q", got)
				}
			}
		})
	}
}
