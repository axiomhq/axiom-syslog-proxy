# Production image based on distroless.
FROM gcr.io/distroless/static-debian11:nonroot
LABEL maintainer="Axiom, Inc. <info@axiom.co>"

# Copy binary into image.
COPY --chown=nonroot:nonroot axiom-syslog-proxy /usr/bin/axiom-syslog-proxy

# Use the project name as working directory.
WORKDIR /axiom-syslog-proxy

# Expose the default application port.
EXPOSE 3101/tcp

# Set the binary as entrypoint.
ENTRYPOINT [ "/usr/bin/axiom-syslog-proxy" ]
