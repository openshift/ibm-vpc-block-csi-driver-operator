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

Follow below steps to add node labels
```shell
# Get worker id 
curl -X GET "https://<region>.iaas.cloud.ibm.com/v1/instances?version=2021-11-23&generation=2&name=<node-name>" -H "Authorization: $iam_token"
# Add node labels 
kubectl label nodes <node-name>  "ibm-cloud.kubernetes.io/worker-id"=<worker-id>

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
export RESIZER_IMAGE=k8s.gcr.io/sig-storage/csi-resizer:v1.2.0
export NODE_LABEL_IMAGE=icr.io/obs/storage/vpc-node-label-updater:v1.0.1

# Run the operator via CLI
./ibm-vpc-block-csi-driver-operator start --kubeconfig $MY_KUBECONFIG --namespace openshift-cluster-csi-drivers
```
