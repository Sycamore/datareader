package datareader

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRLELongInsert(t *testing.T) {
	got, err := rle_decompress(258, []byte{0x40, 240, 'A'})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bytes.Repeat([]byte{'A'}, 258)) {
		t.Fatalf("got %d bytes", len(got))
	}
}

func TestRLECommandsAndInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		input []byte
		want  []byte
	}{
		{append([]byte{0x01, 0}, bytes.Repeat([]byte{'x'}, 320)...), bytes.Repeat([]byte{'x'}, 320)},
		{append([]byte{0x10, 0}, bytes.Repeat([]byte{'x'}, 4160)...), bytes.Repeat([]byte{'x'}, 4160)},
		{append([]byte{0x20}, bytes.Repeat([]byte{'x'}, 96)...), bytes.Repeat([]byte{'x'}, 96)},
		{[]byte{0x50, 0}, bytes.Repeat([]byte{'@'}, 17)},
	} {
		got, err := rle_decompress(len(tc.want), tc.input)
		if err != nil || !bytes.Equal(got, tc.want) {
			t.Fatalf("command %x: %v", tc.input[0], err)
		}
	}
	for _, input := range [][]byte{{0x40}, {0x80}, {0x30}, {0x40, 255, 'a'}} {
		if _, err := rle_decompress(2, input); err == nil {
			t.Errorf("accepted %x", input)
		}
	}
}

func TestRLELongStrings(t *testing.T) {
	f, err := os.Open("test_files/regression/long_strings.sas7bdat")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := NewSAS7BDATReader(f)
	if err != nil {
		t.Fatal(err)
	}
	r.TrimStrings = true
	cols, err := r.Read(-1)
	if err != nil {
		t.Fatal(err)
	}
	got := cols[0].Data().([]string)[0]
	if got != strings.Repeat("A", 257) {
		t.Fatalf("column x47: %q", got)
	}
}
