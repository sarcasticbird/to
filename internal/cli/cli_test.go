package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunDirectConversion(t *testing.T) {
	code, stdout, stderr := run("33mm", "in")
	if code != 0 {
		t.Errorf("Run() code = %d, want 0", code)
	}
	if stdout != "1.299 in\n" {
		t.Errorf("Run() stdout = %q, want %q", stdout, "1.299 in\n")
	}
	if stderr != "" {
		t.Errorf("Run() stderr = %q, want empty", stderr)
	}
}

func TestRunSeparatedMeasurement(t *testing.T) {
	code, stdout, stderr := run("33", "mm", "in")
	if code != 0 || stdout != "1.299 in\n" || stderr != "" {
		t.Fatalf("Run() = (%d, %q, %q), want (0, %q, empty)", code, stdout, stderr, "1.299 in\n")
	}
}

func TestRunSummary(t *testing.T) {
	code, stdout, stderr := run("72f")
	if code != 0 {
		t.Errorf("Run() code = %d, want 0", code)
	}
	for _, want := range []string{"72 °F", "22.22 °C", "295.4 K"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("Run() stdout = %q, want containing %q", stdout, want)
		}
	}
	if stderr != "" {
		t.Errorf("Run() stderr = %q, want empty", stderr)
	}
}

func TestRunCommaSeparatedSummaries(t *testing.T) {
	code, stdout, stderr := run("44mm,", "100mm,", "120cm")
	if code != 0 {
		t.Errorf("Run() code = %d, want 0", code)
	}
	if stderr != "" {
		t.Errorf("Run() stderr = %q, want empty", stderr)
	}
	if strings.Count(stdout, "\n\n") != 2 {
		t.Errorf("Run() stdout = %q, want three blocks separated by blank lines", stdout)
	}
	positions := []int{
		strings.Index(stdout, "44 mm\n"),
		strings.Index(stdout, "100 mm\n"),
		strings.Index(stdout, "120 cm\n"),
	}
	if positions[0] < 0 || positions[1] <= positions[0] || positions[2] <= positions[1] {
		t.Errorf("Run() stdout headings are missing or out of order: %q", stdout)
	}
}

func TestRunCommaSeparatedExpressionsCanHaveTargets(t *testing.T) {
	code, stdout, stderr := run("33", "mm", "in,", "120", "f", "c")
	want := "1.299 in\n\n48.89 °C\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Fatalf("Run() = (%d, %q, %q), want (0, %q, empty)", code, stdout, stderr, want)
	}
}

