FROM alpine:3.10
WORKDIR /
COPY ./bin/manager /manager

ENTRYPOINT ["/manager"]
