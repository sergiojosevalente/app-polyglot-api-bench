# ---- build stage: toolchain Rust lengkap (~1.5GB), dibuang setelah build ----
FROM rust:1-slim-bookworm AS builder
WORKDIR /app

# Trik cache dependency: build dulu dengan main.rs kosong, sehingga layer
# dependency (axum, tokio, serde...) hanya dikompilasi ulang bila Cargo.toml berubah.
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs \
    && cargo build --release \
    && rm -rf src

COPY src ./src
RUN touch src/main.rs \
    && RUSTC_VERSION="$(rustc --version)" cargo build --release

# ---- runtime stage: distroless "cc" = static + glibc (Rust default link ke glibc) ----
FROM gcr.io/distroless/cc-debian12:nonroot
COPY --from=builder /app/target/release/rust-service /rust-service
EXPOSE 8080
ENTRYPOINT ["/rust-service"]
