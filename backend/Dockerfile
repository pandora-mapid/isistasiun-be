# ---- dev target: hot reload with air ----
FROM golang:1.25-alpine AS dev
RUN apk add --no-cache git
WORKDIR /app
ARG AIR_VERSION=v1.67.0
RUN go install github.com/air-verse/air@${AIR_VERSION}
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# ---- prod target: minimal runtime image ----
FROM alpine:3.20 AS prod
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/api ./api
COPY migrations ./migrations
EXPOSE 8080
CMD ["./api"]
