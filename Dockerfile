FROM golang:1.26.2-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/trendstream ./cmd/trendstream

FROM alpine:3.22

RUN addgroup -S trendstream && adduser -S trendstream -G trendstream

WORKDIR /app
COPY --from=build /out/trendstream /app/trendstream

USER trendstream

EXPOSE 8080 9090
ENTRYPOINT ["/app/trendstream"]
