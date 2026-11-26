package datareader

import (
	"bytes"
	"encoding/binary"
	"os"
	"reflect"
	"testing"
)

func TestSLCSignatures(t *testing.T) {
	data, err := os.ReadFile("test_files/data/test7.sas7bdat")
	if err != nil {
		t.Fatal(err)
	}
	original, err := NewSAS7BDATReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if !original.U64 || original.ByteOrder != binary.LittleEndian {
		t.Skip("fixture is not 64-bit little-endian")
	}
	p := original.properties
	patched := bytes.Clone(data)
	seen := map[uint32]bool{}
	rowOffset := 0
	for page := 0; page < p.pageCount; page++ {
		base := p.headerLength + page*p.pageLength
		if base+p.pageLength > len(data) {
			break
		}
		count := int(binary.LittleEndian.Uint16(data[base+p.pageBitOffset+subheader_count_offset:]))
		if count > p.pageLength/p.subheaderPointerLength {
			continue
		}
		for i := 0; i < count; i++ {
			pos := base + p.pageBitOffset + subheader_pointers_offset + i*p.subheaderPointerLength
			if pos+8 > base+p.pageLength {
				break
			}
			off := int(binary.LittleEndian.Uint64(data[pos:]))
			if off < 0 || off+8 > p.pageLength {
				continue
			}
			sig := binary.LittleEndian.Uint32(data[base+off:])
			if sig == 0xf7f7f7f7 || sig == 0xf6f6f6f6 {
				copy(patched[base+off+4:base+off+8], []byte{255, 255, 255, 255})
				seen[sig] = true
				if sig == 0xf7f7f7f7 {
					rowOffset = base + off
				}
			}
		}
	}
	if len(seen) != 2 {
		t.Fatalf("missing row or column signature: %v", seen)
	}
	modified, err := NewSAS7BDATReader(bytes.NewReader(patched))
	if err != nil {
		t.Fatal(err)
	}
	a, err := original.Read(-1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := modified.Read(-1)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || !reflect.DeepEqual(original.ColumnNames(), modified.ColumnNames()) {
		t.Fatal("patched file differs")
	}
	for i := range a {
		if ok, _ := a[i].AllClose(b[i], 0); !ok {
			t.Fatalf("column %d differs", i)
		}
	}
	unknown := bytes.Clone(data)
	copy(unknown[rowOffset:rowOffset+4], []byte{1, 2, 3, 4})
	if _, err := NewSAS7BDATReader(bytes.NewReader(unknown)); err == nil {
		t.Fatal("unknown signature accepted")
	}
}

func TestSignatureClassification(t *testing.T) {
	for _, tc := range []struct {
		order binary.ByteOrder
		sig   []byte
		want  int
		bad   bool
	}{
		{binary.LittleEndian, []byte{0xf8, 0xff, 0xff, 0xff, 0, 0, 0, 0}, subheaderCountsIndex, false},
		{binary.BigEndian, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xf9}, subheaderCountsIndex, false},
		{binary.BigEndian, []byte{0xf7, 0xf7, 0xf7, 0xf7, 0xff, 0xff, 0xff, 0xff}, rowSizeIndex, false},
		{binary.BigEndian, []byte{0, 0, 0, 0, 0xf7, 0xf7, 0xf7, 0xf7}, rowSizeIndex, false},
		{binary.BigEndian, []byte{0, 0, 0, 0, 0xf6, 0xf6, 0xf6, 0xf6}, columnSizeIndex, false},
		{binary.LittleEndian, []byte{1, 2, 3, 4, 0, 0, 0, 0}, 0, true},
	} {
		s := &SAS7BDAT{ByteOrder: tc.order}
		got, err := s.getSubheaderIndex(tc.sig, 0, 0)
		if (err != nil) != tc.bad || (!tc.bad && got != tc.want) {
			t.Errorf("signature %x: got %d, %v", tc.sig, got, err)
		}
		if !tc.bad && got == subheaderCountsIndex {
			if err := s.processSubheader(got, &subheaderPointer{}); err != nil {
				t.Fatal(err)
			}
		}
	}
}
