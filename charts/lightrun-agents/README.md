# Helm Chart for Deploying Lightrun Agents

This Helm chart enables the deployment and management of Lightrun Agents as custom resources within your Kubernetes cluster. Currently, only Java-based agents are supported. The LightrunJavaAgent custom resource will be configured according to the settings specified in the values.yaml file.

## Prerequisites

- Kubernetes 1.19+
- Ability to fetch images of the init containers from [Lightrun Repository Dockerhub](https://hub.docker.com/u/lightruncom). or alternatively have them available in private registry.

## Installation

### 1 - Add the repo to your Helm repository list

```shell
helm repo add lightrun-k8s-operator https://lightrun-platform.github.io/lightrun-k8s-operator

```

### 2 - Prepare values.yaml

The values.yaml file includes the following configurable parameters for each Java agent object:

| Parameter                                          | Description                                                                                                                                                                                                                                     | Default                                                         |
| -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `javaAgents[].agentCliFlags`                       | [Command-line flags for the Lightrun Java Agent.](https://docs.lightrun.com/jvm/agent-configuration/#additional-command-line-flags).                                                                                                            | Optional `""` (empty string)                                    |
| `javaAgents[].agentConfig`                         | [Additional configuration for the Lightrun Java Agent.](https://docs.lightrun.com/jvm/agent-configuration/#agent-flags).                                                                                                                        | Optional `{}` (empty map)                                       |
| `javaAgents[].agentEnvVarName`                     | Specifies the Java environment variable name used to add `--agentpath`.                                                                                                                                                                         | Optional (if not provided, defaults to `"JAVA_TOOL_OPTIONS"`)   |
| `javaAgents[].agentName`                           | Custom name to assign to the Lightrun Java Agent.                                                                                                                                                                                               | Optional (if not provided, defaults to pod name)                |
| `javaAgents[].agentPoolCredentials.existingSecret` | Name of an existing Kubernetes secret that contains the API key and pinned certificate hash for the agent pool. [secret example](https://github.com/lightrun-platform/lightrun-k8s-operator/blob/main/examples/lightrunjavaagent.yaml#L64-L73). | Optional (if not provided, defaults to `name-secret`)           |
| `javaAgents[].agentPoolCredentials.apiKey`         | Lightrun agent API key.                                                                                                                                                                                                                         | Required if `existingSecret` not set                            |
| `javaAgents[].agentPoolCredentials.pinnedCertHash` | 64 character sha256 certificate public key hash for pinning.                                                                                                                                                                                    | Required if `existingSecret` not set                            |
| `javaAgents[].agentTags`                           | [List of Lightrun Java Agent tags](https://docs.lightrun.com/jvm/tagging/#manage-lightrun-java-agent-tags).                                                                                                                                     | Optional `[]` (empty list)                                      |
| `javaAgents[].containerSelector`                   | Selector for containers within the deployment to inject the Lightrun Java Agent.                                                                                                                                                                | Required                                                        |
| `javaAgents[].deploymentName`                      | Name of the Kubernetes deployment to attach the Lightrun Java Agent.                                                                                                                                                                            | Required                                                        |
| `javaAgents[].initContainer.image`                 | Image for the Lightrun Java Agent init container.                                                                                                                                                                                               | Required                                                        |
| `javaAgents[].initContainer.imagePullPolicy` | Image pull policy for the init container. Can be one of: Always, IfNotPresent, or Never. | Optional (if not provided, defaults to `"IfNotPresent"`) |
| `javaAgents[].initContainer.sharedVolumeMountPath` | Mount path for the shared volume in the init container.                                                                                                                                                                                         | Optional (if not provided, defaults to `"/lightrun"`"           |
| `javaAgents[].initContainer.sharedVolumeName`      | Name of the shared volume for the init container.                                                                                                                                                                                               | Optional (if not provided, defaults to `"lightrun-agent-init"`" |
| `javaAgents[].name`                                | Name of the Lightrun Java Agent custom resource.                                                                                                                                                                                                | Required                                                        |
| `javaAgents[].namespace`                           | Namespace of the Lightrun Java Agent custom resource. Must be in the same namespace as the workload                                                                                                                                             | Required                                                        |
| `javaAgents[].serverHostname`                      | Hostname of the Lightrun server to connect the agent.                                                                                                                                                                                           | Required                                                        |

#### 2.1 - Set `initContainer.image`

Based on your workload's OS and architecture, you should select the appropriate DockerHub repository from the following options:

- [linux amd64](https://hub.docker.com/repository/docker/lightruncom/k8s-operator-init-java-agent-linux/general)
- [linux arm64](https://hub.docker.com/repository/docker/lightruncom/k8s-operator-init-java-agent-linux-arm64/general)
- [alpine amd64](https://hub.docker.com/repository/docker/lightruncom/k8s-operator-init-java-agent-alpine/general)
- [alpine arm64](https://hub.docker.com/repository/docker/lightruncom/k8s-operator-init-java-agent-alpine-arm64/general)

After determining the appropriate image, you will need to choose a tag. The tag can either be "latest," which corresponds to the most up-to-date Lightrun version, or it can be a specific Lightrun version following the convention `<x.y.z>-init.<number>`. Typically, the `<number>` part is 0, but it is always good to verify on the DockerHub repository.

For your convenience, here are some possible combinations of how the final image might look:

```text
Linux amd64 with the latest version -> lightruncom/k8s-operator-init-java-agent-linux:latest
Linux amd64 with a specific version -> lightruncom/k8s-operator-init-java-agent-linux:1.39.1-init.0
Linux arm64 with the latest version -> lightruncom/k8s-operator-init-java-agent-linux-arm64:latest
Linux arm64 with a specific version -> lightruncom/k8s-operator-init-java-agent-linux-arm64:1.39.1-init.0
Alpine amd64 with the latest version -> lightruncom/k8s-operator-init-java-agent-alpine:latest
Alpine amd64 with a specific version -> lightruncom/k8s-operator-init-java-agent-alpine:1.39.1-init.0
Alpine arm64 with the latest version -> lightruncom/k8s-operator-init-java-agent-alpine-arm64:latest
Alpine arm64 with a specific version -> lightruncom/k8s-operator-init-java-agent-alpine-arm64:1.39.1-init.0
```

#### 2.2 Install the chart

When installing the chart, it is important to understand that the -n flag provided in the helm install command does not determine where the actual resources will be deployed. Instead, deployment is controlled by the javaAgents[].namespace parameter for each object in the values.yaml file.

Use the -n flag to specify a namespace, either using the same namespace where your Lightrun Kubernetes Operator is installed or creating a new namespace specifically for this purpose, such as "lightrun-agents". This namespace will be referenced if you need to uninstall the chart later.

```bash
helm install <release-name> lightrun-k8s-operator/lightrun-agents -n <namespace> -f values.yaml
```

## Examples

### Basic

- The `my-service-1` does not use an `existingSecret` and instead the `agentPoolCredentials.apiKey` and `agentPoolCredentials.pinnedCertHash` are provided directly.

- The `my-service-2` uses an `existingSecret` named `my-existing-secret`

```yaml
javaAgents:
  - name: 'my-service-1'
    namespace: 'my-namespace-1'
    deploymentName: "my-deployment-1"
    containerSelector:
      - my-container-1
    serverHostname: 'lightrun.example.com'
    initContainer:
      image: "lightruncom/k8s-operator-init-java-agent-linux:latest"
      imagePullPolicy: "IfNotPresent"      
    agentPoolCredentials:
      existingSecret: ""
      apiKey: "xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      pinnedCertHash: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    agentTags:
      - env-production
      - service-my-server
      - region-us_east_1
      - provider-aws
  - name: 'my-service-2'
    namespace: 'my-namespace-2'
    initContainer:
      image: "lightruncom/k8s-operator-init-java-agent-linux:latest"
      imagePullPolicy: "IfNotPresent"
    deploymentName: "my-deployment-2"
    containerSelector:
      - my-container-2
    serverHostname: 'lightrun.example.com'
    agentPoolCredentials:
      existingSecret: "my-existing-secret"
      apiKey: ""
      pinnedCertHash: ""
    agentTags:
      - env-production
      - service-my-other-server
      - region-us_east_1
      - provider-aws
```

### Full

- The `my-service-1` does not use an `existingSecret` and instead the `agentPoolCredentials.apiKey` and `agentPoolCredentials.pinnedCertHash` are provided directly.

- The `my-service-2` uses an `existingSecret` named `my-existing-secret`

```yaml
javaAgents:
  - name: 'my-service-1'
    namespace: 'my-namespace-1'
    deploymentName: "my-deployment-1"
    containerSelector:
      - my-container-1
    serverHostname: 'lightrun.example.com'
    agentEnvVarName: '_JAVA_OPTIONS'
    agentConfig:
      max_log_cpu_cost: "2"
    agentCliFlags: "--lightrun_extra_class_path=<PATH_TO_JAR>:<PATH_TO_JAR>,lightrun_init_wait_time_ms"
    initContainer:
      image: "lightruncom/k8s-operator-init-java-agent-linux:latest"
      imagePullPolicy: "IfNotPresent"
      sharedVolumeName: 'my-shared-volume'
      sharedVolumeMountPath: '/mypath'
    agentPoolCredentials:
      existingSecret: ""
      apiKey: "xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
      pinnedCertHash: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    agentTags:
      - env-production
      - service-my-server
      - region-us_east_1
      - provider-aws
  - name: 'my-service-2'
    namespace: 'my-namespace-2'
    initContainer:
      image: "lightruncom/k8s-operator-init-java-agent-linux:latest"
      imagePullPolicy: "IfNotPresent"
      sharedVolumeName: 'my-shared-volume'
      sharedVolumeMountPath: '/mypath'
    deploymentName: "my-deployment-2"
    containerSelector:
      - my-container-2
    serverHostname: 'lightrun.example.com'
    agentEnvVarName: 'JAVA_OPTS'
    agentConfig:
      max_log_cpu_cost: "2"
    agentCliFlags: "--lightrun_extra_class_path=<PATH_TO_JAR>:<PATH_TO_JAR>,lightrun_init_wait_time_ms"
    agentPoolCredentials:
      existingSecret: "my-existing-secret"
      apiKey: ""
      pinnedCertHash: ""
    agentTags:
      - env-production
      - service-my-other-server
      - region-us_east_1
      - provider-aws
```

## Uninstallation

To uninstall the chart:

```bash
helm uninstall <release-name> -n <namespace>
```

This command removes all the Kubernetes components associated with the chart and deletes the release.
