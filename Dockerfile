FROM golang:1.25 AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/controller ./cmd/controller/

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/controller /controller
USER 65532:65532
ENTRYPOINT ["/controller"]
