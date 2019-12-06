FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

### Required OpenShift Certified Labels
LABEL name="Helm Controller" \
      vendor="Kubedex" \
      version="v1.2.2" \
      release="1.2.2" \
      summary="Kubernetes Operator for Helm." \
      description="A simple way to manage helm charts with a Custom Resource Definitions in k8s. (https://kubedex.com)"

# Required to be in /licenses
RUN  mkdir /licenses
COPY LICENSE /licenses

ENV OPERATOR=/usr/local/bin/helm-controller \
    USER_UID=1001 \
    USER_NAME=helm-controller

# install operator binary
COPY build/_output/bin/helm-controller ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
