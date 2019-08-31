FROM nginx:stable

# Copy basic auth info
COPY ./htpasswd /etc/nginx/htpasswd

# Copy specific configuration
COPY ./default.conf /etc/nginx/conf.d/default.conf

# Copy nginx config
COPY ./nginx.conf /etc/nginx/nginx.conf
