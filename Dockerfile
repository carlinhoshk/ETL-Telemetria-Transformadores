# Multi-stage Go build: single image with all platform binaries.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/ingestion ./cmd/ingestion && \
    CGO_ENABLED=0 go build -o /out/simulator ./cmd/simulator && \
    CGO_ENABLED=0 go build -o /out/dbmigrate ./cmd/dbmigrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/* /usr/local/bin/
COPY --from=build /src/migrations /migrations
COPY --from=build /src/dbt/seeds/transformers.csv /seed/transformers.csv
WORKDIR /
ENTRYPOINT []
