# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/url-shortener ./cmd/url-shortener

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app -u 10001 app && mkdir /data && chown app:app /data
COPY --from=build /out/url-shortener /usr/local/bin/url-shortener
USER 10001:10001
ENV ADDRESS=:8080 DATABASE_PATH=/data/url-shortener.db
VOLUME ["/data"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["url-shortener"]
