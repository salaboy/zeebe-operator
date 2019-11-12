FROM alpine:3.10
RUN apk add --update --no-cache ca-certificates git 

WORKDIR /
COPY ./bin/manager /manager

ENTRYPOINT ["/manager"]
