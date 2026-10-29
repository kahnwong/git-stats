# git-stats

## Dev

Start dev server via `go run . -dev`.

Access via:

```text
http://localhost:8080/?limit=6&exclude=HTML,CSS
```

| Parameter | Description                                                                                  |
| --------- | -------------------------------------------------------------------------------------------- |
| `limit`   | Positive integer limiting displayed languages; default: all. Invalid values return HTTP 400. |
| `exclude` | Comma-separated language names, case-insensitive. May be repeated.                           |
