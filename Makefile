.PHONY: build
build:
	operator-sdk build eks-ebs-node-operator

.PHONY: run
run:
	OPERATOR_NAME=eks-ebs-node-operator WATCH_NAMESPACE="" operator-sdk run --local
