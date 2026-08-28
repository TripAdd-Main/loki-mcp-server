FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/loki-mcp-server ./cmd/loki-mcp-server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/loki-mcp-server /usr/local/bin/loki-mcp-server

# Runs on stdio by default; set MCP_HTTP_ADDR (e.g. :8080) for streamable HTTP.
ENTRYPOINT ["/usr/local/bin/loki-mcp-server"]
