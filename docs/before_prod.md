### Important to know before deploying to production  

  - `LightrunJavaAgent` Customer resource hardly dependent on the secret with `lightrun_key` and `pinned_cert_hash` values. It has do be deployed in the same namespace as the secret.
  - `LightrunJavaAgent` CR has to be installed in the same namespace as deployment
  - You need to create `LightrunJavaAgent` CR per deployment that you want to patch
  - When `creating or deleting CR`, deployment will trigger `recreation of all the pods`, as Pod Template Spec will be changed
  - If, for some reason, your cluster will not be able to `download init container` images from https://hub.docker.com/, your deployment will stuck in this state until it won't be resolved. This is the limitation of the init containers
  - If you will change `secret` values, `agentConfig` or `agentTags`, operator will update Config Map with that data and trigger recreation of the pods to apply new config of the agent
  - Always check `release notes` before upgrading the operator. If CRD fields was changed you'll need to act accordingly during the upgrade 
  - You can't have `duplicate ENV` variable in the container spec. 
  - If you are using `gitops` tools, you'll have to tell them to ignore ENV var of the patched container. Otherwise it will try to default it as per your deployment yaml. Other things that are changed by operator are handled with help of `managedFields`. You can read about it [here](https://kubernetes.io/docs/reference/using-api/server-side-apply/)  
  Example for [Argo CD](https://argo-cd.readthedocs.io/en/stable/user-guide/diffing/)
  ```yaml
      ignoreDifferences:
      - group: apps
        kind: Deployment
        name: <Your deployment name>
        jqPathExpressions:
        - '.spec.template.spec.containers[] | select(.name == "<your container name>").env[] | select(.name == "JAVA_TOOL_OPTIONS")' 
  ```