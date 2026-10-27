FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/monitor .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/monitor /monitor
EXPOSE 8080
ENTRYPOINT ["/monitor"]
