# ibm-vpc-block-csi-driver-operator

An operator to deploy the [IBM VPC Block CSI Driver](https://github.com/IBM/ibm-vpc-block-csi-driver).

This operator is installed by the [cluster-storage-operator](https://github.com/openshift/cluster-storage-operator).

# Quick start

Before running the operator manually, you must remove the operator installed by CVO

```shell
# Scale down CVO and CSO
oc scale --replicas=0 deploy/cluster-version-operator -n openshift-cluster-version
oc scale --replicas=0 deploy/cluster-storage-operator -n openshift-cluster-storage-operator

# Delete operator resources (daemonset, deployments)
oc -n openshift-cluster-csi-drivers delete deployment.apps/ibm-vpc-block-csi-driver-operator deployment.apps/ibm-vpc-block-csi-controller daemonset.apps/ibm-vpc-block-csi-node
```

To build and run the operator locally:

```shell
# Create only the resources the operator needs to run via CLI
oc apply -f manifests/09_cr.yaml

# Build the operator
make 

# Set the environment variables
export DRIVER_IMAGE=gcr.io/k8s-staging-cloud-provider-ibm/ibm-vpc-block-csi-driver:master
export PROVISIONER_IMAGE=k8s.gcr.io/sig-storage/csi-provisioner:v2.2.2
export ATTACHER_IMAGE=k8s.gcr.io/sig-storage/csi-attacher:v3.2.1
export NODE_DRIVER_REGISTRAR_IMAGE=k8s.gcr.io/sig-storage/csi-node-driver-registrar:v2.2.0
export LIVENESS_PROBE_IMAGE=k8s.gcr.io/sig-storage/livenessprobe:v2.3.0

# Run the operator via CLI
./ibm-vpc-block-csi-driver-operator start --kubeconfig $MY_KUBECONFIG --namespace openshift-cluster-csi-drivers
```