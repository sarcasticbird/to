package measure

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Dimension identifies measurements that can be converted to one another.
type Dimension string

const (
	Length      Dimension = "length"
	Weight      Dimension = "weight"
	Volume      Dimension = "volume"
	Temperature Dimension = "temperature"
)

// Unit describes one supported unit of measure.
type Unit struct {
	Symbol    string
	Name      string
	Dimension Dimension
	factor    float64 // base units per unit for linear dimensions
	toBase    func(float64) float64
	fromBase  func(float64) float64
}

// Measurement is a numeric value paired with its source unit.
type Measurement struct {
	Value float64
	Unit  Unit
}

// ErrOutOfRange indicates that a conversion cannot be represented as float64.
var ErrOutOfRange = errors.New("conversion result is outside supported range")

var (
	errMissingUnit     = errors.New("missing unit")
	numberPattern      = regexp.MustCompile(`^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$`)
	measurementPattern = regexp.MustCompile(`^([+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?)\s*(.+)$`)
	aliases            = buildAliases()
)

// Parse interprets a command line containing a measurement and optional target.
func Parse(args []string) (Measurement, *Unit, error) {
	if len(args) == 0 {
		return Measurement{}, nil, errors.New("missing measurement")
	}
	if len(args) > 3 {
		return Measurement{}, nil, errors.New("too many arguments")
	}

	measurement, err := parseMeasurement(args[0])
	targetIndex := 1
	if errors.Is(err, errMissingUnit) && len(args) >= 2 {
		value, valueErr := parseNumber(strings.TrimSpace(args[0]))
		if valueErr != nil {
			return Measurement{}, nil, valueErr
		}
		unit, ok := LookupUnit(args[1])
		if !ok {
			return Measurement{}, nil, fmt.Errorf("unknown unit %q", args[1])
		}
		measurement, err = Measurement{Value: value, Unit: unit}, nil
		targetIndex = 2
	}
	if err != nil {
		return Measurement{}, nil, err
	}
	if len(args) > targetIndex+1 {
		return Measurement{}, nil, errors.New("too many arguments")
	}
	if len(args) == targetIndex {
		return measurement, nil, nil
	}

	target, ok := LookupUnit(args[targetIndex])
	if !ok {
		return Measurement{}, nil, fmt.Errorf("unknown unit %q", args[targetIndex])
	}
	return measurement, &target, nil
}

// LookupUnit resolves a conventional symbol or English unit name.
func LookupUnit(name string) (Unit, bool) {
	unit, ok := aliases[normalizeUnit(name)]
	return unit, ok
}

// Convert converts a measurement to a compatible target unit.
func Convert(measurement Measurement, target Unit) (float64, error) {
	if measurement.Unit.Dimension != target.Dimension {
		return 0, fmt.Errorf("cannot convert %s to %s", measurement.Unit.Dimension, target.Dimension)
	}
	if !isFinite(measurement.Value) {
		return 0, fmt.Errorf("convert %g %s to %s: %w", measurement.Value, measurement.Unit.Symbol, target.Symbol, ErrOutOfRange)
	}
	if measurement.Unit.Dimension == Length || measurement.Unit.Dimension == Weight || measurement.Unit.Dimension == Volume {
		result := measurement.Value * (measurement.Unit.factor / target.factor)
		if !isFinite(result) || (measurement.Value != 0 && result == 0) {
			return 0, fmt.Errorf("convert %g %s to %s: %w", measurement.Value, measurement.Unit.Symbol, target.Symbol, ErrOutOfRange)
		}
		return result, nil
	}

	base := measurement.Unit.toBase(measurement.Value)
	if !isFinite(base) {
		return 0, fmt.Errorf("convert %g %s to %s: %w", measurement.Value, measurement.Unit.Symbol, target.Symbol, ErrOutOfRange)
	}
	if measurement.Unit.Dimension == Temperature && base < 0 {
		return 0, fmt.Errorf("temperature %.12g %s is below absolute zero", measurement.Value, measurement.Unit.Symbol)
	}
	if measurement.Unit.Symbol == target.Symbol {
		return measurement.Value, nil
	}
	result := target.fromBase(base)
	// Avoid cancellation through Kelvin for direct Celsius/Fahrenheit requests.
	if measurement.Unit.Symbol == "°F" && target.Symbol == "°C" {
		result = (measurement.Value - 32) / 9 * 5
	} else if measurement.Unit.Symbol == "°C" && target.Symbol == "°F" {
		result = measurement.Value/5*9 + 32
	}
	if !isFinite(result) {
		return 0, fmt.Errorf("convert %g %s to %s: %w", measurement.Value, measurement.Unit.Symbol, target.Symbol, ErrOutOfRange)
	}
	return result, nil
}

func parseMeasurement(input string) (Measurement, error) {
	raw := strings.TrimSpace(input)
	if numberPattern.MatchString(raw) {
		return Measurement{}, errMissingUnit
	}

	parts := measurementPattern.FindStringSubmatch(raw)
	if parts == nil {
		return Measurement{}, fmt.Errorf("invalid measurement %q", input)
	}
	value, err := parseNumber(parts[1])
	if err != nil {
		return Measurement{}, fmt.Errorf("invalid measurement %q: %w", input, err)
	}
	unit, ok := LookupUnit(parts[2])
	if !ok {
		return Measurement{}, fmt.Errorf("unknown unit %q", strings.TrimSpace(parts[2]))
	}
	return Measurement{Value: value, Unit: unit}, nil
}

