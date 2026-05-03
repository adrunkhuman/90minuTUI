# 90minuTUI

Terminal UI for browsing `90minut.pl` (Polish football archive) without an API.

## Scope

- Small, single-binary Go app.
- Read-only browsing flow: `season -> competition submenu or league -> fixture -> match`, with season/competition selection available as a popup.
- Startup preloads the selected season and opens its preferred competition (defaults to `Ekstraklasa` when present), then lands on standings + fixtures with the latest drillable fixture selected when match details exist, or the latest completed result when the whole competition is linkless.
- Fast navigation and robust HTML parsing over feature breadth.

## Current Features

- Season/competition popup selector from `archsezon.php`, including submenu navigation for III liga, regional leagues, and regional cups.
- Standings plus round/fixture browsing for the selected league, with rounds normalized by round number, fixture-derived round date spans when parseable, and fixtures within each round ordered by parsed match date.
- Linkless fixture support for competitions where results exist without match pages; those fixtures stay browsable in standings/round context and surface an unavailable-details state instead of opening match view.
- Match details view with:
  - centered score line with scorer rows anchored to a shared minute column
  - side-aware timeline with halftime/full-time dividers
  - compact event markers: `⚽` goal, `❌` missed penalty, and red-card blocks, alongside status markers like `HT`, `FT`, `AET`, `PPD`, and `OFF`
  - side-by-side lineups aligned around the same center axis, with substitutions and card badges shown in lineups, including inline substitution annotations, rather than as separate yellow-card timeline markers
  - match date/details rendered directly under the score header
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
- `enter` load/open the selected season, competition submenu, league, or match
- `esc` close match details, step back through submenu history, or open/close the selector popup
- `r` reload current context
- `q` quit

In match view, `pgup`/`pgdn` and `ctrl+u`/`ctrl+d` scroll details. `j/k` and `h/l` still change fixture context, and empty rounds clear stale match data without closing the pane. If the selected fixture has no match page, the UI keeps league context and shows an unavailable hint instead of opening details. If standings are missing on the source page, the UI keeps the layout and shows an unavailable state instead.

## Architecture

- `cmd/90minutui` - app entrypoint
- `internal/site` - HTTP fetch + HTML parsing into typed models
- `internal/ui` - Bubble Tea state/update/view

Core pipeline: `fetch -> parse -> render`.

## Development Notes

- 90minut pages use legacy encoding (`iso-8859-2`) and must be decoded before parsing.
- Prefer semantic selectors over fixed table offsets.
- URL IDs are treated as stable keys (e.g. season/match links).
- League parsing preserves standings table order from the source page, but normalizes rounds by detected round number and fixtures by parsed `WhenInfo` date/time so all consumers see a stable sequence. Parsed fixtures may still be valid when `MatchURL` and `MatchID` are empty.
- Async UI loads are keyed by season/competition/fixture IDs so stale responses are ignored if focus changes.
- Match parsing still assumes mostly three-cell timeline rows and uses heuristic table scoring; add fixtures/tests when source layout drifts.

## Fixture Corpus Maintenance

Parser tests rely on saved HTML fixtures in `internal/site/testdata/fixtures` and `internal/site/testdata/manifest.json`.

```bash
go run ./cmd/fetchfixtures
go test ./...
```

Use this flow when parser behavior changes or when upstream HTML structure drifts. Keep fixture notes in `manifest.json` specific about the parser behavior each saved page protects.

## Quality Gates

- CI runs `go test ./...` on pull requests and on pushes to `master`.
- Local pre-commit checks use `prek` with pre-commit-compatible `.pre-commit-config.yaml`.

```bash
prek install
prek run -a
```
