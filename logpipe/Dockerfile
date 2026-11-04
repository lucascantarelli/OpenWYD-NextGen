# syntax=docker/dockerfile:1
# Sidecar de logs: stdlib apenas, imagem final distroless/nonroot. O único
# privilégio é o socket do Docker montado read-only no compose.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY logpipe ./logpipe
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/logpipe ./logpipe

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/logpipe /logpipe
EXPOSE 7788
ENTRYPOINT ["/logpipe"]
