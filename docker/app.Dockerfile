FROM golang:1.12-stretch

# Install dependencies
RUN apt-get update && apt-get install -y python-yaml
COPY ./app/javascript/node /node
RUN bash /node/install_node_repo.sh
RUN apt-get install -y nodejs
RUN npm install uglify-js -g
RUN rm -r /node

# Create non-root user
RUN useradd -ms /bin/bash nets

WORKDIR /app

RUN chown nets /app

# Copy source
COPY . .

# Bust caches for static resources
RUN ./util/cachebuster.sh ./app/template

WORKDIR /app/app

# Build & minify js
RUN python /app/app/javascript/generate.py
RUN cd /app/app/static/script && uglifyjs ./main.js --compress --mangle toplevel \
    --output ./main.min.js

# Build server
RUN go build -o millionnets -mod vendor

# Become non-root user
USER nets

# Start server
CMD ["./millionnets"]
