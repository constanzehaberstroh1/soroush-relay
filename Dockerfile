# Stage 1: Build the React frontend
FROM oven/bun:1.2-alpine AS frontend-builder
WORKDIR /app

# Copy bun dependency configuration
COPY server-panel/package.json server-panel/bun.lock /app/server-panel/
WORKDIR /app/server-panel
RUN bun install --frozen-lockfile

# Copy the rest of the frontend source code
COPY server-panel/ /app/server-panel/
# Build the frontend which outputs to ../server/dist
RUN bun run build

# Stage 2: Build the Go application
FROM golang:1.26.2-alpine AS backend-builder
WORKDIR /app

# Copy dependency manifests and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the shared soroushlib package (used by server)
COPY soroushlib/ ./soroushlib/

# Copy the compiled frontend assets from Stage 1 into server/dist
COPY --from=frontend-builder /app/server/dist/ ./server/dist/
# Copy the rest of the server source code
COPY server/ ./server/

# Build the self-contained Go server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o soroush-server ./server

# Stage 3: Final minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

# Run as a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app
COPY --from=backend-builder /app/soroush-server .

# Clever Cloud expects the app to run on port 8080 by default (which we also configured)
EXPOSE 8080
ENV PORT=8080

ENTRYPOINT ["./soroush-server"]
