FROM golang:1.25-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -trimpath -ldflags="-s -w" -o /jade-gamble .

FROM alpine:3.20
RUN adduser -D -H player && apk add --no-cache ca-certificates tzdata
USER player
WORKDIR /app
COPY --from=build /jade-gamble /app/jade-gamble
ENV TZ=Asia/Taipei PORT=3002 DB_PATH=/data/jade.db MOCK_AUTH=0
EXPOSE 3002
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q -O /dev/null http://127.0.0.1:3002/api/health || exit 1
ENTRYPOINT ["/app/jade-gamble"]