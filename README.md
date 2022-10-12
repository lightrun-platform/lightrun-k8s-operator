- Add the repo to your Helm repository list
```sh 
helm repo add lightrun-k8s-operator https://lightrun-platform.github.io/lightrun-k8s-operator
```

- Install the Helm chart:   
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


- Uninstall the Helm chart.
```sh
helm delete lightrun-k8s-operator
```
> `CRDs` will not be deleted due to Helm CRDs limitations. You can learn more about the limitations [here](https://helm.sh/docs/topics/charts/#limitations-on-crds).

### Chart version vs controller version
For the sake of simplicity, we are keeping the convention of the same version for both the controller image and the Helm chart. This helps to ensure that controller actions are aligned with CRDs preventing failed resource validation errors.
