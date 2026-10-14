package cli

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/sarcasticbird/to/internal/measure"
	"github.com/sarcasticbird/to/internal/presentation"
)

// Version is replaced at build time for releases.
var Version = "dev"

func version(moduleVersion string) string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if moduleVersion != "" && moduleVersion != "(devel)" {
		return moduleVersion
	}
	return "dev"
}

const help = `Usage:
  to <measurement> [target-unit]
  to <value> <source-unit> [target-unit]
  to <expression>, <expression> [...]

Without a target unit, to shows a short list of useful equivalents.
Separate multiple complete expressions with commas.

Length:      mm, cm, m, km, in, ft, yd, mi
Weight:      mg, g, kg, oz, lb, st
Volume:      ml, l, tsp, tbsp, floz, cup, pt, qt, gal
Speed:       m/s, km/h, mph, ft/s, kn
Temperature: c, f, k

Customary volumes use US definitions. oz is weight; floz is volume.

Speed aliases (quote multiword names; singular and plural accepted):
  m/s:  meter/metre per second, meters/metres per second, mps
  km/h: kilometer/kilometre per hour, kilometers/kilometres per hour, kph, kmph
  mph:  mile/miles per hour
  ft/s: foot/feet per second, fps
  kn:   knot/knots, kt
Exact definitions: 1 km/h = 1/3.6 m/s; 1 mph = 0.44704 m/s;
  1 ft/s = 0.3048 m/s; 1 kn = 1852/3600 m/s.
Speed summaries show m/s, km/h, mph, kn, omitting the source unit and
  out-of-range equivalents. ft/s is available as an explicit target.

Examples:
  to 33mm
  to 33mm in
  to 33mm in, 120f
  to 72f
  to 72f c
  to 85kg lb
  to 500ml cup
  to 85kg lb, 500ml cup
  to 100km/h mph
  to 100 km/h mph
  to 10 "metres per second" "miles per hour"
  to 60mph
  to 1m/s ft/s
  to 100km/h mph, 85kg lb`

// Run executes the CLI without terminating the calling process.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			return writeOutput(stdout, stderr, help)
		case "-v", "--version":
			moduleVersion := ""
			if info, ok := debug.ReadBuildInfo(); ok {
				moduleVersion = info.Main.Version
			}
			return writeOutput(stdout, stderr, "to "+version(moduleVersion))
		}
	}

	var (
		output string
		err    error
	)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, ",") {
		output, err = renderBatch(args)
	} else {
		output, err = renderExpression(args)
	}
	if err != nil {
		return reportError(stderr, err)
	}
	return writeOutput(stdout, stderr, output)
}

func writeOutput(stdout, stderr io.Writer, output string) int {
	if _, err := fmt.Fprintln(stdout, output); err != nil {
		reportError(stderr, fmt.Errorf("write output: %w", err))
		return 1
	}
	return 0
}

func renderBatch(args []string) (string, error) {
	// Split only on commas, retaining shell argument boundaries (including
	// quoted aliases such as "degrees celsius").
	expressions := [][]string{nil}
	for _, arg := range args {
		for index, part := range strings.Split(arg, ",") {
			if index > 0 {
				expressions = append(expressions, nil)
			}
			if part = strings.TrimSpace(part); part != "" {
				last := len(expressions) - 1
				expressions[last] = append(expressions[last], part)
			}
		}
	}
	outputs := make([]string, 0, len(expressions))
	for index, expression := range expressions {
		if len(expression) == 0 {
			return "", fmt.Errorf("measurement %d is empty", index+1)
		}
		output, err := renderExpression(expression)
		if err != nil {
			return "", fmt.Errorf("measurement %d: %w", index+1, err)
		}
		outputs = append(outputs, output)
	}
	return strings.Join(outputs, "\n\n"), nil
}

func renderExpression(args []string) (string, error) {
	measurement, target, err := measure.Parse(args)
	if err != nil {
		return "", err
	}
	if target == nil {
		return presentation.Summary(measurement)
	}
	return presentation.Direct(measurement, *target)
}

func reportError(stderr io.Writer, err error) int {
	if _, writeErr := fmt.Fprintf(stderr, "to: %v\n", err); writeErr != nil {
		return 1
	}
	return 2
}
