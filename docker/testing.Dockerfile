FROM golang:1.12-stretch

WORKDIR /app

# Copy source
COPY . .

# Run tests
CMD ./util/tests.sh
