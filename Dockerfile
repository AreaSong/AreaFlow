# syntax=docker/dockerfile:1.7
FROM golang:1.26-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSION=1.0.0-dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/areasong/areaflow/internal/version.Version=${VERSION} -X github.com/areasong/areaflow/internal/version.Commit=${COMMIT} -X github.com/areasong/areaflow/internal/version.Date=${BUILD_DATE}" \
    -o /out/areaflow ./cmd/areaflow

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-build /out/areaflow /usr/local/bin/areaflow
USER nonroot:nonroot
EXPOSE 3847 9090
ENTRYPOINT ["/usr/local/bin/areaflow"]
CMD ["server"]
