# Production image based on alpine.
FROM alpine
LABEL maintainer="Axiom, Inc. <info@axiom.co>"

# Upgrade packages and install ca-certificates.
RUN apk update --no-cache && \
    apk upgrade --no-cache && \
    apk add --no-cache ca-certificates

# Copy binary into image.
COPY axiom-syslog-proxy /usr/bin/axiom-syslog-proxy

# Use the project name as working directory.
WORKDIR /axiom-syslog-proxy

# Expose the default application port.
EXPOSE 3101/tcp

# Set the binary as entrypoint.
ENTRYPOINT [ "/usr/bin/axiom-syslog-proxy" ]
