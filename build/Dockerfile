FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/generic-sidecar-injector \
    USER_UID=1001 \
    USER_NAME=generic-sidecar-injector

# install operator binary
COPY build/_output/bin/generic-sidecar-injector ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
