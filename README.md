# 90minuTUI

Terminal UI for browsing `90minut.pl` (Polish football archive) without an API.

## Scope

- Small, single-binary Go app.
- Read-only browsing flow: `league -> fixture -> match`, with season/league selection available as a popup.
- Startup preloads the selected season and opens its preferred competition (defaults to `Ekstraklasa` when present), then lands on standings + fixtures.
- Fast navigation and robust HTML parsing over feature breadth.

## Current Features

- Season/competition popup selector from `archsezon.php`.
- Standings plus round/fixture browsing for the selected league.
- Match details view with:
  - competition/date/meta
  - score
  - side-aware timeline (goals/cards/subs)
  - side-by-side lineups
  - persistent standings/fixture context beside match details

## Run

```bash
go run ./cmd/90minutui
```

## Controls

- `tab` focus cycle inside the season/competition popup
- `j/k` move in the focused list; in match view they jump to the previous/next fixture
- `h/l` previous/next round; in match view they jump to the first fixture in that round
- `pgup`/`pgdn` or `ctrl+u`/`ctrl+d` scroll the match detail pane
- `enter` load/open
- `esc` close match details or open/close the selector popup
- `r` reload current context
- `q` quit

In match view, `pgup`/`pgdn` and `ctrl+u`/`ctrl+d` scroll details. `j/k` and `h/l` still change fixture context, and empty rounds clear stale match data without closing the pane. If standings are missing on the source page, the UI keeps the layout and shows an unavailable state instead.

## Architecture

- `cmd/90minutui` - app entrypoint
- `internal/site` - HTTP fetch + HTML parsing into typed models
- `internal/ui` - Bubble Tea state/update/view

Core pipeline: `fetch -> parse -> render`.

## Development Notes

- 90minut pages use legacy encoding (`iso-8859-2`) and must be decoded before parsing.
- Prefer semantic selectors over fixed table offsets.
- URL IDs are treated as stable keys (e.g. season/match links).
- Async UI loads are keyed by season/competition/fixture IDs so stale responses are ignored if focus changes.
- Match parsing still assumes mostly three-cell timeline rows and uses heuristic table scoring; add fixtures/tests when source layout drifts.

## Fixture Corpus Maintenance

Parser tests rely on saved HTML fixtures in `internal/site/testdata/fixtures` and `internal/site/testdata/manifest.json`.

```bash
go run ./cmd/fetchfixtures
go test ./...
```

Use this flow when parser behavior changes or when upstream HTML structure drifts.

## Quality Gates

- CI runs `go test ./...` on pull requests and on pushes to `master`.
- Local pre-commit checks use `prek` with pre-commit-compatible `.pre-commit-config.yaml`.

```bash
prek install
prek run -a
```
