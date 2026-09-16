package presentation

import (
	"math"
	"strings"
	"testing"

	"github.com/sarcasticbird/to/internal/measure"
)

func TestDirectFormatsAConciseConversion(t *testing.T) {
	mm := unit(t, "mm")
	inches := unit(t, "in")

	got, err := Direct(measure.Measurement{Value: 33, Unit: mm}, inches)
	if err != nil {
		t.Fatalf("Direct() unexpected error: %v", err)
	}
	if got != "1.299 in" {
		t.Errorf("Direct() = %q, want %q", got, "1.299 in")
	}
}

func TestDirectFormatsTinyFiniteValuesInScientificNotation(t *testing.T) {
	meters := unit(t, "m")
	millimeters := unit(t, "mm")

	got, err := Direct(measure.Measurement{Value: math.SmallestNonzeroFloat64, Unit: meters}, millimeters)
	if err != nil {
		t.Fatalf("Direct() unexpected error: %v", err)
	}
	if got != "4.941e-321 mm" {
		t.Errorf("Direct() = %q, want %q", got, "4.941e-321 mm")
	}
}

func TestDirectFormatsLargeFiniteValuesInScientificNotation(t *testing.T) {
	millimeters := unit(t, "mm")
	inches := unit(t, "in")

	got, err := Direct(measure.Measurement{Value: 1e308, Unit: millimeters}, inches)
	if err != nil {
		t.Fatalf("Direct() unexpected error: %v", err)
	}
	if got != "3.937e+306 in" {
		t.Errorf("Direct() = %q, want %q", got, "3.937e+306 in")
	}
}

func TestSummaryUsesSmallLengthProfile(t *testing.T) {
	mm := unit(t, "mm")

	got, err := Summary(measure.Measurement{Value: 33, Unit: mm})
	if err != nil {
		t.Fatalf("Summary() unexpected error: %v", err)
	}
	want := strings.Join([]string{
		"33 mm",
		"  3.3 cm",
		"  1.299 in",
		"  ≈ 1 19/64 in",
	}, "\n")
	if got != want {
		t.Errorf("Summary() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSummaryUsesHumanScaleLengthProfile(t *testing.T) {
	meters := unit(t, "m")

	got, err := Summary(measure.Measurement{Value: 1.8, Unit: meters})
	if err != nil {
		t.Fatalf("Summary() unexpected error: %v", err)
	}
	want := strings.Join([]string{
		"1.8 m",
		"  ≈ 5 ft 10 55/64 in",
		"  1.969 yd",
	}, "\n")
	if got != want {
		t.Errorf("Summary() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSummaryCarriesRoundedInchesIntoFeet(t *testing.T) {
	meters := unit(t, "m")

	got, err := Summary(measure.Measurement{Value: 1.82879, Unit: meters})
	if err != nil {
		t.Fatalf("Summary() unexpected error: %v", err)
	}
	if !strings.Contains(got, "  ≈ 6 ft 0 in") {
		t.Errorf("Summary() =\n%s\nwant carried feet and inches", got)
	}
}

func TestSummaryDoesNotMarkExactFeetAndInchesApproximate(t *testing.T) {
	meters := unit(t, "m")

	got, err := Summary(measure.Measurement{Value: 1.8288, Unit: meters})
	if err != nil {
		t.Fatalf("Summary() unexpected error: %v", err)
	}
	if !strings.Contains(got, "  6 ft 0 in") || strings.Contains(got, "≈ 6 ft 0 in") {
		t.Errorf("Summary() =\n%s\nwant exact feet and inches without approximation marker", got)
	}
}

func TestSummaryUsesDistanceProfile(t *testing.T) {
	miles := unit(t, "mi")

	got, err := Summary(measure.Measurement{Value: 26.2, Unit: miles})
	if err != nil {
		t.Fatalf("Summary() unexpected error: %v", err)
	}
	want := strings.Join([]string{
		"26.2 mi",
		"  42.16 km",
	}, "\n")
	if got != want {
		t.Errorf("Summary() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSummaryShowsTemperatureEquivalents(t *testing.T) {
	fahrenheit := unit(t, "f")

	got, err := Summary(measure.Measurement{Value: 72, Unit: fahrenheit})
	if err != nil {
		t.Fatalf("Summary() unexpected error: %v", err)
	}
	want := strings.Join([]string{
		"72 °F",
		"  22.22 °C",
		"  295.4 K",
	}, "\n")
	if got != want {
		t.Errorf("Summary() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSummaryRejectsTemperatureBelowAbsoluteZero(t *testing.T) {
	celsius := unit(t, "c")

	_, err := Summary(measure.Measurement{Value: -274, Unit: celsius})
	if err == nil || !strings.Contains(err.Error(), "below absolute zero") {
		t.Fatalf("Summary() error = %v, want absolute-zero error", err)
	}
}

func unit(t *testing.T, name string) measure.Unit {
	t.Helper()
	got, ok := measure.LookupUnit(name)
	if !ok {
		t.Fatalf("measure.LookupUnit(%q) failed", name)
	}
	return got
}

func TestNumberRounding(t *testing.T) {
	for _, tt := range []struct {
		value float64
		want  string
	}{
		{1234567, "1235000"}, {-1234567, "-1235000"},
		{9999.6, "10000"}, {0.0001, "0.0001"}, {999.96, "1000"},
	} {
		if got := formatNumber(tt.value); got != tt.want {
			t.Errorf("formatNumber(%g) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestFractionAccuracy(t *testing.T) {
	for _, value := range []float64{1e-9, -1e-9, math.SmallestNonzeroFloat64} {
		got, exact := formatFractionalInches(value)
		if got != "0 in" || exact {
			t.Errorf("fraction(%g) = %q, exact %v", value, got, exact)
		}
	}
	if _, exact := formatFeetAndInches(72 + 1e-10); exact {
		t.Error("nearby feet measurement should be approximate")
	}
}
