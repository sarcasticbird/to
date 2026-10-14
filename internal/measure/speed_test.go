package measure

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestSpeedDefinitionsAndAliases(t *testing.T) {
	for _, tt := range []struct {
		symbol  string
		factor  float64
		aliases []string
	}{
		{"m/s", 1, []string{"meter per second", "metre per second", "meters per second", "metres per second", "mps"}},
		{"km/h", 1 / 3.6, []string{"kilometer per hour", "kilometre per hour", "kilometers per hour", "kilometres per hour", "kph", "kmph"}},
		{"mph", 0.44704, []string{"mile per hour", "miles per hour"}},
		{"ft/s", 0.3048, []string{"foot per second", "feet per second", "fps"}},
		{"kn", 1852.0 / 3600, []string{"knot", "knots", "kt"}},
	} {
		t.Run(tt.symbol, func(t *testing.T) {
			for _, alias := range append([]string{tt.symbol, strings.ToUpper(tt.symbol)}, tt.aliases...) {
				unit, ok := LookupUnit(alias)
				if !ok {
					t.Fatalf("LookupUnit(%q) failed", alias)
				}
				if unit.Symbol != tt.symbol || unit.Dimension != Speed || unit.factor != tt.factor {
					t.Errorf("LookupUnit(%q) = (%q, %q, %.17g), want (%q, %q, %.17g)", alias, unit.Symbol, unit.Dimension, unit.factor, tt.symbol, Speed, tt.factor)
				}
			}
		})
	}
}

func TestParseSpeed(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		value   float64
		source  string
		target  string
		wantErr string
	}{
		{name: "attached summary", args: []string{"10m/s"}, value: 10, source: "m/s"},
		{name: "separated summary", args: []string{"10", "knots"}, value: 10, source: "kn"},
		{name: "attached source and target", args: []string{"36km/h", "m/s"}, value: 36, source: "km/h", target: "m/s"},
		{name: "separated source and target", args: []string{"60", "mph", "km/h"}, value: 60, source: "mph", target: "km/h"},
		{name: "quoted source alias", args: []string{"10", "metres per second", "mph"}, value: 10, source: "m/s", target: "mph"},
		{name: "quoted target alias", args: []string{"10m/s", "feet per second"}, value: 10, source: "m/s", target: "ft/s"},
		{name: "quoted measurement and aliases", args: []string{"10 miles per hour", "kilometres per hour"}, value: 10, source: "mph", target: "km/h"},
		{name: "case insensitive", args: []string{"10FT/S", "KN"}, value: 10, source: "ft/s", target: "kn"},
		{name: "negative scientific notation", args: []string{"-1e2kph", "mps"}, value: -100, source: "km/h", target: "m/s"},
		{name: "fractional shorthand", args: []string{".5fps", "kt"}, value: 0.5, source: "ft/s", target: "kn"},
		{name: "unknown target", args: []string{"10m/s", "parsecs per hour"}, wantErr: "unknown unit"},
		{name: "unquoted multiword source", args: []string{"10", "meters", "per", "second"}, wantErr: "too many arguments"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, target, err := Parse(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Parse(%q) error = %v, want containing %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.args, err)
			}
			if got.Value != tt.value || got.Unit.Symbol != tt.source || got.Unit.Dimension != Speed {
				t.Errorf("Parse(%q) measurement = %#v, want %g %s (speed)", tt.args, got, tt.value, tt.source)
			}
			if tt.target == "" {
				if target != nil {
					t.Errorf("Parse(%q) target = %#v, want nil", tt.args, target)
				}
			} else if target == nil || target.Symbol != tt.target || target.Dimension != Speed {
				t.Errorf("Parse(%q) target = %#v, want %s (speed)", tt.args, target, tt.target)
			}
		})
	}
}

func TestConvertSpeed(t *testing.T) {
	for _, tt := range []struct {
		from, to    string
		value, want float64
	}{
		{"km/h", "m/s", 36, 10},
		{"mph", "km/h", 60, 96.56064},
		{"ft/s", "m/s", 10, 3.048},
		{"kn", "km/h", 10, 18.52},
		{"m/s", "mph", -10, -10 / 0.44704},
		{"m/s", "ft/s", 1, 1 / 0.3048},
	} {
		from, _ := LookupUnit(tt.from)
		to, _ := LookupUnit(tt.to)
		got, err := Convert(Measurement{Value: tt.value, Unit: from}, to)
		if err != nil || !nearlyEqual(got, tt.want) {
			t.Errorf("Convert(%g %s to %s) = %.17g, %v; want %.17g", tt.value, tt.from, tt.to, got, err, tt.want)
		}
	}
}

func TestSpeedDimensionMismatch(t *testing.T) {
	speed, _ := LookupUnit("m/s")
	for _, symbol := range []string{"m", "kg", "l", "c"} {
		other, _ := LookupUnit(symbol)
		for _, pair := range [][2]Unit{{speed, other}, {other, speed}} {
			_, err := Convert(Measurement{Value: 1, Unit: pair[0]}, pair[1])
			want := "cannot convert " + string(pair[0].Dimension) + " to " + string(pair[1].Dimension)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("Convert(%s to %s) error = %v, want %q", pair[0].Symbol, pair[1].Symbol, err, want)
			}
		}
	}
}

func TestSpeedRange(t *testing.T) {
	for _, tt := range []struct {
		name        string
		from, to    string
		value, want float64
		outOfRange  bool
	}{
		{"large representable", "km/h", "mph", 1e308, 1e308 * ((1.0 / 3.6) / 0.44704), false},
		{"negative large representable", "km/h", "m/s", -math.MaxFloat64, -math.MaxFloat64 * (1.0 / 3.6), false},
		{"small representable without base underflow", "km/h", "mph", math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64, false},
		{"negative small representable", "ft/s", "kn", -math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, false},
		{"overflow", "m/s", "km/h", 1e308, 0, true},
		{"negative overflow", "m/s", "mph", -1e308, 0, true},
		{"underflow", "km/h", "m/s", math.SmallestNonzeroFloat64, 0, true},
		{"negative underflow", "ft/s", "m/s", -math.SmallestNonzeroFloat64, 0, true},
		{"zero", "km/h", "m/s", 0, 0, false},
		{"nan", "mph", "kn", math.NaN(), 0, true},
		{"infinity", "mph", "kn", math.Inf(1), 0, true},
		{"negative infinity", "mph", "kn", math.Inf(-1), 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			from, _ := LookupUnit(tt.from)
			to, _ := LookupUnit(tt.to)
			got, err := Convert(Measurement{Value: tt.value, Unit: from}, to)
			if tt.outOfRange {
				if !errors.Is(err, ErrOutOfRange) {
					t.Fatalf("Convert() error = %v, want ErrOutOfRange", err)
				}
				return
			}
			if err != nil || !isFinite(got) || math.Abs(got-tt.want) > 1e-15*math.Abs(tt.want) {
				t.Errorf("Convert() = %.17g, %v; want %.17g", got, err, tt.want)
			}
		})
	}
}

func TestSpeedSelfConversionEdges(t *testing.T) {
	for _, symbol := range []string{"m/s", "km/h", "mph", "ft/s", "kn"} {
		source, _ := LookupUnit(symbol)
		for _, value := range []float64{math.MaxFloat64, -math.MaxFloat64, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, 0, math.Copysign(0, -1)} {
			got, err := Convert(Measurement{Value: value, Unit: source}, source)
			if err != nil || math.Float64bits(got) != math.Float64bits(value) {
				t.Errorf("Convert(%g %s to itself) = %g, %v; want unchanged bits", value, symbol, got, err)
			}
		}
	}
}
