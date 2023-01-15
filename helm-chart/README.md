# [Lightrun k8s operator](https://github.com/lightrun-platform/lightrun-k8s-operator)

![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

## Operator docs
[Github readme](https://github.com/lightrun-platform/lightrun-k8s-operator/tree/main/docs)

## Requirements

Kubernetes: `>= 1.19.0`

## Dependencies

Custom Resource of the operator is strictly depends on the secret with `lightrun_key` and `pinned_cert_hash` values  
[Example](https://github.com/lightrun-platform/lightrun-k8s-operator/tree/main/examples/lightrunjavaagent.yaml#L56)

## Installation  
- Add the repo to your Helm repository list
```sh 
helm repo add lightrun-k8s-operator https://lightrun-platform.github.io/lightrun-k8s-operator
```

-  Install the Helm chart:   
> _Using default [values](../helm-chart/values.yaml)_  
  
```sh
helm install lightrun-k8s-operator/lightrun-k8s-operator  -n lightrun-operator --create-namespace
```  

  > _Using custom values file_

```sh
helm install lightrun-k8s-operator/lightrun-k8s-operator  -f <values file>  -n lightrun-operator --create-namespace
```
> `helm upgrade --install` or `helm install --dry-run` may not work properly due to limitations of how Helm work with CRDs.
You can find more info [here](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/)


## Uninstall
```sh
helm delete lightrun-k8s-operator
```
> `CRDs` will not be deleted due to Helm CRDs limitations. You can learn more about the limitations [here](https://helm.sh/docs/topics/charts/#limitations-on-crds).

## Chart version vs controller version
For the sake of simplicity, we are keeping the convention of the same version for both the controller image and the Helm chart. This helps to ensure that controller actions are aligned with CRDs preventing failed resource validation errors.


## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| controllerManager.kubeRbacProxy.image.repository | string | `"gcr.io/kubebuilder/kube-rbac-proxy"` |  |
| controllerManager.kubeRbacProxy.image.tag | string | `"v0.11.0"` |  |
| controllerManager.kubeRbacProxy.resources.limits.cpu | string | `"500m"` |  |
| controllerManager.kubeRbacProxy.resources.limits.memory | string | `"128Mi"` |  |
| controllerManager.kubeRbacProxy.resources.requests.cpu | string | `"5m"` |  |
| controllerManager.kubeRbacProxy.resources.requests.memory | string | `"64Mi"` |  |
| controllerManager.manager.image.repository | string | `"lightruncom/lightrun-k8s-operator"` |  |
| controllerManager.manager.image.tag | string | `"latest"` | For simplicity of version compatibilities we are keeping the same controller and chart versions So the most safe approach is to use same version as the Chart. When installing chart from the helm repo, every helm package version will have controller image set to chart version |
| controllerManager.manager.nodeSelector | object | `{}` |  |
| controllerManager.manager.resources.limits.cpu | string | `"500m"` |  |
| controllerManager.manager.resources.limits.memory | string | `"128Mi"` |  |
| controllerManager.manager.resources.requests.cpu | string | `"10m"` |  |
| controllerManager.manager.resources.requests.memory | string | `"64Mi"` |  |
| controllerManager.manager.tolerations | list | `[]` |  |
| controllerManager.replicas | int | `1` |  |
| managerConfig.controllerManagerConfigYaml.health.healthProbeBindAddress | string | `":8081"` |  |
| managerConfig.controllerManagerConfigYaml.leaderElection.leaderElect | bool | `true` |  |
| managerConfig.controllerManagerConfigYaml.leaderElection.resourceName | string | `"5b425f09.lightrun.com"` |  |
| managerConfig.controllerManagerConfigYaml.metrics.bindAddress | string | `"127.0.0.1:8080"` |  |
| managerConfig.controllerManagerConfigYaml.webhook.port | int | `9443` |  |
| managerConfig.logLevel | string | `"info"` | Log level: 1 - 5 Higher number - more logs Documentation of logr module https://pkg.go.dev/github.com/go-logr/logr@v1.2.0#hdr-Verbosity On level info (0) (default) you'll see only deployments that are being added or deleted and errors On level 1 you'll see 1 additional log per every successful reconciliation loop run On level 2 you'll see all debug prints with intermediate steps while patching deployment per every reconciliation loop run |
| managerConfig.operatorScope | object | `{"namespacedScope":false,"namespaces":["default"]}` | Operator may work in 2 scopes: cluster and namespaced Cluster scope will give permissions to operator to watch and patch deployment in the whole cluster With namespaced scope you need to provide list of namespaces that operator will be able to watch. Namespaced scope implemented by both controller code and creation of the appropriate Roles by the chart Any change to the list of namespaces will cause restart of the operator controller pod. |
| metricsService | object | `{"ports":[{"name":"https","port":8443,"protocol":"TCP","targetPort":8443}],"type":"ClusterIP"}` | Metrics service for prometheus compatible poller |
| nameOverride | string | `"lightrun-k8s-operator"` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.0](https://github.com/norwoodj/helm-docs/releases/v1.11.0)
