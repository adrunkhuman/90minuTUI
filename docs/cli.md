# CLI Data Export

`90minutui` exports parsed `90minut.pl` data as JSON for scripts.

Data goes to stdout. Errors go to stderr. Fetch or parse failure exits non-zero.

## Usage

```bash
90minutui seasons
90minutui competitions --season 107
90minutui competitions --url 'http://www.90minut.pl/ligireg.php?poziom=4&id_sezon=105'
90minutui league 'http://www.90minut.pl/liga/1/liga14072.html'
90minutui fixtures 'http://www.90minut.pl/liga/1/liga14072.html'
90minutui match 1930640
```

Use exported `url` values for exact follow-up fetches. `league_key` is stable for joins and is accepted as a convenience when the league URL uses a known `90minut.pl` league path.

## Exports

### seasons

```json
{
  "seasons": [
    {
      "label": "2025/26",
      "season_id": "107",
      "url": "http://www.90minut.pl/archsezon.php?id_sezon=107",
      "current": true
    }
  ]
}
```

### competitions

```json
{
  "title": "Competitions",
  "season_id": "107",
  "url": "http://www.90minut.pl/archsezon.php?id_sezon=107",
  "competitions": [
    {
      "name": "PKO Bank Polski Ekstraklasa 2025/2026",
      "league_key": "liga14072",
      "url": "http://www.90minut.pl/liga/1/liga14072.html"
    }
  ]
}
```

### league

```json
{
  "title": "PKO Bank Polski Ekstraklasa 2025/2026",
  "league_key": "liga14072",
  "url": "http://www.90minut.pl/liga/1/liga14072.html",
  "standings": [
    {
      "position": 1,
      "team": "Team",
      "played": 1,
      "won": 1,
      "drawn": 0,
      "lost": 0,
      "points": 3
    }
  ],
  "rounds": []
}
```

### fixtures

```json
{
  "league_key": "liga14072",
  "url": "http://www.90minut.pl/liga/1/liga14072.html",
  "rounds": [
    {
      "phase": "",
      "section": "",
      "name": "1. kolejka",
      "fixtures": [
        {
          "home": "Team A",
          "away": "Team B",
          "score": "1-0",
          "when": "18 lipca, 18:00",
          "match_id": "1930640",
          "match_url": "http://www.90minut.pl/mecz.php?id_mecz=1930640"
        }
      ]
    }
  ]
}
```

### match

```json
{
  "match_id": "1930640",
  "url": "http://www.90minut.pl/mecz.php?id_mecz=1930640",
  "title": "Match title",
  "competition": "League or cup name",
  "meta": "stadium, attendance, referee",
  "weather": "weather text",
  "home_team": "Team A",
  "away_team": "Team B",
  "score": "1-0",
  "events": [
    {
      "minute": 35,
      "minute_text": "35",
      "stoppage": 0,
      "has_minute": true,
      "kind": "GOAL",
      "team_side": "home",
      "text": "Player 35",
      "substitution_out": "",
      "substitution_in": ""
    }
  ],
  "home_lineup": [
    {
      "name": "Player A",
      "events": [],
      "raw_text": "Player A"
    }
  ],
  "away_lineup": [],
  "news_title": "",
  "news_url": ""
}
```

## Notes

- Subcommands write JSON.
- Missing lists are `[]`.
- Missing source values are `""`.
- `season_id`, `league_key`, and `match_id` are stable keys for joining exports.
- `url` is the exact fetch identifier for follow-up commands.
- The app uses a per-process cache only. Each new process fetches fresh data.
