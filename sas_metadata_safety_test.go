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

func TestMetadataSubheaderRemainder(t *testing.T) {
	for _, width := range []int{4, 8} {
		for _, valid := range []bool{true, false} {
			page := make([]byte, 64)
			length := 4 + 2*width
			if !valid {
				page[width] = 1
			}
			s := &SAS7BDAT{ByteOrder: binary.LittleEndian, cachedPage: page, properties: &sasProperties{intLength: width}}
			if err := s.validateSubheaderRemainder(0, length); (err == nil) != valid {
				t.Fatalf("width=%d valid=%v: %v", width, valid, err)
			}
			if err := s.validateSubheaderRemainder(0, length-1); err == nil {
				t.Fatalf("width=%d: short subheader accepted", width)
			}
		}
	}
}

func TestInvalidColumnAttributeType(t *testing.T) {
	page := make([]byte, 64)
	binary.LittleEndian.PutUint16(page[4:], 20)
	page[22] = 3
	s := &SAS7BDAT{ByteOrder: binary.LittleEndian, cachedPage: page, properties: &sasProperties{intLength: 4}}
	if err := s.processColumnAttributesSubheader(0, 32); err == nil {
		t.Fatal("invalid column type accepted")
	}
}
