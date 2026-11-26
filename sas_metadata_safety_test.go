package datareader

import (
	"encoding/binary"
	"testing"
)

func TestInvalidFormatTextReferenceReturnsError(t *testing.T) {
	page := make([]byte, 128)
	binary.LittleEndian.PutUint16(page[40:], 75)
	binary.LittleEndian.PutUint16(page[42:], 20)
	s := &SAS7BDAT{ByteOrder: binary.LittleEndian, cachedPage: page, properties: &sasProperties{intLength: 4}, columnNamesStrings: []string{"short"}}
	if err := s.processFormatSubheader(0, 64); err == nil {
		t.Fatal("invalid reference accepted")
	}
}
