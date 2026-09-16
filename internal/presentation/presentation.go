package presentation

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sarcasticbird/to/internal/measure"
)

const significantDigits = 4

// Direct renders one requested conversion.
func Direct(measurement measure.Measurement, target measure.Unit) (string, error) {
	value, err := measure.Convert(measurement, target)
	if err != nil {
		return "", err
	}
	return formatNumber(value) + " " + target.Symbol, nil
}

// Summary renders a short set of useful equivalents for a measurement.
func Summary(measurement measure.Measurement) (string, error) {
	if _, err := measure.Convert(measurement, measurement.Unit); err != nil {
		return "", err
	}

	lines := []string{formatNumber(measurement.Value) + " " + measurement.Unit.Symbol}
	switch measurement.Unit.Dimension {
	case measure.Length:
		lengthLines, err := lengthSummary(measurement)
		if err != nil {
			return "", err
		}
		lines = append(lines, lengthLines...)
	case measure.Temperature:
		for _, symbol := range []string{"c", "f", "k"} {
			target := mustUnit(symbol)
			if target.Symbol == measurement.Unit.Symbol {
				continue
			}
			line, err := Direct(measurement, target)
			if err != nil {
				return "", err
			}
			lines = append(lines, "  "+line)
		}
	default:
		return "", fmt.Errorf("unsupported dimension %q", measurement.Unit.Dimension)
	}
	return strings.Join(lines, "\n"), nil
}

func lengthSummary(measurement measure.Measurement) ([]string, error) {
	// Compare in the source unit so profile selection cannot overflow or
	// underflow through an intermediate conversion of the user's value.
	oneMeter, err := measure.Convert(measure.Measurement{Value: 1, Unit: mustUnit("m")}, measurement.Unit)
	if err != nil {
		return nil, err
	}
	oneKilometer, err := measure.Convert(measure.Measurement{Value: 1000, Unit: mustUnit("m")}, measurement.Unit)
	if err != nil {
		return nil, err
	}

	switch absolute := math.Abs(measurement.Value); {
	case absolute < oneMeter:
		lines, err := convertedLines(measurement, []string{"mm", "cm", "in"})
		if err != nil {
			return nil, err
		}
		inches, err := measure.Convert(measurement, mustUnit("in"))
		if err != nil {
			return nil, err
		}
		fraction, exact := formatFractionalInches(inches)
		decimalInches := formatNumber(inches) + " in"
		if fraction != decimalInches {
			if !exact {
				fraction = "≈ " + fraction
			}
			lines = append(lines, "  "+fraction)
		}
		return lines, nil
	case absolute < oneKilometer:
		lines, err := convertedLines(measurement, []string{"m"})
		if err != nil {
			return nil, err
		}
		inches, err := measure.Convert(measurement, mustUnit("in"))
		if err != nil {
			return nil, err
		}
		feetAndInches, exact := formatFeetAndInches(inches)
		if !exact {
			feetAndInches = "≈ " + feetAndInches
		}
		lines = append(lines, "  "+feetAndInches)
		yards, err := convertedLines(measurement, []string{"yd"})
		if err != nil {
			return nil, err
		}
		return append(lines, yards...), nil
	default:
		return convertedLines(measurement, []string{"km", "mi"})
	}
}

func convertedLines(measurement measure.Measurement, symbols []string) ([]string, error) {
	lines := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		target := mustUnit(symbol)
		if target.Symbol == measurement.Unit.Symbol {
			continue
		}
		line, err := Direct(measurement, target)
		if err != nil {
			return nil, err
		}
		lines = append(lines, "  "+line)
	}
	return lines, nil
}

func formatNumber(value float64) string {
	if value == 0 {
		return "0"
	}

	abs := math.Abs(value)
	if abs < 1e-4 || abs >= 1e12 {
		return strconv.FormatFloat(value, 'g', significantDigits, 64)
	}
	decimalPlaces := significantDigits - 1 - int(math.Floor(math.Log10(abs)))
	factor := math.Pow10(decimalPlaces)
	rounded := math.Round(value*factor) / factor
	if rounded == 0 {
		return "0"
	}
	if decimalPlaces < 0 {
		decimalPlaces = 0
	}
	formatted := strconv.FormatFloat(rounded, 'f', decimalPlaces, 64)
	if strings.Contains(formatted, ".") {
		formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	}
	return formatted
}

func formatFractionalInches(inches float64) (string, bool) {
	sign := ""
	if inches < 0 {
		sign = "-"
	}
	abs := math.Abs(inches)
	totalSixtyFourths := int64(math.Round(abs * 64))
	whole := totalSixtyFourths / 64
	numerator := totalSixtyFourths % 64
	exact := fractionExact(abs, float64(totalSixtyFourths)/64)
	if totalSixtyFourths == 0 {
		sign = ""
	}

	var value string
	switch {
	case numerator == 0:
		value = strconv.FormatInt(whole, 10)
	case whole == 0:
		denominator := int64(64)
		divisor := greatestCommonDivisor(numerator, denominator)
		value = fmt.Sprintf("%d/%d", numerator/divisor, denominator/divisor)
	default:
		denominator := int64(64)
		divisor := greatestCommonDivisor(numerator, denominator)
		value = fmt.Sprintf("%d %d/%d", whole, numerator/divisor, denominator/divisor)
	}
	return sign + value + " in", exact
}

func formatFeetAndInches(inches float64) (string, bool) {
	sign := ""
	if inches < 0 {
		sign = "-"
	}
	totalSixtyFourths := int64(math.Round(math.Abs(inches) * 64))
	feet := totalSixtyFourths / (12 * 64)
	remaining := totalSixtyFourths % (12 * 64)
	wholeInches := remaining / 64
	numerator := remaining % 64

	inchValue := strconv.FormatInt(wholeInches, 10)
	if numerator != 0 {
		denominator := int64(64)
		divisor := greatestCommonDivisor(numerator, denominator)
		inchValue += fmt.Sprintf(" %d/%d", numerator/divisor, denominator/divisor)
	}
	exact := fractionExact(math.Abs(inches), float64(totalSixtyFourths)/64)
	return fmt.Sprintf("%s%d ft %s in", sign, feet, inchValue), exact
}

// Allow floating-point roundoff, but never equate a nonzero length with zero.
// These renderers only receive lengths below 1000 meters.
func fractionExact(value, rounded float64) bool {
	if rounded == 0 {
		return value == 0
	}
	ulp := math.Nextafter(rounded, math.Inf(1)) - rounded
	return math.Abs(value-rounded) <= 4*ulp
}

func greatestCommonDivisor(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func mustUnit(name string) measure.Unit {
	unit, ok := measure.LookupUnit(name)
	if !ok {
		panic("presentation references unknown unit " + name)
	}
	return unit
}
