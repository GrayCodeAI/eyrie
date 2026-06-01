FROM golang:1.26.3-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w \
      -X main.Version=${VERSION} \
      -X main.Commit=${COMMIT} \
      -X main.BuildDate=${BUILD_DATE}" \
    -o eyrie ./cmd/eyrie

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tini && \
    adduser -D -u 1000 -h /data eyrie

COPY --from=builder /build/eyrie /usr/local/bin/eyrie
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

USER eyrie
WORKDIR /data
EXPOSE 8080

ENTRYPOINT ["tini", "--", "eyrie"]
CMD ["serve", "8080"]