func TestRunValidatesEntireBatchBeforeWritingOutput(t *testing.T) {
	code, stdout, stderr := run("33mm,", "nonsense,", "72f")
	if code != 2 {
		t.Errorf("Run() code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("Run() stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "measurement 2") || !strings.Contains(stderr, "invalid measurement") {
		t.Errorf("Run() stderr = %q, want indexed invalid-measurement error", stderr)
	}
}

func TestRunRejectsEmptyBatchItems(t *testing.T) {
	code, stdout, stderr := run("33mm,")
	if code != 2 || stdout != "" {
		t.Fatalf("Run() = (%d, %q, %q), want code 2 and empty stdout", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "measurement 2 is empty") {
		t.Errorf("Run() stderr = %q, want empty-item error", stderr)
	}
}

func TestRunHelp(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		t.Run(flag, func(t *testing.T) {
			code, stdout, stderr := run(flag)
			if code != 0 {
				t.Errorf("Run() code = %d, want 0", code)
			}
			for _, want := range []string{"Usage:", "to <measurement> [target-unit]", "to 33mm", "to 33mm in, 120f", "Length:", "Temperature:"} {
				if !strings.Contains(stdout, want) {
					t.Errorf("Run() stdout missing %q:\n%s", want, stdout)
				}
			}
			if stderr != "" {
				t.Errorf("Run() stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, stderr := run("--version")
	if code != 0 || stdout != "to dev\n" || stderr != "" {
		t.Fatalf("Run() = (%d, %q, %q), want (0, %q, empty)", code, stdout, stderr, "to dev\n")
	}
}

func TestRunReportsUsageErrorsOnStderr(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing measurement", want: "missing measurement"},
		{name: "unknown unit", args: []string{"33blargh"}, want: "unknown unit"},
		{name: "incompatible units", args: []string{"33mm", "c"}, want: "cannot convert length to temperature"},
		{name: "below absolute zero", args: []string{"-1k"}, want: "below absolute zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := run(tt.args...)
			if code != 2 {
				t.Errorf("Run() code = %d, want 2", code)
			}
			if stdout != "" {
				t.Errorf("Run() stdout = %q, want empty", stdout)
			}
			if !strings.HasPrefix(stderr, "to: ") || !strings.Contains(stderr, tt.want) {
				t.Errorf("Run() stderr = %q, want a to error containing %q", stderr, tt.want)
			}
		})
	}
}

func run(args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestMalformedSourceBatchIsAtomic(t *testing.T) {
	code, stdout, stderr := run("33mm,", "1", "2m", "cm")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "measurement 2: unknown unit") {
		t.Fatalf("Run = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestTemperatureLandmarks(t *testing.T) {
	code, stdout, stderr := run("32f", "c,", "212f", "c,", "-459.67f", "k")
	if code != 0 || stdout != "0 °C\n\n100 °C\n\n0 K\n" || stderr != "" {
		t.Fatalf("Run = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestSameUnitPreservesValues(t *testing.T) {
	code, stdout, stderr := run("1e-14c", "c,", "1e308km", "km,", "5e-324mm", "mm")
	if code != 0 || stdout != "1e-14 °C\n\n1e+308 km\n\n4.941e-324 mm\n" || stderr != "" {
		t.Fatalf("Run = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestLengthConversionAvoidsIntermediateRangeErrors(t *testing.T) {
	code, stdout, stderr := run("1e308km", "mi,", "1e-321mm", "cm")
	if code != 0 || stdout != "6.214e+307 mi\n\n9.881e-323 cm\n" || stderr != "" {
		t.Fatalf("Run = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestLengthSummaryAvoidsIntermediateRangeErrors(t *testing.T) {
	code, stdout, stderr := run("1e308km,", "1e-321mm")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "6.214e+307 mi") || !strings.Contains(stdout, "9.881e-323 cm") {
		t.Fatalf("Run = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestBatchPreservesQuotedUnits(t *testing.T) {
	code, stdout, stderr := run("20", "degrees celsius", "f,", "1m", "cm")
	if code != 0 || stdout != "68 °F\n\n100 cm\n" || stderr != "" {
		t.Fatalf("Run = (%d, %q, %q)", code, stdout, stderr)
	}
}

func TestRejectsInputUnderflow(t *testing.T) {
	for _, args := range [][]string{
		{"1e-999m", "cm"}, {"1e-999", "m", "cm"},
		{"-1e-999k"}, {"33mm,", "-1e-999", "k"},
	} {
		code, stdout, stderr := run(args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "outside supported range") {
			t.Errorf("Run(%q) = (%d, %q, %q)", args, code, stdout, stderr)
		}
	}
	code, stdout, stderr := run("0e-999m", "cm")
	if code != 0 || stdout != "0 cm\n" || stderr != "" {
		t.Fatalf("zero input = (%d, %q, %q)", code, stdout, stderr)
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

func TestOutputFailures(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}, {"33mm"}, {"33mm,", "72f"}} {
		var stderr bytes.Buffer
		if code := Run(args, failedWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "write output: disk full") {
			t.Errorf("Run(%q) = %d, %q", args, code, stderr.String())
		}
	}
	if code := Run(nil, failedWriter{}, failedWriter{}); code != 1 {
		t.Errorf("failed stderr exit = %d", code)
	}
}

func TestVersionSelection(t *testing.T) {
	old := Version
	t.Cleanup(func() { Version = old })
	for _, tt := range []struct{ override, module, want string }{
		{"dev", "v1.2.3", "v1.2.3"}, {"dev", "(devel)", "dev"},
		{"dev", "", "dev"}, {"v2.0.0", "v1.2.3", "v2.0.0"},
	} {
		Version = tt.override
		if got := version(tt.module); got != tt.want {
			t.Errorf("version = %q, want %q", got, tt.want)
		}
	}
}

func FuzzRun(f *testing.F) {
	for _, seed := range []string{"33mm in, 120f", "1 2m cm", "NaN m", "-459.67f", "1e308f c", "1e-9in", ",", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		code, stdout, _ := run(strings.Fields(input)...)
		if code == 2 && stdout != "" {
			t.Fatal("invalid input produced partial output")
		}
	})
}
