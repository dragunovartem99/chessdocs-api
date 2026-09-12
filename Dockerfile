# Build the binary against the module cache, then ship it alone: the runtime
# image holds no toolchain, no shell, and no source.
FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/chessdocs-api ./cmd/chessdocs-api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/chessdocs-api /usr/local/bin/chessdocs-api

USER nonroot:nonroot
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/chessdocs-api"]
