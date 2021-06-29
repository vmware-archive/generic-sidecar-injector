# generic-sidecar-injector 
[![License][license-img]][license] [![Go Report Card][go-report-img]][go-report] [![Build Status][build-img]][build]


## Overview
When working with Kubernetes, engineers often need to add a sidecar container to a pod for various reasons, scrape metrics or logs at scale, debug network issues or apply networking configurations, without having to add complexity, additional logic and configuration to the application.

For a Kubernetes cluster with several services, it becomes a hassle having to add the sidecar container configuration to all services. In addition, it also makes the manifest file harder to read by a human.

To solve this problem, we introduce a generic sidecar injector, where it injects a sidecar container to a pod based on a specific annotation added to the service manifest.

## Try it out

### Prerequisites

* Kubernetes 1.10+
* kubectl v1.14+
* jq
* openssl or cfssl

### Install

1. Add your Kubernetes cluster's CA bundle value to the mutating webhook configuration file:
    ```shell script
    ./scripts/apply-ca-bundle.sh --context <k8s-context>
    ```

2. Create a signed certificate and store it in a Kubernetes `secret` that will be consumed by the generic-sidecar deployment:
    ```shell script
    ./scripts/create-signed-cert.sh  --context <k8s-context> [--cfssl]
    ```

3. Deploy the sidecar configuration (e.g. Telegraf):
    ```shell script
    kubectl apply -f examples/telegraf
    ```
   
4. Deploy the generic-sidecar-injector:
    ```shell script
    kubectl apply -f kubernetes/
    ```

## Build
Build and push a docker image:

```shell script
docker build -t generic-sidecar-injector:latest -f Dockerfile .
docker push generic-sidecar-injector:latest
```

## Documentation

## Roadmap


## Contributing

The generic-sidecar-injector project team welcomes contributions from the community. Before you start working with generic-sidecar-injector, please
read our [Developer Certificate of Origin](https://cla.vmware.com/dco). All contributions to this repository must be
signed as described on that page. Your signature certifies that you wrote the patch or have the right to pass it on
as an open-source patch. For more detailed information, refer to [CONTRIBUTING.md](CONTRIBUTING.md).



[go-report-img]: https://goreportcard.com/badge/github.com/vmware/generic-sidecar-injector
[go-report]: https://goreportcard.com/report/github.com/vmware/generic-sidecar-injector
[license-img]: https://img.shields.io/badge/License-Apache%202.0-blue.svg
[license]: https://opensource.org/licenses/Apache-2.0
[build-img]: https://travis-ci.com/vmware/generic-sidecar-injector.svg?token=1c5PBgizz4smpAF14oTf&branch=master
[build]: https://travis-ci.com/vmware/generic-sidecar-injector