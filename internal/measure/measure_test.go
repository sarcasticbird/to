package measure

import (
	"math"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantValue  float64
		wantSource string
		wantTarget string
		wantErr    string
	}{
		{name: "attached millimeters", args: []string{"33mm"}, wantValue: 33, wantSource: "mm"},
		{name: "attached source and target", args: []string{"33mm", "inches"}, wantValue: 33, wantSource: "mm", wantTarget: "in"},
		{name: "separated source", args: []string{"33", "millimeters"}, wantValue: 33, wantSource: "mm"},
		{name: "separated source and target", args: []string{"33", "millimeters", "inches"}, wantValue: 33, wantSource: "mm", wantTarget: "in"},
		{name: "negative temperature", args: []string{"-40f", "c"}, wantValue: -40, wantSource: "°F", wantTarget: "°C"},
		{name: "decimal without leading zero", args: []string{".5", "meters", "cm"}, wantValue: 0.5, wantSource: "m", wantTarget: "cm"},
		{name: "scientific notation", args: []string{"1e3mm", "m"}, wantValue: 1000, wantSource: "mm", wantTarget: "m"},
		{name: "degree symbol alias", args: []string{"20°C", "fahrenheit"}, wantValue: 20, wantSource: "°C", wantTarget: "°F"},
		{name: "missing input", args: nil, wantErr: "missing measurement"},
		{name: "numeric prefixed source", args: []string{"1", "2m", "cm"}, wantErr: "unknown unit"},
		{name: "nonfinite source", args: []string{"NaN", "m"}, wantErr: "invalid measurement"},
		{name: "infinite source", args: []string{"Inf", "m"}, wantErr: "invalid measurement"},
		{name: "missing unit", args: []string{"33"}, wantErr: "missing unit"},
		{name: "unknown unit", args: []string{"12parsecs"}, wantErr: "unknown unit"},
		{name: "malformed value", args: []string{"manymm"}, wantErr: "invalid measurement"},
		{name: "too many arguments", args: []string{"33", "mm", "in", "extra"}, wantErr: "too many arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			measurement, target, err := Parse(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse(%q) error = %v, want containing %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.args, err)
			}
			if measurement.Value != tt.wantValue {
				t.Errorf("Parse(%q) value = %v, want %v", tt.args, measurement.Value, tt.wantValue)
			}
			if measurement.Unit.Symbol != tt.wantSource {
				t.Errorf("Parse(%q) source = %q, want %q", tt.args, measurement.Unit.Symbol, tt.wantSource)
			}
			if tt.wantTarget == "" {
				if target != nil {
					t.Errorf("Parse(%q) target = %q, want nil", tt.args, target.Symbol)
				}
				return
			}
			if target == nil || target.Symbol != tt.wantTarget {
				t.Fatalf("Parse(%q) target = %#v, want %q", tt.args, target, tt.wantTarget)
			}
		})
	}
}

func TestConvertLength(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		from  string
		to    string
		want  float64
	}{
		{name: "millimeters to inches", value: 33, from: "mm", to: "in", want: 1.2992125984251968},
		{name: "miles to kilometers", value: 1, from: "mi", to: "km", want: 1.609344},
		{name: "yards to meters", value: 3, from: "yd", to: "m", want: 2.7432},
		{name: "feet to centimeters", value: 6, from: "ft", to: "cm", want: 182.88},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, ok := LookupUnit(tt.from)
			if !ok {
				t.Fatalf("LookupUnit(%q) failed", tt.from)
			}
			to, ok := LookupUnit(tt.to)
			if !ok {
				t.Fatalf("LookupUnit(%q) failed", tt.to)
			}
			got, err := Convert(Measurement{Value: tt.value, Unit: from}, to)
			if err != nil {
				t.Fatalf("Convert() unexpected error: %v", err)
			}
			if !nearlyEqual(got, tt.want) {
				t.Errorf("Convert() = %.12g, want %.12g", got, tt.want)
			}
		})
	}
}

func TestConvertTemperature(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		from  string
		to    string
		want  float64
	}{
		{name: "same at negative forty", value: -40, from: "f", to: "c", want: -40},
		{name: "freezing to kelvin", value: 32, from: "f", to: "k", want: 273.15},
		{name: "absolute zero", value: 0, from: "k", to: "c", want: -273.15},
		{name: "absolute zero celsius", value: -273.15, from: "c", to: "k", want: 0},
		{name: "absolute zero fahrenheit", value: -459.67, from: "f", to: "k", want: 0},
		{name: "room temperature", value: 20, from: "c", to: "f", want: 68},
		{name: "extreme fahrenheit", value: 1e308, from: "f", to: "c", want: 1e308 / 9 * 5},
		{name: "extreme kelvin", value: 9e307, from: "k", to: "f", want: 1.62e308},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := LookupUnit(tt.from)
			to, _ := LookupUnit(tt.to)
			got, err := Convert(Measurement{Value: tt.value, Unit: from}, to)
			if err != nil {
				t.Fatalf("Convert() unexpected error: %v", err)
			}
			if !nearlyEqual(got, tt.want) {
				t.Errorf("Convert() = %.12g, want %.12g", got, tt.want)
			}
		})
	}
}

func TestConvertRejectsIncompatibleDimensions(t *testing.T) {
	mm, _ := LookupUnit("mm")
	celsius, _ := LookupUnit("c")

	_, err := Convert(Measurement{Value: 33, Unit: mm}, celsius)
	if err == nil || !strings.Contains(err.Error(), "cannot convert length to temperature") {
		t.Fatalf("Convert() error = %v, want dimensional mismatch", err)
	}
}

func TestConvertRejectsLengthOverflow(t *testing.T) {
	kilometers, _ := LookupUnit("km")
	meters, _ := LookupUnit("m")

	_, err := Convert(Measurement{Value: 1e308, Unit: kilometers}, meters)
	if err == nil || !strings.Contains(err.Error(), "outside supported range") {
		t.Fatalf("Convert() error = %v, want supported-range error", err)
	}
}

func TestConvertRejectsLengthUnderflow(t *testing.T) {
	millimeters, _ := LookupUnit("mm")
	kilometers, _ := LookupUnit("km")

	_, err := Convert(Measurement{Value: math.SmallestNonzeroFloat64, Unit: millimeters}, kilometers)
	if err == nil || !strings.Contains(err.Error(), "outside supported range") {
		t.Fatalf("Convert() error = %v, want supported-range error", err)
	}
}

func TestConvertRejectsTemperatureBelowAbsoluteZero(t *testing.T) {
	fahrenheit, _ := LookupUnit("f")
	tests := []struct {
		name  string
		value float64
		unit  string
	}{
		{name: "kelvin", value: -0.0000000005, unit: "k"},
		{name: "celsius", value: -273.1500000005, unit: "c"},
		{name: "fahrenheit", value: -459.6700000005, unit: "f"},
		{name: "next float below fahrenheit zero", value: math.Nextafter(-459.67, math.Inf(-1)), unit: "f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, _ := LookupUnit(tt.unit)
			_, err := Convert(Measurement{Value: tt.value, Unit: source}, fahrenheit)
			if err == nil || !strings.Contains(err.Error(), "below absolute zero") {
				t.Fatalf("Convert() error = %v, want absolute-zero error", err)
			}
		})
	}
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}
