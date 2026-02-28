FROM golang:1.26-bookworm AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/weather-exporter ./cmd/weather-exporter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/weather-exporter /usr/local/bin/weather-exporter

EXPOSE 9798
ENTRYPOINT ["/usr/local/bin/weather-exporter"]
