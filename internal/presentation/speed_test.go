package presentation

import (
	"errors"
	"math"
	"testing"

	"github.com/sarcasticbird/to/internal/measure"
)

func TestSpeedSummary(t *testing.T) {
	for _, tt := range []struct {
		source string
		value  float64
		want   string
	}{
		{"m/s", 1, "1 m/s\n  3.6 km/h\n  2.237 mph\n  1.944 kn"},
		{"km/h", 1, "1 km/h\n  0.2778 m/s\n  0.6214 mph\n  0.54 kn"},
		{"mph", 1, "1 mph\n  0.447 m/s\n  1.609 km/h\n  0.869 kn"},
		{"ft/s", 1, "1 ft/s\n  0.3048 m/s\n  1.097 km/h\n  0.6818 mph\n  0.5925 kn"},
		{"kn", 1, "1 kn\n  0.5144 m/s\n  1.852 km/h\n  1.151 mph"},
		{"m/s", -1, "-1 m/s\n  -3.6 km/h\n  -2.237 mph\n  -1.944 kn"},
		{"ft/s", 0, "0 ft/s\n  0 m/s\n  0 km/h\n  0 mph\n  0 kn"},
		{"m/s", 6e307, "6e+307 m/s\n  1.342e+308 mph\n  1.166e+308 kn"},
		{"m/s", -6e307, "-6e+307 m/s\n  -1.342e+308 mph\n  -1.166e+308 kn"},
		{"m/s", math.MaxFloat64, "1.798e+308 m/s"},
		{"km/h", math.SmallestNonzeroFloat64, "4.941e-324 km/h\n  4.941e-324 mph\n  4.941e-324 kn"},
		{"ft/s", -math.SmallestNonzeroFloat64, "-4.941e-324 ft/s\n  -4.941e-324 km/h\n  -4.941e-324 mph\n  -4.941e-324 kn"},
	} {
		t.Run(tt.source+"/"+formatNumber(tt.value), func(t *testing.T) {
			got, err := Summary(measure.Measurement{Value: tt.value, Unit: unit(t, tt.source)})
			if err != nil || got != tt.want {
				t.Errorf("Summary() = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestSpeedDirect(t *testing.T) {
	for _, tt := range []struct {
		from, to string
		value    float64
		want     string
	}{
		{"mph", "km/h", 60, "96.56 km/h"},
		{"m/s", "ft/s", 1, "3.281 ft/s"},
		{"km/h", "m/s", -36, "-10 m/s"},
		{"kn", "mph", 0, "0 mph"},
		{"km/h", "mph", 1e308, "6.214e+307 mph"},
		{"km/h", "mph", math.SmallestNonzeroFloat64, "4.941e-324 mph"},
		{"ft/s", "ft/s", -math.SmallestNonzeroFloat64, "-4.941e-324 ft/s"},
	} {
		got, err := Direct(measure.Measurement{Value: tt.value, Unit: unit(t, tt.from)}, unit(t, tt.to))
		if err != nil || got != tt.want {
			t.Errorf("Direct(%g %s to %s) = %q, %v; want %q", tt.value, tt.from, tt.to, got, err, tt.want)
		}
	}
	for _, value := range []float64{1e308, -1e308, math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64} {
		from, to := "m/s", "km/h"
		if math.Abs(value) < 1 {
			from, to = to, from
		}
		got, err := Direct(measure.Measurement{Value: value, Unit: unit(t, from)}, unit(t, to))
		if got != "" || !errors.Is(err, measure.ErrOutOfRange) {
			t.Errorf("Direct(%g %s to %s) = %q, %v; want empty result and ErrOutOfRange", value, from, to, got, err)
		}
	}
}

func TestSpeedSummaryRejectsNonfiniteSource(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		got, err := Summary(measure.Measurement{Value: value, Unit: unit(t, "m/s")})
		if got != "" || !errors.Is(err, measure.ErrOutOfRange) {
			t.Errorf("Summary(%g m/s) = %q, %v; want empty result and ErrOutOfRange", value, got, err)
		}
	}
}
