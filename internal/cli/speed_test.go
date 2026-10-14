package cli

import (
	"strings"
	"testing"
)

func TestSpeedCLIForms(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"attached", []string{"36km/h", "m/s"}, "10 m/s\n"},
		{"attached highway speed", []string{"100km/h", "mph"}, "62.14 mph\n"},
		{"separated highway speed", []string{"100", "km/h", "mph"}, "62.14 mph\n"},
		{"separated", []string{"60", "mph", "km/h"}, "96.56 km/h\n"},
		{"quoted source", []string{"36", "kilometres per hour", "mps"}, "10 m/s\n"},
		{"quoted target", []string{"10m/s", "miles per hour"}, "22.37 mph\n"},
		{"quoted measurement", []string{"10 feet per second", "meters per second"}, "3.048 m/s\n"},
		{"uppercase symbols", []string{"10KN", "KM/H"}, "18.52 km/h\n"},
		{"explicit feet per second", []string{"1m/s", "ft/s"}, "3.281 ft/s\n"},
		{"negative", []string{"-36kmph", "mps"}, "-10 m/s\n"},
		{"zero", []string{"0e-999fps", "kt"}, "0 kn\n"},
		{"large representable", []string{"1e308kph", "mph"}, "6.214e+307 mph\n"},
		{"subnormal representable", []string{"5e-324km/h", "mph"}, "4.941e-324 mph\n"},
		{"self conversion edges", []string{"1.7976931348623157e308kn", "kn,", "-5e-324ft/s", "ft/s"}, "1.798e+308 kn\n\n-4.941e-324 ft/s\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, err := run(tt.args...)
			if code != 0 || out != tt.want || err != "" {
				t.Errorf("Run(%q) = (%d, %q, %q), want (0, %q, empty)", tt.args, code, out, err, tt.want)
			}
		})
	}
}

func TestSpeedCLISummariesAndBatches(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"attached summary", []string{"1m/s"}, "1 m/s\n  3.6 km/h\n  2.237 mph\n  1.944 kn\n"},
		{"separated alias summary", []string{"1", "knots"}, "1 kn\n  0.5144 m/s\n  1.852 km/h\n  1.151 mph\n"},
		{"overflow summary", []string{"6e307m/s"}, "6e+307 m/s\n  1.342e+308 mph\n  1.166e+308 kn\n"},
		{"underflow summary", []string{"-5e-324ft/s"}, "-4.941e-324 ft/s\n  -4.941e-324 km/h\n  -4.941e-324 mph\n  -4.941e-324 kn\n"},
		{"mixed dimensions", []string{"36km/h", "m/s,", "1m", "cm,", "1kg", "g,", "1l", "ml,", "32f", "c"}, "10 m/s\n\n100 cm\n\n1000 g\n\n1000 ml\n\n0 \u00b0C\n"},
		{"quoted aliases in batch", []string{"36", "kilometers per hour", "metres per second,", "1", "feet per second", "km/h"}, "10 m/s\n\n1.097 km/h\n"},
		{"compact comma batch", []string{"36km/h,1kn"}, "36 km/h\n  10 m/s\n  22.37 mph\n  19.44 kn\n\n1 kn\n  0.5144 m/s\n  1.852 km/h\n  1.151 mph\n"},
		{"summary and direct batch", []string{"1m/s,", "1ft/s", "m/s"}, "1 m/s\n  3.6 km/h\n  2.237 mph\n  1.944 kn\n\n0.3048 m/s\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, err := run(tt.args...)
			if code != 0 || out != tt.want || err != "" {
				t.Errorf("Run(%q) = (%d, %q, %q), want (0, %q, empty)", tt.args, code, out, err, tt.want)
			}
		})
	}
}

func TestSpeedCLIErrorsAreAtomic(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"speed to length", []string{"1m/s", "m"}, "cannot convert speed to length"},
		{"length to speed", []string{"1m", "m/s"}, "cannot convert length to speed"},
		{"mismatch in mixed batch", []string{"1kg", "g,", "1mph", "c"}, "measurement 2: cannot convert speed to temperature"},
		{"overflow", []string{"1e308m/s", "km/h"}, "outside supported range"},
		{"negative overflow in batch", []string{"1m", "cm,", "-1e308m/s", "mph"}, "measurement 2: convert"},
		{"underflow", []string{"5e-324km/h", "m/s"}, "outside supported range"},
		{"negative underflow in batch", []string{"1l", "ml,", "-5e-324ft/s", "m/s"}, "outside supported range"},
		{"input underflow", []string{"1m/s,", "1e-999kn"}, "outside supported range"},
		{"negative input underflow", []string{"-1e-999", "mph"}, "outside supported range"},
		{"input overflow", []string{"1m/s,", "1e999mph"}, "invalid measurement"},
		{"invalid middle expression", []string{"1m/s,", "nonsense,", "1kg"}, "measurement 2: invalid measurement"},
		{"unknown target", []string{"1m/s,", "10mph", "parsecs per hour"}, "measurement 2: unknown unit"},
		{"empty first expression", []string{",1m/s"}, "measurement 1 is empty"},
		{"empty middle expression", []string{"1m/s,,1kg"}, "measurement 2 is empty"},
		{"empty last expression", []string{"1m/s,"}, "measurement 2 is empty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, out, err := run(tt.args...)
			if code != 2 || out != "" || !strings.HasPrefix(err, "to: ") || !strings.Contains(err, tt.want) {
				t.Errorf("Run(%q) = (%d, %q, %q), want code 2, no stdout, and error containing %q", tt.args, code, out, err, tt.want)
			}
		})
	}
}

func TestSpeedHelp(t *testing.T) {
	code, out, err := run("--help")
	if code != 0 || err != "" {
		t.Fatalf("--help = (%d, %q, %q)", code, out, err)
	}
	for _, text := range []string{
		"Speed:       m/s, km/h, mph, ft/s, kn",
		"meters/metres per second, mps",
		"kilometers/kilometres per hour, kph, kmph",
		"mile/miles per hour", "foot/feet per second, fps", "knot/knots, kt",
		"1 km/h = 1/3.6 m/s", "1 mph = 0.44704 m/s",
		"1 ft/s = 0.3048 m/s", "1 kn = 1852/3600 m/s",
		"omitting the source unit", "out-of-range equivalents",
		"ft/s is available as an explicit target", "to 100km/h mph", "to 100 km/h mph",
		`to 10 "metres per second" "miles per hour"`,
	} {
		if !strings.Contains(out, text) {
			t.Errorf("--help missing %q", text)
		}
	}
}
