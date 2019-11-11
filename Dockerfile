FROM amd64/alpine:3.10
WORKDIR /
COPY bin/manager /manager
#USER nonroot:nonroot

ENTRYPOINT ["/manager"]
