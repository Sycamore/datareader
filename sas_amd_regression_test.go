package datareader

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestTrailingAMDTextReferences(t *testing.T) {
	f, err := os.Open("test_files/regression/medium_real_data.sas7bdat")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := NewSAS7BDATReader(f)
	if err != nil {
		t.Fatal(err)
	}
	if r.RowCount() != 46641 {
		t.Fatalf("rows: %d", r.RowCount())
	}
	want := []string{"RAS", "RAH", "RAD", "JRAS", "JRAD", "CONTROL"}
	if !reflect.DeepEqual(r.ColumnNames(), want) {
		t.Fatalf("names: %v", r.ColumnNames())
	}
	if r.ColumnLabels()[0] != "Type of alteration/repair - 1999 version" {
		t.Fatalf("labels: %v", r.ColumnLabels())
	}
	cols, err := r.Read(1)
	if err != nil {
		t.Fatal(err)
	}
	if got := cols[2].Data().([]float64)[0]; got != 600 {
		t.Fatalf("RAD: %v", got)
	}
}

func TestInvalidWindows1252Rows(t *testing.T) {
	for i := 19; i <= 21; i++ {
		f, err := os.Open(fmt.Sprintf("test_files/data/test%d.sas7bdat", i))
		if err != nil {
			t.Fatal(err)
		}
		r, err := NewSAS7BDATReader(f)
		if err == nil {
			_, err = r.Read(-1)
		}
		f.Close()
		if err == nil {
			t.Errorf("test%d: expected invalid encoding error", i)
		}
	}
}

func TestTruncatedSASDoesNotPanic(t *testing.T) {
	for _, name := range []string{"test_files/data/test1.sas7bdat", "test_files/regression/medium_real_data.sas7bdat", "test_files/regression/comp_deleted.sas7bdat"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range []int{0, 100, 287, 288, len(data) / 2, len(data) - 1} {
			t.Run(fmt.Sprintf("%s/%d", name, n), func(t *testing.T) {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("panic: %v", p)
					}
				}()
				r, err := NewSAS7BDATReader(bytes.NewReader(data[:n]))
				if err == nil {
					_, err = r.Read(-1)
				}
				if err == nil {
					t.Error("expected truncation error")
				}
			})
		}
	}
}
