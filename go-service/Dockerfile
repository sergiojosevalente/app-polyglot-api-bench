# ---- build stage: toolchain Go lengkap (~300MB), dibuang setelah build ----
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
# CGO_ENABLED=0 -> binary statis murni, bisa jalan di image tanpa libc.
# -ldflags="-s -w" membuang symbol table & debug info -> binary lebih kecil.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/go-service ./cmd/server

# ---- runtime stage: hanya binary, tanpa shell/package manager ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/go-service /go-service
EXPOSE 8080
ENTRYPOINT ["/go-service"]
