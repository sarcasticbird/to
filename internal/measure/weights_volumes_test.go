package measure

import (
	"math"
	"testing"
)

func TestWeightVolumeDefinitions(t *testing.T) {
	for _, tt := range []struct {
		from, to    string
		value, want float64
	}{
		{"lb", "g", 1, 453.59237}, {"oz", "g", 1, 28.349523125},
		{"st", "lb", 1, 14}, {"kg", "mg", 1, 1e6},
		{"gal", "l", 1, 3.785411784}, {"gal", "qt", 1, 4},
		{"qt", "pt", 1, 2}, {"pt", "cup", 1, 2},
		{"cup", "floz", 1, 8}, {"floz", "tbsp", 1, 2},
		{"tbsp", "tsp", 1, 3}, {"l", "ml", 1, 1000},
		{"mg", "kg", 0, 0}, {"l", "ml", -2, -2000},
		{"kg", "lb", 1e308, 0}, // actual result overflows
	} {
		from, _ := LookupUnit(tt.from)
		to, _ := LookupUnit(tt.to)
		got, err := Convert(Measurement{Value: tt.value, Unit: from}, to)
		if tt.value == 1e308 {
			if err == nil {
				t.Error("expected overflow error")
			}
			continue
		}
		if err != nil || math.Abs(got-tt.want) > 1e-12*math.Max(1, math.Abs(tt.want)) {
			t.Errorf("%g %s to %s = %g, %v; want %g", tt.value, tt.from, tt.to, got, err, tt.want)
		}
	}
}

func TestWeightVolumeRange(t *testing.T) {
	for _, pair := range [][2]string{{"kg", "st"}, {"l", "gal"}} {
		from, _ := LookupUnit(pair[0])
		to, _ := LookupUnit(pair[1])
		got, err := Convert(Measurement{Value: 1e308, Unit: from}, to)
		if err != nil || !isFinite(got) || got <= 0 {
			t.Errorf("representable large conversion %v = %g, %v", pair, got, err)
		}
	}
	for _, pair := range [][2]string{{"mg", "kg"}, {"ml", "gal"}} {
		from, _ := LookupUnit(pair[0])
		to, _ := LookupUnit(pair[1])
		if _, err := Convert(Measurement{Value: math.SmallestNonzeroFloat64, Unit: from}, to); err == nil {
			t.Errorf("expected underflow for %v", pair)
		}
	}
}

func TestWeightVolumeAliases(t *testing.T) {
	for alias, symbol := range map[string]string{
		"milligrams": "mg", "grams": "g", "kilograms": "kg", "ounces": "oz", "lbs": "lb", "pounds": "lb", "stones": "st",
		"millilitres": "ml", "LITERS": "l", "teaspoons": "tsp", "tablespoons": "tbsp", "fl oz": "floz", "fluid ounces": "floz", "cups": "cup", "pints": "pt", "quarts": "qt", "gallons": "gal",
	} {
		unit, ok := LookupUnit(alias)
		if !ok || unit.Symbol != symbol {
			t.Errorf("LookupUnit(%q) = %v, %v", alias, unit, ok)
		}
	}
	for _, alias := range []string{"imperial gallon", "dry pint", "troy ounce"} {
		if _, ok := LookupUnit(alias); ok {
			t.Errorf("unsupported definition accepted: %s", alias)
		}
	}
}
