# GNAF Geocoder

Australian address geocoder built in Go using [G-NAF Core](https://geoscape.com.au/data/g-naf-core/) and SQLite.

## Features

- Auto-downloads the **latest G-NAF GDA2020** release from [data.gov.au](https://data.gov.au/data/dataset/geocoded-national-address-file-g-naf)
- Loads ~15M addresses from the full G-NAF zip into SQLite
- REST API with hierarchical fallback: street → suburb → postcode → state
- Suburb, SA3 region, and address autocomplete for typeahead UIs
- Returns geocode accuracy level with each response
- Optional API key auth, data freshness checks, and Docker image

## Requirements

- Go 1.27+ (or Docker)
- ~5 GB free disk space for G-NAF download and SQLite database
- Acceptance of the [G-NAF End User Licence Agreement](https://data.gov.au/data/dataset/geocoded-national-address-file-g-naf/resource/09f74802-08b1-4214-a6ea-3591b2753d30)

## Quick start

```bash
make build
./geocoder setup --accept-eula --data-dir ./data --db ./data/gnaf.db
./geocoder serve --db ./data/gnaf.db --port 8080
```

The `setup` step downloads G-NAF (~1.8 GB) and ABS region data, then loads everything into SQLite. Expect 1–2 hours depending on disk speed. Use `--force` to re-download even when local files are up to date.

Test the API:

```bash
curl "http://localhost:8080/v1/geocode?street=1+Collins+St&suburb=Melbourne&state=VIC&postcode=3000"
curl "http://localhost:8080/v1/suggest/addresses?q=1+collins+st+melbourne&state=VIC"
curl "http://localhost:8080/health"
```

## Setup (step by step)

```bash
# Download G-NAF (~1.8 GB ZIP, requires EULA acceptance)
./geocoder data download --accept-eula --data-dir ./data

# Load into SQLite (~30–60 min)
./geocoder data load --data-dir ./data --db ./data/gnaf.db

# Download ABS ASGS 2021 mesh block allocation
./geocoder regions download --data-dir ./data

# Assign SA3 regions and rebuild centroids (~5 min)
./geocoder regions load --data-dir ./data --db ./data/gnaf.db

# Start API
./geocoder serve --db ./data/gnaf.db --port 8080
```

## Docker

Build and run with a mounted data volume (populate `/data` via `setup` first, or bind-mount a host `./data` directory):

```bash
docker build -t geocoder .
docker run --rm -p 8080:8080 -v "$(pwd)/data:/data" geocoder
```

With API key auth:

```bash
docker run --rm -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  -e API_KEY=dev-key \
  geocoder
```

## API

| Endpoint | Description |
|----------|-------------|
| `GET /v1/geocode` | Geocode an address (`street`, `suburb`, `state`, `postcode`) |
| `GET /v1/suggest/suburbs` | Suburb autocomplete |
| `GET /v1/suggest/regions` | SA3 region autocomplete |
| `GET /v1/suggest/addresses` | Address autocomplete |
| `GET /health` | Liveness check |
| `GET /health/data` | Database vs local manifest freshness |
| `GET /version` | Application version |

```bash
curl "http://localhost:8080/v1/geocode?street=1+Collins+St&suburb=Melbourne&state=VIC&postcode=3000"
curl "http://localhost:8080/v1/suggest/suburbs?q=mount&state=VIC"
curl "http://localhost:8080/v1/suggest/regions?q=melbourne&state=VIC"
curl "http://localhost:8080/v1/suggest/addresses?q=42+demo+rd+rich&state=VIC"
```

When `API_KEY` is set (comma-separated list), `/v1/*` routes require a key via `X-API-Key` header or `Authorization: Bearer <key>`. `/health`, `/health/data`, and `/version` stay open.

```bash
API_KEY=dev-key,prod-key ./geocoder serve --db ./data/gnaf.db
curl -H "X-API-Key: dev-key" "http://localhost:8080/v1/geocode?suburb=Melbourne&state=VIC&postcode=3000"
```

Example geocode response:

```json
{
  "latitude": -37.8136,
  "longitude": 144.9631,
  "accuracy": "street",
  "matched_address": "1 Collins St Melbourne VIC 3000",
  "address_detail_pid": "...",
  "address": {
    "number": "1",
    "street": "Collins St",
    "suburb": "Melbourne",
    "state": "VIC",
    "postcode": "3000",
    "region": "Melbourne City"
  },
  "address_slug": "1-collins-st-melbourne-vic-3000",
  "suburb_slug": "melbourne-vic-3000",
  "region_slug": "melbourne-city-vic"
}
```

`address.region` is the ABS SA3 name, populated after running `regions load`.

Suggest endpoints accept `q` (min 2 chars), optional `state`, and optional `limit` (default 10, max 25). Address suggest parses street and suburb from `q` (e.g. `42 demo rd richmond vic 3121`); optional `suburb`, `state`, and `postcode` params narrow results further.

API responses include an `X-API-Version` header. Version format is `YYYY.MM.DD+build` (e.g. `2026.09.02+1`).

## Keeping data up to date

```bash
./geocoder data check-update --data-dir ./data
./geocoder regions check-update --data-dir ./data
```

After downloading newer files, reload the database:

```bash
./geocoder data load --data-dir ./data --db ./data/gnaf.db
./geocoder regions load --data-dir ./data --db ./data/gnaf.db
```

`/health` returns a simple liveness check. `/health/data` compares database metadata against local manifests. Start the server with `--if-stale` to also check data.gov.au and print warnings:

```bash
./geocoder serve --db ./data/gnaf.db --data-dir ./data --if-stale
```

## Commands

| Command | Description |
|---------|-------------|
| `setup` | Download G-NAF + ABS allocation and load into SQLite |
| `data download` | Download latest G-NAF Core zip from data.gov.au |
| `data load` | Import G-NAF zip into SQLite and build centroid tables |
| `data check-update` | Compare local G-NAF manifest against latest CKAN release |
| `regions download` | Download ABS ASGS 2021 mesh block allocation |
| `regions load` | Assign SA3 regions and rebuild centroids |
| `regions check-update` | Compare local region manifest against latest CKAN resource |
| `serve` | Start REST geocoding API (`--if-stale` warns when data is behind CKAN) |
| `version` | Print application version |

## Development

```bash
make build          # build binary with embedded VERSION
make test           # run unit tests
make clean          # remove binary
make version-bump   # bump build number in VERSION file
```

Integration tests (require a loaded database):

```bash
GEOCODER_TEST_DB=./data/gnaf.db go test -tags=integration ./internal/geocode/...
```

## Project layout

```
cmd/geocoder/     CLI entrypoint
internal/api/     HTTP handlers and server
internal/geocode/ Geocoding and suggest logic
internal/store/   SQLite schema, import, and centroids
internal/gnaf/    G-NAF download and manifest
internal/regions/ ABS SA3 region enrichment
internal/parse/   Street address parsing
internal/normalize/ Text normalization
```

## License

G-NAF data is subject to the [G-NAF End User Licence Agreement](https://data.gov.au/data/dataset/geocoded-national-address-file-g-naf/resource/09f74802-08b1-4214-a6ea-3591b2753d30). Do not use for mail generation without secondary address verification.
