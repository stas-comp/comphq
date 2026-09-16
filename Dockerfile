# syntax=docker/dockerfile:1
# Build stage: pinned by digest (docs/decisions.md D-02).
FROM golang:1.27.1-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "-X main.version=${VERSION}" -o /comphq ./cmd/comphq

# Final stage: distroless, nothing to patch (SPEC B1).
FROM gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3
COPY --from=build /comphq /comphq
EXPOSE 8080
ENTRYPOINT ["/comphq"]
