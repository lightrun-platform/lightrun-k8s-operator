### How it works

 - The User begins by creating a custom resource of `Kind: LightrunJavaAgent`  
  [Example](../config/samples/agents_v1beta_lightrunjavaagent.yaml)  
  [Detailed explanation of CR fields](custom_resource.md)
 - The Controller receives all updates about all CRs (custom resources) of kind `LightrunJavaAgent` across all the cluster or specific namespaces
   (subject to how it's been installed). 
 Every event related to these CRs triggers the reconcile loop of the controller. You can find logic of this loop [here](reconcile_loop.excalidraw.png)  
 - When triggered, the controller performs several actions:
   - Check if it has access to deployment
   - Fetch data from the CR secret
   - Create config map with agent config from CR data
   - Patch the deployment:
     - insert init container
     - add volume
     - map that volume to the specified container
     - add/update specified ENV variable in order to let Java know where agent files are found (the mapped volume)
 - After deployment is patched, k8s will `recreate all the pods` in the deployment. New Pods will be initialized with the Lightrun agent
 - If user deletes the `LightrunJavaAgent` CR, the Controller will roll back all the changes to deployment. This will trigger `recreation of all pods` again
 - [High level diagram](resource_relations.excalidraw.png) of resources created/edited by the operator