func parseNumber(raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid measurement %q: %w", raw, err)
	}
	if !isFinite(value) {
		return 0, fmt.Errorf("measurement %q is outside supported range", raw)
	}
	if value == 0 {
		// ParseFloat silently rounds sufficiently small nonzero inputs to zero.
		mantissa := strings.FieldsFunc(raw, func(r rune) bool { return r == 'e' || r == 'E' })[0]
		if strings.ContainsAny(mantissa, "123456789") {
			return 0, fmt.Errorf("measurement %q is outside supported range", raw)
		}
	}
	return value, nil
}

func buildAliases() map[string]Unit {
	// International avoirdupois pound in grams and US liquid gallon in liters.
	const pound = 453.59237
	const gallon = 3.785411784
	units := []struct {
		unit    Unit
		aliases []string
	}{
		{linearUnit("mm", "millimeter", Length, 0.001), []string{"millimeter", "millimeters", "millimetre", "millimetres"}},
		{linearUnit("cm", "centimeter", Length, 0.01), []string{"centimeter", "centimeters", "centimetre", "centimetres"}},
		{linearUnit("m", "meter", Length, 1), []string{"meter", "meters", "metre", "metres"}},
		{linearUnit("km", "kilometer", Length, 1000), []string{"kilometer", "kilometers", "kilometre", "kilometres"}},
		{linearUnit("in", "inch", Length, 0.0254), []string{"inch", "inches", `"`}},
		{linearUnit("ft", "foot", Length, 0.3048), []string{"foot", "feet", `'`}},
		{linearUnit("yd", "yard", Length, 0.9144), []string{"yard", "yards"}},
		{linearUnit("mi", "mile", Length, 1609.344), []string{"mile", "miles"}},
		{linearUnit("mg", "milligram", Weight, 0.001), []string{"milligram", "milligrams"}},
		{linearUnit("g", "gram", Weight, 1), []string{"gram", "grams"}},
		{linearUnit("kg", "kilogram", Weight, 1000), []string{"kilogram", "kilograms"}},
		{linearUnit("oz", "ounce", Weight, pound/16), []string{"ounce", "ounces"}},
		{linearUnit("lb", "pound", Weight, pound), []string{"lbs", "pound", "pounds"}},
		{linearUnit("st", "stone", Weight, pound*14), []string{"stone", "stones"}},
		{linearUnit("ml", "milliliter", Volume, 0.001), []string{"milliliter", "milliliters", "millilitre", "millilitres"}},
		{linearUnit("l", "liter", Volume, 1), []string{"liter", "liters", "litre", "litres"}},
		{linearUnit("tsp", "US teaspoon", Volume, gallon/768), []string{"teaspoon", "teaspoons"}},
		{linearUnit("tbsp", "US tablespoon", Volume, gallon/256), []string{"tablespoon", "tablespoons"}},
		{linearUnit("floz", "US fluid ounce", Volume, gallon/128), []string{"fl oz", "fluid ounce", "fluid ounces"}},
		{linearUnit("cup", "US cup", Volume, gallon/16), []string{"cups"}},
		{linearUnit("pt", "US liquid pint", Volume, gallon/8), []string{"pint", "pints"}},
		{linearUnit("qt", "US liquid quart", Volume, gallon/4), []string{"quart", "quarts"}},
		{linearUnit("gal", "US liquid gallon", Volume, gallon), []string{"gallon", "gallons"}},
		{temperatureUnit("°C", "Celsius", celsiusToKelvin, kelvinToCelsius), []string{"c", "celsius", "centigrade"}},
		{temperatureUnit("°F", "Fahrenheit", fahrenheitToKelvin, kelvinToFahrenheit), []string{"f", "fahrenheit"}},
		{temperatureUnit("K", "Kelvin", identity, identity), []string{"k", "kelvin", "kelvins"}},
	}

	result := make(map[string]Unit)
	for _, item := range units {
		result[normalizeUnit(item.unit.Symbol)] = item.unit
		for _, alias := range item.aliases {
			result[normalizeUnit(alias)] = item.unit
		}
	}
	return result
}

func linearUnit(symbol, name string, dimension Dimension, factor float64) Unit {
	return Unit{
		Symbol:    symbol,
		Name:      name,
		Dimension: dimension,
		factor:    factor,
		toBase:    func(value float64) float64 { return value * factor },
		fromBase:  func(value float64) float64 { return value / factor },
	}
}

func temperatureUnit(symbol, name string, toBase, fromBase func(float64) float64) Unit {
	return Unit{Symbol: symbol, Name: name, Dimension: Temperature, toBase: toBase, fromBase: fromBase}
}

func normalizeUnit(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "degrees ")
	name = strings.TrimPrefix(name, "degree ")
	name = strings.TrimSpace(strings.TrimPrefix(name, "°"))
	return name
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func identity(value float64) float64        { return value }
func celsiusToKelvin(value float64) float64 { return value + 273.15 }
func kelvinToCelsius(value float64) float64 { return value - 273.15 }
func fahrenheitToKelvin(value float64) float64 {
	if value > math.MaxFloat64/5 {
		return (value-32)/9*5 + 273.15
	}
	return (value-32)*5/9 + 273.15
}

func kelvinToFahrenheit(value float64) float64 {
	if value > math.MaxFloat64/9 {
		return (value-273.15)/5*9 + 32
	}
	return (value-273.15)*9/5 + 32
}
