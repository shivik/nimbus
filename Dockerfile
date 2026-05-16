# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/nimbus ./cmd/nimbus

# Runtime stage - tiny final image
FROM scratch
COPY --from=build /out/nimbus /nimbus
EXPOSE 4566
ENTRYPOINT ["/nimbus"]
