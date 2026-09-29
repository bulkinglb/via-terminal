# Contributing

Issues and pull requests are welcome.

## Setup

    go build
    go vet ./... && go test ./...

The tests don't need a keyboard. When you try changes on a real board, run `via-terminal export` first and `via-terminal import` after, so it ends up as it was.

## Adding a keyboard

Put its VIA v3 definition in `internal/defs/keyboards/` and note where it came from in `SOURCES.md`. Only add files whose license allows it.

## Style

- `gofmt`, small functions, return errors instead of panicking.
- Comments explain why, not what.
- No new dependencies for what a few lines can do.
- This project is MIT: don't copy code from GPL projects like VIA or via-cli. Keycode numbers and protocol details are fine.

## Commits and branches

- [Conventional Commits](https://www.conventionalcommits.org): `feat(#12): add layer names`, or `fix: …` without an issue. Lowercase, imperative, no period.
- Branches: `<type>/<issue>-<short-desc>`, e.g. `feat/12-layer-names`.
