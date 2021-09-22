# ibm-vpc-block-csi-driver-operator

An operator to deploy the [IBM VPC Block CSI Driver](https://github.com/IBM/ibm-vpc-block-csi-driver).

This operator is installed by the [cluster-storage-operator](https://github.com/openshift/cluster-storage-operator).

# Quick start

Before running the operator manually, you must remove the operator installed by CVO

```shell
# Scale down CVO
oc scale -n openshift-cluster-version  deployment/cluster-version-operator --replicas=0

# Delete operator resources (daemonset, deployments)
oc -n openshift-cluster-csi-drivers delete deployment.apps/ibm-vpc-block-csi-driver-operator deployment.apps/ibm-vpc-block-csi-controller daemonset.apps/ibm-vpc-block-csi-node
```

To build and run the operator locally:

```shell
# Create only the resources the operator needs to run via CLI
oc apply -f manifests/00_crd.yaml
oc apply -f manifests/01_namespace.yaml
oc apply -f manifests/09_cr.yaml

# Build the operator
make all

# Set the environment variables
export DRIVER_IMAGE=icr.io/ibm/ibm-vpc-block-csi-driver:v3.0.0
export PROVISIONER_IMAGE=quay.io/k8scsi/csi-provisioner:v1.6.0
export ATTACHER_IMAGE=quay.io/k8scsi/csi-attacher:v2.2.0
export NODE_DRIVER_REGISTRAR_IMAGE=quay.io/k8scsi/csi-node-driver-registrar:v1.2.0
export LIVENESS_PROBE_IMAGE=quay.io/k8scsi/livenessprobe:v2.0.0

# Run the operator via CLI
./ibm-vpc-block-csi-driver-operator start --kubeconfig $MY_KUBECONFIG --namespace openshift-cluster-csi-drivers
```

To run the latest build of the operator:
```shell
# Set the environment variables
export DRIVER_IMAGE=icr.io/ibm/ibm-vpc-block-csi-driver:v3.0.0
export PROVISIONER_IMAGE=quay.io/k8scsi/csi-provisioner:v1.6.0
export ATTACHER_IMAGE=quay.io/k8scsi/csi-attacher:v2.2.0
export NODE_DRIVER_REGISTRAR_IMAGE=quay.io/k8scsi/csi-node-driver-registrar:v1.2.0
export LIVENESS_PROBE_IMAGE=quay.io/k8scsi/livenessprobe:v2.0.0

# Deploy the operator and everything it needs
oc apply -f manifests/
```
