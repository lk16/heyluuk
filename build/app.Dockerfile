# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from golang:1.13-alpine base image
FROM golang:1.13-alpine

# The latest alpine images don't have some tools like (`git` and `bash`).
# Adding git, bash and openssh to the image
RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh

# Add Maintainer Info
LABEL maintainer="Luuk Verweij <luuk_verweij@msn.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependancies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

ADD ./cmd /app/cmd
ADD ./internal /app/internal
ADD ./web /app/web

# Build the Go app
ENV CGO_ENABLED 0
RUN go install ./cmd/heyluuk

# Run the executable
CMD ["heyluuk"]