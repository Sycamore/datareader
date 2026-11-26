package main

import (
	"encoding/json"
	"fmt"
	"github.com/kannanr/datareader"
	"os"
	"time"
)

func main() {
	f, e := os.Open(os.Args[1])
	if e != nil {
		panic(e)
	}
	defer f.Close()
	r, e := datareader.NewSAS7BDATReader(f)
	if e != nil {
		panic(e)
	}
	r.NoAlignCorrection = os.Getenv("NO_ALIGN") != ""
	r.ConvertDates = os.Getenv("SAS_DATES") != ""
	r.TrimStrings = true
	enc := json.NewEncoder(os.Stdout)
	if e = enc.Encode(map[string]interface{}{"rows": r.RowCount(), "names": r.ColumnNames(), "labels": r.ColumnLabels(), "formats": r.ColumnFormats, "types": r.ColumnTypes()}); e != nil {
		panic(e)
	}
	for remain := r.RowCount(); remain > 0; {
		n := 1024
		if remain < n {
			n = remain
		}
		cols, e := r.Read(n)
		if e != nil {
			panic(e)
		}
		if len(cols) == 0 || cols[0].Length() == 0 {
			panic("unexpected end of data")
		}
		for i := 0; i < cols[0].Length(); i++ {
			row := make([]interface{}, len(cols))
			for j, c := range cols {
				if c.Missing() != nil && c.Missing()[i] {
					continue
				}
				switch v := c.Data().(type) {
				case []float64:
					row[j] = v[i]
				case []string:
					row[j] = v[i]
				case []time.Time:
					row[j] = v[i].Format(time.RFC3339Nano)
				default:
					panic(fmt.Sprintf("unexpected type %T", v))
				}
			}
			if e = enc.Encode(row); e != nil {
				panic(e)
			}
		}
		remain -= cols[0].Length()
	}
}
