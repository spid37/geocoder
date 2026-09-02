# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build

ENV GOTOOLCHAIN=auto
RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN VERSION="$(tr -d '[:space:]' < VERSION)" && \
	CGO_ENABLED=0 go build -trimpath \
	-ldflags="-s -w -X github.com/spid37/geocoder/internal/version.ldflagsVersion=${VERSION}" \
	-o /geocoder ./cmd/geocoder

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
	&& adduser -D -H -u 10001 geocoder

COPY --from=build /geocoder /usr/local/bin/geocoder

USER geocoder

EXPOSE 8080

VOLUME ["/data"]

# Optional comma-separated API keys for /v1/* routes. Unset = open access.
# ENV API_KEY=key-one,key-two


ENTRYPOINT ["geocoder"]
CMD ["serve", "--db", "/data/gnaf.db", "--data-dir", "/data", "--port", "8080"]
