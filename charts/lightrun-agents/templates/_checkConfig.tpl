{{/*
Template for checking configuration

The messages templated here will be combined into a single `fail` call.

Message format:

```
checker:
    MESSAGE
```
*/}}
{{/*
Compile all warnings into a single message, and call fail.

Due to gotpl scoping, we can't make use of `range`, so we have to add action lines.
*/}}
{{- define "checkConfig" -}}
{{- $messages := list -}}
{{/* add templates here */}}

{{- $messages = append $messages (include "checkNamespace" .) -}}
{{- $messages = append $messages (include "checkSecret" .) -}}
{{- $messages = append $messages (include "checkName" .) -}}
{{- $messages = append $messages (include "checkContainerSelector" .) -}}
{{- $messages = append $messages (include "checkDeploymentName" .) -}}
{{- $messages = append $messages (include "checkInitContainerImage" .) -}}
{{- $messages = append $messages (include "checkServerHostname" .) -}}


{{- /* prepare output */}}
{{- $messages = without $messages "" -}}
{{- $message := join "\n" $messages -}}

{{- /* print output */}}
{{- if $message -}}
{{-   printf "\nCONFIGURATION CHECKS:\n%s" $message | fail -}}
{{- end -}}
{{- end -}}


{{- define "checkContainerSelector" -}}
{{- range .Values.javaAgents }}
{{- if not .containerSelector }}
{{- printf "containerSelector Checker:\nError: The 'containerSelector' field is missing in %s java agent object. Please specify a 'containerSelector' paramter.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "checkDeploymentName" -}}
{{- range .Values.javaAgents }}
{{- if not .deploymentName }}
{{- printf "deploymentName Checker:\nError: The 'deploymentName' field is missing in %s java agent object. Please specify a 'deploymentName' paramter.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "checkInitContainerImage" -}}
{{- range .Values.javaAgents }}
{{- if not .initContainer.image }}
{{- printf "initContainerImage Checker:\nError: The 'initContainer.image' field is missing in %s java agent object. Please specify a 'initContainer.image' parameter.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "checkName" -}}
{{- range .Values.javaAgents }}
{{- if not .name }}
{{- printf "Name Checker:\nError: The '.name' field is missing in %s java agent object. Please specify a '.name' parameter.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "checkServerHostname" -}}
{{- range .Values.javaAgents }}
{{- if not .serverHostname }}
{{- printf "serverHostname Checker:\nError: The '.serverHostname' field is missing in %s java agent object. Please specify a '.serverHostname' parameter.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "checkNamespace" -}}
{{- range .Values.javaAgents }}
{{- if not .namespace }}
{{- printf "Namespace Checker:\nError: The 'namespace' field is missing in %s java agent object. Please specify a namespace.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "checkSecret" -}}
{{- range .Values.javaAgents }}
{{- if and .agentPoolCredentials.existingSecret .agentPoolCredentials.apiKey .agentPoolCredentials.pinnedCertHash }}
{{- printf "Secret Checker:\nError: both '.agentPoolCredentials.existingSecret' field and '.agentPoolCredentials.apiKey' '.agentPoolCredentials.pinnedCertHash' are provided in %s java agent object. Please choose either existingSecret or apiKey and pinnedCertHash.\n" .name }}
{{- end }}
{{- end }}
{{- end -}}


