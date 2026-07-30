FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /server ./cmd/server
RUN go build -o /worker ./cmd/worker

FROM alpine:latest
RUN apk add --no-cache docker-cli tzdata libc6-compat

WORKDIR /app
COPY --from=builder /server /app/server
COPY --from=builder /worker /app/worker

# Ensure uploads directory exists (populated by the Docker volume at runtime)
RUN mkdir -p /app/uploads

EXPOSE 8080
CMD ["/app/server"]
