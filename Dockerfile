# Build stage: full Go toolchain with module and build caches.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/adera-api ./cmd/api

# Runtime stage: distroless static — CA certs, tzdata, and a non-root user
# included; no shell or package manager.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/adera-api /adera-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/adera-api"]
CMD ["serve"]
