package datareader

import "testing"

func TestSASFormatClass(t *testing.T) {
	for _, tc := range []struct {
		format string
		want   FormatClass
	}{
		{"DATE9.", FormatDate}, {"IS8601DA", FormatDate}, {"DTDATE", FormatDate},
		{"DATETIME20.6", FormatDateTime}, {"B8601DN", FormatDateTime},
		{"TIME12.6", FormatTime}, {"TOD", FormatTime},
		{"MINGUO", FormatNone}, {"DATEFOO", FormatNone}, {"date9.", FormatNone},
	} {
		if got := SASFormatClass(tc.format); got != tc.want {
			t.Errorf("%q: %q != %q", tc.format, got, tc.want)
		}
	}
}
