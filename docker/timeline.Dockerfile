FROM golang:1.12-stretch

# Create non-root user
RUN useradd -ms /bin/bash nets

WORKDIR /app

RUN chown nets /app

# Copy source
COPY . .

WORKDIR /app/timeline

# Build server
RUN go build -o timeline -mod vendor

# Become non-root user
USER nets

CMD ["./timeline"]
