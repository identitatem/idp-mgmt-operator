
# Copyright Red Hat

FROM registry.ci.openshift.org/stolostron/builder:go1.17-linux AS builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY main.go main.go
COPY main_test.go main_test.go
COPY cmd/ cmd/
COPY deploy/ deploy/
COPY resources/ resources/
COPY webhook/ webhook/
COPY pkg/ pkg/
COPY config/ config/
COPY controllers/ controllers/

RUN GOFLAGS="" go build -a -o idp-mgmt main.go

RUN GOFLAGS="" go test -covermode=atomic \
    -coverpkg=github.com/identitatem/idp-mgmt-operator/controllers/...,\
github.com/identitatem/idp-mgmt-operator/pkg/... \
    -c -tags testrunmain \
    github.com/identitatem/idp-mgmt-operator \
    -o idp-mgmt-coverage

COPY build/bin/ build/bin/

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
RUN microdnf update

ENV OPERATOR=/usr/local/bin/idp-mgmt \
    USER_UID=1001 \
    USER_NAME=idp-mgmt-operator

COPY --from=builder /workspace/idp-mgmt ${OPERATOR}
COPY --from=builder /workspace/idp-mgmt-coverage ${OPERATOR}-coverage
COPY --from=builder /workspace/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
