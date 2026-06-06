FROM golang:1.26.4-alpine AS builder

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

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /build/eyrie /usr/local/bin/eyrie
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

USER nonroot
WORKDIR /data
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/eyrie"]
CMD ["serve", "8080"]
