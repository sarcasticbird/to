# to

Measurements, translated. `to` is a tiny command-line converter for lengths
and temperatures. Ask for inches, Celsius, or another unit—or leave off the
target and get a short list of useful equivalents.

One Go binary. No runtime dependencies, network requests, or configuration.

```text
$ to 33mm
33 mm
  3.3 cm
  1.299 in
  ≈ 1 19/64 in

$ to 72f c
22.22 °C
```

## Install

With Go 1.26 or newer:

```sh
go install github.com/sarcasticbird/to/cmd/to@latest
to 33mm in
```

Make sure your Go binary directory (`GOBIN`, or `$(go env GOPATH)/bin` by
default) is on your `PATH`. Until the first tagged release, `@latest` installs
the latest available development revision.

### Build from source

[Flox](https://flox.dev/docs/install-flox/) supplies the pinned Go toolchain:

```sh
git clone https://github.com/sarcasticbird/to.git
cd to
flox activate -- go build -o bin/to ./cmd/to
export PATH="$PWD/bin:$PATH"
to 33mm
```

To install from the checkout instead, run `flox activate -- go install ./cmd/to`.
Homebrew packaging is planned; there is no formula yet.

## Everyday commands

```text
to 33mm                       Show useful equivalents
to 33mm in                    Convert to one unit
to 33 mm inches               Separate the value and unit
to 1.8m                       Show meters, feet and inches, and yards
to 72f c                      Convert Fahrenheit to Celsius
to 44mm, 100mm, 120cm          Convert several measurements
to 33mm in, 120f               Mix direct conversions and summaries
to --help                     Show the command reference
to --version                  Show the build version
```

The grammar is small:

```text
to <measurement> [target-unit]
to <value> <source-unit> [target-unit]
to <expression>, <expression> [...]
```

### One answer or a short list

```text
$ to 33mm in
1.299 in

$ to -40 f c
-40 °C
```

Omit the target to get equivalents suited to the measurement's size. Small
lengths show metric units and inches; human-sized lengths show feet and inches;
distances show kilometers and miles. Temperatures show the other two scales.

```text
$ to 1.8m
1.8 m
  ≈ 5 ft 10 55/64 in
  1.969 yd
```

### Several measurements at once

Commas separate complete expressions. Each expression can have its own target
unit or produce a summary:

```text
$ to 33mm in, 120f
1.299 in

120 °F
  48.89 °C
  322 K

$ to 33 mm in, 120 f c, 2 km mi
1.299 in

48.89 °C

1.243 mi
```

`to` validates the entire comma-separated batch before printing. Empty or
invalid items report their position and produce no partial results.

## Supported units

| Dimension | Symbols | Names and aliases |
| --- | --- | --- |
| Length | `mm`, `cm`, `m`, `km`, `in`, `ft`, `yd`, `mi` | Metric and US/UK singular or plural names, including `metre` spellings |
| Temperature | `c`, `f`, `k` | Celsius, Fahrenheit, Kelvin, and optional degree symbols |

Symbols and names are case-insensitive. Decimal and scientific notation work,
including `.5m` and `1e3mm`. Quote multiword aliases in your shell:

```sh
to 20 "degrees celsius" f, 1m cm
```

## Precision and errors

Decimal output is rounded to four significant digits, with scientific notation
for very small or large values. Fractional inches use the nearest 1/64 inch;
`≈` marks a fraction that differs from the calculated value beyond floating-point
roundoff. Calculations use float64, so this is a convenient everyday converter,
not an arbitrary-precision calculator. Input fractions such as `1/2in` are not
supported; use `.5in`.

Unknown units, incompatible dimensions, temperatures below absolute zero, and
values outside the supported numeric range produce an error. A batch is
validated before any results are printed.

| Exit status | Meaning |
| --- | --- |
| `0` | Success, including help and version output |
| `1` | Output error |
| `2` | Invalid input or unsupported conversion |

Results go to stdout; diagnostics go to stderr.

## Develop to

Run the checks and build a local binary with the pinned Flox environment:

```sh
flox activate -- go test ./...
flox activate -- go test -race ./...
flox activate -- go vet ./...
flox activate -- go build -o ./bin/to ./cmd/to
```

The code has three small layers: `internal/measure` parses and converts,
`internal/presentation` formats answers, and `internal/cli` handles arguments,
batches, and exit statuses. `cmd/to` connects the CLI to the shell.

## License

Apache License 2.0. See [LICENSE](LICENSE).
