### How it works

 - User creating custom resource of `Kind: LightrunJavaAgent`  
  [Example](../config/samples/agents_v1beta_lightrunjavaagent.yaml)  
  [Detailed explanation of CR fields](custom_resource.md)
 - Controller is receiving all updates about all CRs(custom resources)`LightrunJavaAgent` across all the cluster (or specific namespaces). Every event triggering reconcile loop of the controller. You can find logic of this loop [here](reconcile_loop.excalidraw.png)  
 - When triggered it is doing several actions:
   - Check if it has access to deployment
   - Fetch data from the specified in CR secret
   - Create config map with agent config from CR data
   - Patching deployment:
     - insert init container
     - add volume
     - map that volume to specified container
     - add/update specified ENV variable in order to let Java know where agent is placed
 - After deployment is being patched, k8s will `recreate all the pods` in the deployment. New Pods will be using Lightrun agent from their start
 - If user deleting `LightrunJavaAgent` CR, controller will rollback all the changes to deployment. This will trigger `recreation of all pods` again
 - [High level diagram](resource_relations.excalidraw.png) of resources created/edited by the operator