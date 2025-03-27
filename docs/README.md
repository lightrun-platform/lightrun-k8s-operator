<p align="center">
    <a href="https://www.lightrun.com/" target="_blank">
      <img src="https://user-images.githubusercontent.com/33126908/135755862-3c2d9143-c9bc-49b6-933c-f80df720d44e.png" alt="Lightrun">
    </a>
</p>

# [Lightrun](lightrun.com) Kubernetes Operator

[![Build Status](https://github.com/lightrun-platform/lightrun-k8s-operator/actions/workflows/release.yaml/badge.svg)](https://github.com/lightrun-platform/lightrun-k8s-operator/actions/workflows/release.yaml/) 
[![Tests](https://github.com/lightrun-platform/lightrun-k8s-operator/actions/workflows/e2e.yaml/badge.svg)](https://github.com/lightrun-platform/lightrun-k8s-operator/actions/workflows/e2e.yaml)

The ***Lightrun Kubernetes(K8s) Operator*** makes it easy to insert Lightrun agents into your K8s workloads without changing your docker or manifest files. The ***Lightrun K8s Operator*** project was initially scaffolded using [operator-sdk](https://sdk.operatorframework.io/) and [kubebuilder book](https://book.kubebuilder.io/), and aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

Table of contents
=================

<!--ts-->
   * [Description](#description)
   * [Example](#example)
   * [Example with Helm Chart](#example-with-helm-chart)
   * [Limitations](#limitations)
   * [Contributing Guide ](#contributing-guide)
   * [Licence](#license)
<!--te-->


## Description

In theory for adding a Lightrun agent to an application running on Kubernetes, you must:
1. Install the agent into the Kubernetes pod.
2. Notify the running application to start using the installed agent.

The ***Lightrun K8s operator*** does those steps for you. [details](https://github.com/lightrun-platform/lightrun-k8s-operator/blob/main/docs/how.md)

> Important - [Read this before deploying to production](https://github.com/lightrun-platform/lightrun-k8s-operator/blob/main/docs/before_prod.md).

### Requirements
- Kubernetes >= 1.19

### Example

To set up the Lightrun K8s operator:

1. Create namespace for the operator and test  deployment
```sh
kubectl create namespace lightrun-operator
kubectl create namespace lightrun-agent-test
```
_`lightrun-operator` namespace is hardcoded in the example `operator.yaml` due to Role and RoleBinding objects_
_If you want to deploy operator to a different namespace - you can use helm chart_

2. Deploy operator to the operator namesapce
```sh
kubectl apply -f https://raw.githubusercontent.com/lightrun-platform/lightrun-k8s-operator/main/examples/operator.yaml -n lightrun-operator
```  

3. Create simple deployment for test  
> _App source code [PrimeMain.java](../examples/app/PrimeMain.java)_  
```sh
kubectl apply -f https://raw.githubusercontent.com/lightrun-platform/lightrun-k8s-operator/main/examples/deployment.yaml -n lightrun-agent-test
```

4. Download Lightrun agent config 
```sh
curl https://raw.githubusercontent.com/lightrun-platform/lightrun-k8s-operator/main/examples/lightrunjavaagent.yaml > agent.yaml
```

5. Update the following config parameters in the `agent.yaml` file.
  - serverHostname     - for SaaS it is `app.lightrun.com`, for on-prem use your own hostname  

  - lightrun_key       - You can find this value on the set up page, 2nd step  
  ![](setup.png)

  - pinned_cert_hash   - you can fetch it from **https://`<serverHostname>`/api/getPinnedServerCert**  
    > have to be authenticated

6. Create agent custom resource
```sh
kubectl apply -f agent.yaml -n lightrun-agent-test
```

7. Go to the Lightrun server and check if you see new agent registered in the list of the agents  
![](agents.png)  

## Example with Helm Chart

[Helm chart](../charts/lightrun-operator/) is available in repository branch `helm-repo`  
- Add the repo to your Helm repository list
```sh 
helm repo add lightrun-k8s-operator https://lightrun-platform.github.io/lightrun-k8s-operator
```

- Install the Helm chart:   
> _Using default [values](../charts/lightrun-operator/values.yaml)_  
  
```sh
helm install lightrun-k8s-operator/lightrun-k8s-operator  -n lightrun-operator --create-namespace
```  

  > _Using custom values file_

```sh
helm install lightrun-k8s-operator/lightrun-k8s-operator  -f <values file>  -n lightrun-operator --create-namespace
```
> `helm upgrade --install` or `helm install --dry-run` may not work properly due to limitations of how Helm work with CRDs.
You can find more info [here](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/)


- Uninstall the Helm chart.
```sh
helm delete lightrun-k8s-operator
```
> `CRDs` will not be deleted due to Helm CRDs limitations. You can learn more about the limitations [here](https://helm.sh/docs/topics/charts/#limitations-on-crds).

### Chart version vs controller version
For the sake of simplicity, we are keeping the convention of the same version for both the controller image and the Helm chart. This helps to ensure that controller actions are aligned with CRDs preventing failed resource validation errors.

## Limitations

- Operator can only patch environment variable that configured as a key/value pair
  ```
  env:
    - name: JAVA_TOOL_OPTIONS
      value: "some initital value"
  ```
  if value mapped from the configMap or secret using `valueFrom`, operator will fail to update the deployment with the following error:
  ```
  'Deployment.apps "<deployment name>" is invalid: spec.template.spec.containers[0].env[31].valueFrom:
      Invalid value: "": may not be specified when `value` is not empty'
  ```

- If an application has [JDWR](https://en.wikipedia.org/wiki/Java_Debug_Wire_Protocol) enabled, it will cause a conflict with the Lightrun agent installed by the Lightrun K8s operator.
- You must install the correct init container for your application’s container platform. For example, _lightruncom/k8s-operator-init-java-agent-`linux`:1.7.0-init.0_.
    #### Supported Platforms
    - Linux
    - Alpine
  > Available init containers:
  > - [Java agent for linux x86_64](https://hub.docker.com/r/lightruncom/k8s-operator-init-java-agent-linux/tags)
  > - [Java agent for linux arm64 ](https://hub.docker.com/r/lightruncom/k8s-operator-init-java-agent-linux-arm64)
  > - [Java agent for alpine x86_64](https://hub.docker.com/r/lightruncom/k8s-operator-init-java-agent-alpine/tags)
  > - [Java agent for alpine arm64 ](https://hub.docker.com/r/lightruncom/k8s-operator-init-java-agent-alpine-arm64)
- K8s type of resources
    - Deployment
- Application's language
    - Java

## Contributing Guide
If you have any idea for an improvement or find a bug do not hesitate in opening an issue, just simply fork and create a pull-request.
Please open an issue first for any big changes.


> `make post-commit-hook`  
  Run this command to add post commit hook. It will regenerate rules and CRD from the code after every commit, so you'll not forget to do it.
  You'll need to commit those changes as well.

### Test It Out Locally
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) or [K3S](https://k3s.io/) to get a local cluster for testing, or run against a remote cluster.  
**Note:** When using `make` commands, your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).  

1. Clone repo
```sh
git clone git@github.com:lightrun-platform/lightrun-k8s-operator.git
cd lightrun-k8s-operator
```

2. Install the CRDs into the cluster:

```sh
make install
```

3. Run your controller (this will run in the foreground):
```sh
make run
```

4. Open another terminal tab and deploy simple app to your cluster
```sh
kubectl apply -f ./examples/deployment.yaml
kubectl get deployments sample-deployment
```

5. Update `lightrun_key`, `pinned_cert_hash` and `serverHostname` in the [CR example file](../examples/lightrunjavaagent.yaml)  


6. Create LightrunJavaAgent custom resource
```sh
kubectl apply -f ./examples/lightrunjavaagent.yaml
```

At this point you will see in the controller logs that it recognized new resource and started to work.
If you run the following command, you will see that changes done by the controller (init container, volume, patched ENV var).
```sh
kubectl describe deployments sample-deployment
```

## License

Copyright 2022 Lightrun

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
