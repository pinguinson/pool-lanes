# The SQLite driver is pure Go, so the binary cross-compiles without cgo.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go page.html ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /pool-lanes .

# Alpine rather than distroless for its wget, which the homelab's health
# checks use.
FROM alpine:3.24
COPY --from=build /pool-lanes /usr/local/bin/pool-lanes
# Unraid's nobody:users, which owns /mnt/user/appdata.
USER 99:100
VOLUME /data
EXPOSE 8080
HEALTHCHECK CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["pool-lanes"]
CMD ["-db", "/data/pool-lanes.db", "-addr", ":8080"]
