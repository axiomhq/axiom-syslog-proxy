# Production image based on distroless.
FROM gcr.io/distroless/static-debian12
LABEL maintainer="Axiom, Inc. <info@axiom.co>"

# Copy binary into image.
COPY axiom-syslog-proxy /usr/bin/axiom-syslog-proxy

# Use the project name as working directory.
WORKDIR /axiom-syslog-proxy

# Expose the default application ports.
EXPOSE 514/udp 601/tcp

# Set the binary as entrypoint.
ENTRYPOINT [ "/usr/bin/axiom-syslog-proxy" ]
