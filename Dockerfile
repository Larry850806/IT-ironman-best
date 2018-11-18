FROM golang:1.11 AS builder
WORKDIR /src
COPY go.mod go.sum main.go /src/
RUN go build

FROM gcr.io/distroless/base
COPY --from=builder /src/main /src/main
CMD ["/src/main"]