# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`AGENTS.md` holds the authoritative project rules; this file summarizes them and
adds architecture notes. Read `AGENTS.md` when the two disagree.

## Commands

```sh
go test ./...                                   # required before completing any change
go test ./domains/ -run TestBiomeTableCoversEveryClimate   # single test
go test ./pipelines/... -run TestRunProducesDeterministicPNG
gofmt -w <changed files>                        # format every changed Go file
go mod tidy                                     # after dependency changes; review go.mod and go.sum
```

Run a pipeline end to end (all `mappe` subcommands require `--output`):

```sh
go run ./cmd/mappe                              # list pipelines
go run ./cmd/mappe flat-terrain-png --seed 42 --width 640 --height 320 \
  --iterations 10000 --wrap --output /tmp/flat-terrain.png
go run ./cmd/olsson -seed 42 > world.json       # normalized height map as JSON
go run ./cmd/mappe version                      # Mappe core version
```

`go.mod` declares Go 1.22.12 even when a newer toolchain is installed locally;
do not use language or standard-library features newer than that.

## Architecture

The repository is a three-stage pipeline architecture. Packages are layered and
the dependency direction is strict:

```
generators  ->  domains  <-  renderers
        \                   /
         \                 /
           pipelines  ->  cmd/mappe
```

- **`domains/`** is the shared vocabulary and the only package generators and
  renderers have in common. It owns immutable, validated value types
  (`NormalizedHeightMap`, `HeightField`, `EnvironmentalMap`) plus the stable
  elevation, heat, moisture, climate, and 27-value terrain enumerations
  migrated from `wgva`. Constructors validate and defensively copy; accessors
  return copies; out-of-range coordinate access panics.
- **Generators** (`olsson/`, `internal/generators/flat/`) turn a `Config` into a
  domain object. Every generator exposes both `GenerateNormalizedHeightMap` and
  `GenerateHeightField`; the height-field variant is where the generator
  *derives* its `domains.GridTopology` (Olsson: east-west wrap only, distinct
  polar edges; flat: both axes from `Config.Wrap`). Downstream stages must read
  topology from the field rather than take duplicate wrapping flags.
- **Renderers** (`renderers/*`) take a domain object plus an optional renderer
  `Config` and write an external format to an `io.Writer`. They never generate.
- **Pipelines** (`pipelines/<name>/`) are the public integration boundary. Each
  exposes a single `Run(w io.Writer, generatorConfig, rendererConfig) error` and
  wraps every stage error as `"<pipeline-name>: <stage>: %w"` so a failure names
  the stage; tests assert on that. `internal/` generators are reachable by
  callers only through a named pipeline.
- **`cmd/mappe`** is a Cobra tree with one subcommand per pipeline. It owns flag
  defaults, converts `--seed` into `rand.NewPCG(uint64(seed), 0)`, and creates
  the output file. Commands hold no generation logic.

### Value conventions

- All physical values are normalized `float64` in `[0, 1]`. `0.5` is the
  neutral value for heat, moisture, basin, and volcanic tendency; signed
  source-model values are mapped linearly onto that range.
- Sea level is **not** `0.5`. Generators min-max normalize, so elevation
  boundaries come from a `domains.ElevationLevels` value (absolute elevations for
  abyss, shelf, sea level, upland, highland, mountain) produced by a swappable
  stage; the default is `internal/levels/percentile`, which checks tile
  percentages (default 48% ocean) against the histogram. One pixel is one tile.
- Grids are finite, Cartesian, and row-major (`y*width + x`).
- Terrain, climate, and elevation IDs are serialized domain values: their
  numeric IDs and `String()` names are stable and must not be renumbered.
  `TerrainInlandSea` and `TerrainLake` hold reserved IDs but are not yet
  produced by the classifier.
- `ClassificationConfig.Validate` enforces the ordering that keeps every
  declared band and terrain rule reachable; the ordered classifier in
  `domains/terrain.go` (ocean depth, ice, volcanic, elevated, wetland, coast,
  then the climate-cover biome table) preserves the source precedence, and
  `TestTerrainRulesPreserveSourcePrecedence` guards it.

## Go conventions

- Errors are declared as constants via `internal/cerrs.Error`, package-prefixed
  (`"olsson: width must be twice height"`). Library code returns errors; it does
  not log or exit.
- Determinism is the contract. Configs carry a caller-owned `math/rand/v2`
  `rand.Source`; never use `math/rand`, wall-clock time, or process-global
  randomness, and avoid mutable package-level state.
- Every Go file carries the AGPL-3.0 header block used throughout the tree.
- Prefer the standard library; new dependencies need narrow justification.
- Keep generator-specific code and its tests in one package. Share code only
  after two accepted generators demonstrate the same need.

## Tests

- Use fixed seeds. Tests should distinguish the intended behavior from plausible
  wrong implementations, especially at coordinate, wrapping, and range
  boundaries.
- Add golden fixtures only when the serialized or rendered bytes are themselves
  the contract; the current pipeline tests instead assert determinism by
  rendering twice and comparing.
- When visual output changes, inspect representative PNGs in addition to running
  the tests.

## Migration workflow

The repository is in an inventory-and-review phase. Every source project in the
`README.md` table has a linked GitHub issue that is its decision record.

- Do not import code from a source project until its issue records an explicit
  **Include** decision.
- When importing, preserve the license, attribution, deterministic behavior,
  useful tests, and documentation; place preserved notices in `docs/licenses/`.
- Record the migration boundary and anything intentionally omitted in the review
  issue, and update the README inventory when a project's status changes.
- Do not publish private repository names or metadata here without authorization.

## Delivery

- Work directly on `main` unless asked otherwise. After the tests pass, commit
  and push to `origin/main` unless the user says not to.
- Assign every issue and pull request to `mdhender` at creation
  (`gh issue create --assignee @me`, `gh pr create --assignee @me`).
- Keep generated artifacts (PNGs, JSON output) out of version control.
