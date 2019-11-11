FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY bin/manager /manager
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
