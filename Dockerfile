# Build stage needs GOEXPERIMENT=jsonv2 because go-sse uses encoding/json/v2.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOEXPERIMENT=jsonv2 go build -o /dashboard-demo ./example

FROM gcr.io/distroless/static-debian12
COPY --from=build /dashboard-demo /dashboard-demo
EXPOSE 8080
ENTRYPOINT ["/dashboard-demo"]
