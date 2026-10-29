# git-stats

Data is cached in `data/` for 24 hours.

## SVG

```text
http://localhost:8080/?limit=6&exclude=HTML,CSS
```

| Parameter | Description                                                                                  |
| --------- | -------------------------------------------------------------------------------------------- |
| `limit`   | Positive integer limiting displayed languages; default: all. Invalid values return HTTP 400. |
| `exclude` | Comma-separated language names, case-insensitive. May be repeated.                           |
