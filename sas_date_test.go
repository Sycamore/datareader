package datareader

import (
	"os"
	"testing"
	"time"
)

func TestDateConversionOptInAndRange(t *testing.T) {
	for _, convert := range []bool{false, true} {
		f, err := os.Open("test_files/regression/datetime.sas7bdat")
		if err != nil {
			t.Fatal(err)
		}
		r, err := NewSAS7BDATReader(f)
		if err != nil {
			t.Fatal(err)
		}
		r.ConvertDates = convert
		cols, err := r.Read(-1)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !convert {
			if _, ok := cols[1].Data().([]float64); !ok {
				t.Fatal("DATE converted without opt-in")
			}
			continue
		}
		dates := cols[1].Data().([]time.Time)
		if got := dates[3].Format("2006-01-02"); got != "2262-04-11" {
			t.Errorf("date %s", got)
		}
		times := cols[3].Data().([]time.Time)
		if got := times[2].Format("2006-01-02T15:04:05.000000"); got != "2016-02-29T23:59:59.123456" {
			t.Errorf("datetime %s", got)
		}
	}
}
