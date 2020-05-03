.PHONY: build
build:
	operator-sdk build eks-ebs-node-operator

.PHONY: run
run:
	OPERATOR_NAME=eks-ebs-node-operator WATCH_NAMESPACE="" operator-sdk run --local

.PHONY: cluster-create
cluster-create:
	kind create cluster --name operator
	kubectl label nodes operator-control-plane beta.kubernetes.io/instance-type=m5a.2xlarge

.PHONY: cluster-delete
cluster-delete:
	kind delete cluster --name operator
