FROM golang:1.24-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /apt_exporter ./cmd/apt_exporter

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -r apt-exporter && \
    useradd -r -g apt-exporter -s /sbin/nologin apt-exporter
COPY --from=builder /apt_exporter /usr/local/bin/apt_exporter
USER apt-exporter
EXPOSE 9120
ENTRYPOINT ["/usr/local/bin/apt_exporter"]
CMD ["--apt.rootfs=/host"]
