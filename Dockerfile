FROM golang:1.16.3 as builder
#FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.16-openshift-4.9 as builder // TODO change base image

WORKDIR /go/src/github.com/IBM/ibm-vpc-block-csi-driver-operator
ADD . /go/src/github.com/IBM/ibm-vpc-block-csi-driver-operator

ARG TAG
ARG OS
ARG ARCH

RUN cd /go/src/github.com/IBM/ibm-vpc-block-csi-driver-operator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o ibm-vpc-block-csi-driver-operator cmd/ibm-vpc-block-csi-driver-operator/main.go
#RUN make // TODO

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4-205
COPY --from=builder /go/src/github.com/IBM/ibm-vpc-block-csi-driver-operator/ibm-vpc-block-csi-driver-operator /usr/bin
COPY manifests /manifests
RUN chmod +x /usr/bin/ibm-vpc-block-csi-driver-operator
ENTRYPOINT ["/usr/bin/ibm-vpc-block-csi-driver-operator"]
LABEL io.k8s.display-name="OpenShift IBM VPC Block CSI Driver Operator" \
	io.k8s.description="The IBM VPC Block CSI Driver Operator installs and maintains the IBM VPC Block CSI Driver on a cluster."
