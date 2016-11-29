FROM alpine:latest
COPY consul-router /consul-router
ENTRYPOINT ["/consul-router"]
