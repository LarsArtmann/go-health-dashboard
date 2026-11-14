# Build stage needs GOEXPERIMENT=jsonv2 because go-sse uses encoding/json/v2.
# Base pinned ABOVE go.mod's `go` floor (currently 1.27; floors move only via
# the documented fleet-floor process): an image older than the floor fails
# `go mod download` with "go.mod requires go >= ..." (hit 2026-09-23: the
# compose stack had never been booted since the floor moved). When bumping
# go.mod's floor, move this pin in the same change.
FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOEXPERIMENT=jsonv2 go build -o /dashboard-demo ./example

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /dashboard-demo /dashboard-demo
EXPOSE 8080
ENTRYPOINT ["/dashboard-demo"]
