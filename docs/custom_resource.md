```yaml
apiVersion: agents.lightrun.com/v1beta
kind: LightrunJavaAgent
metadata:
  name: example-cr 
spec:
  # Init container with agent. Differes by agent version and platform that it will be used for. For now supported platforms are `linux` and `alpine`  
  initContainer:  
    # parts that may vary here are 
    # platform - `linux/alpine`
    # agent version - first part of the tag (1.7.0)
    # init container sub-version - last part of the tag (init.0)
    image: "lightruncom/k8s-operator-init-java-agent-linux:1.7.0-init.0"
    # imagePullPolicy of the init container. Can be one of: Always, IfNotPresent, or Never.
    imagePullPolicy: "IfNotPresent"
    # Volume name in case you have some convention in the names
    sharedVolumeName: lightrun-agent-init
    # Mount path where volume will be parked. Various distributions may have it's limitations.
    # For example you can't mount volumes to any path except `/tmp` when using AWS Fargate
    sharedVolumeMountPath: "/lightrun"
  # Name of the workload that you are going to patch.
  # Has to be in the same namespace
  workloadName: app
  # Type of the workload that you are going to patch.
  # Has to be one of `Deployment` or `StatefulSet`
  workloadType: Deployment
  # deprecated, use workloadName and workloadType instead
  deploymentName: app  
  # Name of the secret where agent will take `lightrun_key` and `pinned_cert_hash` from
  # Has to be in the same namespace
  secretName: lightrun-secrets 
  # Hostname of the server. Will be different for on-prem ans single-tenant installations
  # For saas it will be app.lightrun.com
  serverHostname: <lightrun_server>  
  # Env var that will be patched with agent path.
  # If your application not using any, recommended option is to use "JAVA_TOOL_OPTIONS"
  # Also may be "_JAVA_OPTIONS", "JAVA_OPTS"
  # There are also different variations if using Maven -  "MAVEN_OPTS", and so on
  # You can find more info here: https://docs.lightrun.com/jvm/agent/
  agentEnvVarName: JAVA_TOOL_OPTIONS
  # Agent config will override  default configuration with provided values
  # You can find list of available options here https://docs.lightrun.com/jvm/agent-configuration/
  agentConfig:
    max_log_cpu_cost: "2"
  # Tags that agent will be using. You'll see them in the UI and in the IDE plugin as well
  agentTags:
    - operator
  # Agent name. If not provided, pod name will be used
  #agentName: "operator-test-agent"
  # List of container names inside the pod of the deployment
  # If container not mentioned here it will be not patched
  containerSelector:
    - app
  # useSecretsAsMountedFiles determines whether to use secret values as environment variables (false) or as mounted files (true)
  # Default is false for backward compatibility
  useSecretsAsMountedFiles: false
---
apiVersion: v1
metadata: 
  name: lightrun-secrets
stringData:
  # Lightrun key you can take from the server UI at the "setup agent" step
  lightrun_key: <lightrun_key_from_ui>
  # Server certificate hash. It is ensuring that agent is connected to the right Lightrun server
  pinned_cert_hash: <pinned_cert_hash>
kind: Secret
type: Opaque
```
