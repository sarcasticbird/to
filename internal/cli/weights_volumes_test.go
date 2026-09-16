package cli

import (
	"strings"
	"testing"
)

func TestWeightsAndVolumes(t *testing.T) {
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"85kg", "lb"}, "187.4 lb\n"},
		{[]string{"12oz", "g"}, "340.2 g\n"},
		{[]string{"1st", "lb"}, "14 lb\n"},
		{[]string{"1g", "mg"}, "1000 mg\n"},
		{[]string{"500ml", "cup"}, "2.113 cup (US)\n"},
		{[]string{"2gal", "l"}, "7.571 l\n"},
		{[]string{"1cup", "floz"}, "8 floz (US)\n"},
		{[]string{"1tbsp", "tsp"}, "3 tsp (US)\n"},
		{[]string{"1qt", "pt"}, "2 pt (US)\n"},
		{[]string{"1", "fluid ounces", "ml"}, "29.57 ml\n"},
		{[]string{"2", "LITRES", "ml"}, "2000 ml\n"},
		{[]string{"85kg", "lb,", "500ml", "cup"}, "187.4 lb\n\n2.113 cup (US)\n"},
	} {
		code, out, err := run(tt.args...)
		if code != 0 || out != tt.want || err != "" {
			t.Errorf("Run(%q) = (%d, %q, %q), want %q", tt.args, code, out, err, tt.want)
		}
	}
}

func TestWeightVolumeMismatchIsAtomic(t *testing.T) {
	for _, args := range [][]string{{"1oz", "floz"}, {"1cup", "g"}, {"33mm,", "1kg", "l"}} {
		code, out, err := run(args...)
		if code != 2 || out != "" || !strings.Contains(err, "cannot convert") {
			t.Errorf("Run(%q) = (%d, %q, %q)", args, code, out, err)
		}
	}
}

func TestWeightVolumeSummaries(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"500mg", "500 mg\n  0.5 g\n  0.01764 oz\n"},
		{"12oz", "12 oz\n  340.2 g\n  0.75 lb\n"},
		{"85kg", "85 kg\n  187.4 lb\n  13.39 st\n"},
		{"5ml", "5 ml\n  1.014 tsp (US)\n  0.3381 tbsp (US)\n"},
		{"500ml", "500 ml\n  16.91 floz (US)\n  2.113 cup (US)\n"},
		{"2gal", "2 gal (US)\n  7.571 l\n  8 qt (US)\n"},
	} {
		code, out, err := run(tt.input)
		if code != 0 || out != tt.want || err != "" {
			t.Errorf("Run(%q) = (%d, %q, %q), want %q", tt.input, code, out, err, tt.want)
		}
	}
}

func TestWeightVolumeSummaryRange(t *testing.T) {
	for _, tt := range []struct{ input, want string }{
		{"1e308kg", "1e+308 kg\n  1.575e+307 st\n"},
		{"5e-324mg", "4.941e-324 mg\n"},
		{"5e-324ml", "4.941e-324 ml\n"},
	} {
		code, out, err := run(tt.input)
		if code != 0 || out != tt.want || err != "" {
			t.Errorf("Run(%q) = (%d, %q, %q), want %q", tt.input, code, out, err, tt.want)
		}
	}
	code, out, err := run("1e308kg", "lb")
	if code != 2 || out != "" || !strings.Contains(err, "outside supported range") {
		t.Fatalf("direct overflow = (%d, %q, %q)", code, out, err)
	}
}
