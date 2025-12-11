# build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy files to container
COPY main.go ./
COPY templates ./templates/

# Initialize go module and build
RUN go mod init subnet-scanner && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subnet-scanner .

# run stage
FROM alpine:latest

WORKDIR /root/

# Copy the files from builder
COPY --from=builder /app/subnet-scanner .
COPY --from=builder /app/templates ./templates/

EXPOSE 8080

# default values for env variables
ENV PORT=8080
ENV DEFAULT_SUBNET=192.168.1.0/24

# Run application
CMD ["./subnet-scanner"]