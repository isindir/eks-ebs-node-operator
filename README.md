[![Go Report Card](https://goreportcard.com/badge/github.com/isindir/eks-ebs-node-operator?)](https://goreportcard.com/report/github.com/isindir/eks-ebs-node-operator)
[![CircleCI](https://circleci.com/gh/isindir/eks-ebs-node-operator.svg?style=svg)](https://circleci.com/gh/isindir/eks-ebs-node-operator)
[![GitHub release](https://img.shields.io/github/tag/isindir/eks-ebs-node-operator.svg)](https://github.com/isindir/eks-ebs-node-operator/releases)
[![Docker pulls](https://img.shields.io/docker/pulls/isindir/eks-ebs-node-operator.svg)](https://hub.docker.com/r/isindir/eks-ebs-node-operator)
[![MPL v2.0](http://img.shields.io/github/license/isindir/eks-ebs-node-operator.svg)](LICENSE)

# EKS ebs node operator

Operator adds custom resource limit to the AWS EKS worker nodes, calculated from
node type and some AWS imposed limits. At the time of writing there is a
configuration mismatch between Kubernetes EBS CSI and AWS EC2 instanced imposed
limitations, which in some cases leads to pods with ebs volumes being scheduled
on a node, where it is impossible to attach EBS volume anymore.

When an operator is deployed in the cluster it will automatically add extra
custom resource limit, but pods needs to consume this resource via `resources`,
like CPU or Memory. Kubernetes will automatically calculate the amount of EBS
volume attachments left on a node and will not schedule pods with EBS volumes if
the resource is exhausted.
