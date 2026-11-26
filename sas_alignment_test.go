package datareader

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func TestShortNumericByteOrder(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for width := 3; width <= 8; width++ {
			full := make([]byte, 8)
			order.PutUint64(full, math.Float64bits(3.5))
			var src []byte
			if order == binary.LittleEndian {
				src = full[8-width:]
			} else {
				src = full[:width]
			}
			s := &SAS7BDAT{ByteOrder: order, cachedPage: src, properties: &sasProperties{columnCount: 1}, columnDataLengths: []int{width}, columnDataOffsets: []int{0}, columns: []*column{{ctype: SASNumericType}}, bytechunk: [][]byte{make([]byte, 8)}}
			if err := s.processByteArrayWithData(0, width); err != nil {
				t.Fatal(err)
			}
			if got := math.Float64frombits(order.Uint64(s.bytechunk[0])); got != 3.5 {
				t.Errorf("%v width %d: %g", order, width, got)
			}
		}
	}
}

func TestStatTransferMixedPageShortNumerics(t *testing.T) {
	f, err := os.Open("test_files/data/types.sas7bdat")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := NewSAS7BDATReader(f)
	if err != nil {
		t.Fatal(err)
	}
	cols, err := r.Read(-1)
	if err != nil {
		t.Fatal(err)
	}
	for j, want := range []float64{3.140000104904175, 3.14, 2, 2, 2} {
		got := cols[j].Data().([]float64)[0]
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("column %d = %g, want %g", j, got, want)
		}
	}
	if got := cols[5].Data().([]string)[0]; got != "2" {
		t.Errorf("string = %q", got)
	}
}
