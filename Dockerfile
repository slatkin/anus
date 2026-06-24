FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN VITE_API=web npm run build

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN go build -tags production -o /anus-web ./cmd/anus-web

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /anus-web /usr/local/bin/anus-web
ENV DATA_DIR=/data
VOLUME /data
EXPOSE 8080
CMD ["anus-web"]
