# topologytopod

## Description

Copy the Node's topology labels `topology.kubernetes.io/*` to the Pod.

```log
+-------------------+    +-------------------+    +-------------------+
| Kubernetes        | -> | Webhook Server    | -> | Update Pod        |
| APIServer         |    | topologytopod     |    | topology labels   |
+-------------------+    +-------------------+    +-------------------+
```

`topologytopod` is a Kubernetes webhook that copies node topology labels to pods when they are bound to nodes. This ensures that pods have the same topology labels as the nodes they are scheduled on.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+

## Installation

Add the `topologytopod` Helm repository:

```shell
helm repo add topologytopod https://topologytopod.github.io/topologytopod
helm repo update
```

Install the `topologytopod` chart:

```shell
helm install topologytopod topologytopod/topologytopod --wait --debug
```

To customize the installation, use the following values:

```yaml
image:
  repository: ghcr.io/topologytopod/topologytopod
  tag: "v0.0.2"
```
