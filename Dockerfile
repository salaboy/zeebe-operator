FROM alpine:3.10
RUN apk add --update --no-cache ca-certificates git 

WORKDIR /
COPY ./bin/manager /manager

RUN /bin/bash -c 'chmod +x /manager'

CMD ["/manager"]

#CMD ["/manager"]
