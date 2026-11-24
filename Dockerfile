# Stage 1: Build frontend
FROM node:26-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.27.4-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist/ ./internal/api/frontend_dist/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ztp-dashboard .
RUN mkdir -p /data

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /home/nonroot
ENV ZTP_DB_PATH=/data/ztp-events.db
COPY --from=go-builder /ztp-dashboard /ztp-dashboard
COPY --from=go-builder --chown=65532:65532 /data /data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/ztp-dashboard"]
CMD ["serve"]
