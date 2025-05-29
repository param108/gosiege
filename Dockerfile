# Use the official Golang image to build the binary
FROM golang:1.23 AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Go binary
RUN go build -o app .

FROM debian:bookworm

WORKDIR /app

COPY --from=builder /app/app /app/app

COPY --from=builder /app/config.json /app/config.json

# Command to run the binary
CMD /bin/sh -c './app run -c ./config.json'
