# Audiobookshelf Exporter

> [!CAUTION]
> This project is a work in progress — I’m still experimenting with the Audiobookshelf API and tweaking which data gets scraped. Things may break or change unexpectedly!

A lightweight Prometheus exporter for [Audiobookshelf](https://github.com/advplyr/audiobookshelf), written in Go.  
This exporter collects listening statistics, user activity, library metrics, and exposes them for Grafana dashboards.

---

## Features

- Total listening time per book, user, and device
- Duration grouped by day of week
- Active users and session counts
- Library-wide metrics (listening time, session totals)
- Built-in Prometheus `/metrics` endpoint
- Structured JSON logging with [zap](https://github.com/uber-go/zap)

---

## Environment Variables

| Variable | Description | Default |
|-----------|--------------|----------|
| `ABS_URL` | Your Audiobookshelf base URL | `(required)` |
| `ABS_API_KEY` | Your API key (found in Audiobookshelf user settings) | `(required)` |
| `EXPORTER_PORT` | Port to expose metrics on  | `9860` |
| `SCRAPE_INTERVAL_SECONDS` | How often it scrapes. | `30` |
| `LOG_FORMAT` | Optional. `console` for dev logs. | `JSON` |

---

## Example Metrics

`audiobookshelf_users_total`

`audiobookshelf_sessions_total`

`audiobookshelf_user_sessions_total`

`audiobookshelf_user_listening_seconds_total`

`audiobookshelf_book_listening_seconds_total`

`audiobookshelf_weekday_listening_seconds_total`

---

## Docker Usage

### Run manually
```bash
docker run -d \
  --name audiobookshelf-exporter \
  -p 9860:9860 \
  -e ABS_URL="http://192.168.0.106:13378" \
  -e ABS_API_KEY="YOUR_API_KEY" \
  ghcr.io/operationeth/audiobookshelf-exporter:latest
```
### Docker Compose
```bash
  audiobookshelf-exporter:
    image: ghcr.io/operationeth/audiobookshelf-exporter:latest
    container_name: audiobookshelf-exporter
    ports:
      - 9860:9860
    environment:
      ABS_URL: http://192.168.0.106:13378
      ABS_API_KEY: "your_api_key_here"
      EXPORTER_PORT: 9860
    restart: unless-stopped
```  
Example `.env` file:
```bash
ABS_URL=http://192.168.0.106:13378
ABS_API_KEY=abc123
AUDIOBOOKSHELF_EXPORTER_PORT=9860
```

---

This wouldnt be legit if i didnt add a 👉🚀
