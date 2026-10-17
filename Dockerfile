# Build stage needs GOEXPERIMENT=jsonv2 because go-sse uses encoding/json/v2.
# Base pinned to go.mod's floor (go 1.27.1) — an older golang image fails
# `go mod download` with "go.mod requires go >= 1.27.1" (hit 2026-09-23:
# the compose stack had never been booted since the floor moved).
FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOEXPERIMENT=jsonv2 go build -o /dashboard-demo ./example

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /dashboard-demo /dashboard-demo
EXPOSE 8080
ENTRYPOINT ["/dashboard-demo"]
