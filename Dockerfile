FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/higgsfield-action ./cmd/higgsfield-action

# Root is required: GitHub Actions mounts GITHUB_WORKSPACE as uid 1001 (mode 0755).
# Distroless :nonroot is uid 65532 and cannot mkdir output paths there.
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/higgsfield-action /higgsfield-action

ENTRYPOINT ["/higgsfield-action"]
