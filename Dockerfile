# Stage 1:
# Use base Alpine image to prepare our binary, label it 'app'
FROM alpine:3.8 as app
# Add user and group so that the Docker process in Scratch doesn't run as root
RUN addgroup -S portpusher \
 && adduser -S -u 10000 -g portpusher portpusher

# Stage 2:
# Use the Docker Scratch image to copy our previous stage into
# FROM scratch
FROM alpine:3.8
ARG TARGETARCH
# Grab necessary certificates as Scratch has none
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy our binary to the root of the Scratch image (note: --from=app, the name we gave our first stage)
COPY build/portpusher-${TARGETARCH} /app/portpusher
# Copy the user that we created in the first stage so that we don't run the process as root
COPY --from=app /etc/passwd /etc/passwd
# Change to the non-root user
USER portpusher
# Run our app by directly executing the binary
ENTRYPOINT ["/app/portpusher"]