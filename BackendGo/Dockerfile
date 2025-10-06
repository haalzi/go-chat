# --- builder ---
FROM golang:1.22 AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -o /out/app ./cmd/api

# --- runner ---
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/app /app
COPY web /web
ENV ADDR=:8080
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]