{{/*
Compile all warnings into a single message, and call fail.
*/}}

{{- define "javaAgents.checkConfig" -}}
{{- $objectErrors := dict -}}  {{/* Create a dictionary to store errors by agent name */}}

{{- range .Values.javaAgents }}
  {{- $objectName := .name }}
  {{- $objectErrorMsgs := list -}}  {{/* Create a list to store errors for the current agent */}}

  {{- /* Add error messages to the list if fields are missing */}}
  {{- if not .namespace }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Namespace Checker:\n  Error: The 'namespace' field is missing. Please provide the 'namespace' parameter." -}}
  {{- end }}
  {{- if not .serverHostname }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Server Hostname Checker:\n  Error: The 'serverHostname' field is missing. Please provide the 'serverHostname' parameter." -}}
  {{- end }}
  {{- if not .name }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Name Checker:\n Error: The 'name' field is missing. Please provide the 'name' parameter." -}}
  {{- end }}
  {{- if not .initContainer.image }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Init Container Image Checker:\n Error: The 'initContainer.image' field is missing. Please provide the 'initContainer.image' parameter." -}}
  {{- end }}

  {{- /* Workload configuration validation */}}
  {{- $hasDeploymentName := .deploymentName }}
  {{- $hasWorkloadConfig := and .workloadName .workloadType }}
  
  {{- if and $hasDeploymentName $hasWorkloadConfig }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Workload Configuration Checker:\n  Error: Both 'deploymentName' (legacy) and 'workloadName'/'workloadType' (new) are specified. Please use only one configuration method: either 'deploymentName' OR 'workloadName' with 'workloadType'." -}}
  {{- else if not (or $hasDeploymentName $hasWorkloadConfig) }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Workload Configuration Checker:\n  Error: No workload configuration specified. Please provide either 'deploymentName' (legacy) OR 'workloadName' with 'workloadType' (recommended)." -}}
  {{- end }}

  {{- /* Validate workloadType if workloadName is provided */}}
  {{- if .workloadName }}
    {{- if not .workloadType }}
      {{- $objectErrorMsgs = append $objectErrorMsgs "Workload Type Checker:\n  Error: 'workloadName' is specified but 'workloadType' is missing. Please provide 'workloadType' (either 'Deployment' or 'StatefulSet')." -}}
    {{- else if not (or (eq .workloadType "Deployment") (eq .workloadType "StatefulSet")) }}
      {{- $objectErrorMsgs = append $objectErrorMsgs "Workload Type Checker:\n  Error: Invalid 'workloadType' value. Must be either 'Deployment' or 'StatefulSet'." -}}
    {{- end }}
  {{- end }}

  {{- if not .containerSelector }}
    {{- $objectErrorMsgs = append $objectErrorMsgs "Container Selector Checker:\n Error: The 'containerSelector' field is missing. Please provide the 'containerSelector' parameter." -}}
  {{- end }}

  {{- if .agentPoolCredentials.existingSecret }}
    {{- if and .agentPoolCredentials.apiKey .agentPoolCredentials.pinnedCertHash }}
      {{- $objectErrorMsgs = append $objectErrorMsgs "Secret Checker:\n Error: Both 'agentPoolCredentials.existingSecret' and 'agentPoolCredentials.apiKey' with 'agentPoolCredentials.pinnedCertHash' are defined. Please use only one of the following: 'existingSecret' or 'apiKey' with 'pinnedCertHash'." -}}
    {{- end }}
  {{- end }}

  {{- if not .agentPoolCredentials.existingSecret }}
    {{- if not (and .agentPoolCredentials.apiKey .agentPoolCredentials.pinnedCertHash) }}
      {{- $objectErrorMsgs = append $objectErrorMsgs "Secret Checker:\n Error: neither 'agentPoolCredentials.existingSecret' nor 'agentPoolCredentials.apiKey' with 'agentPoolCredentials.pinnedCertHash' are defined. Please use one of the following: 'existingSecret' or 'apiKey' with 'pinnedCertHash'." -}}
    {{- end }}
  {{- end }}

  {{- if $objectErrorMsgs }}
    {{- $objectErrors = merge $objectErrors (dict $objectName $objectErrorMsgs) -}}
  {{- end }}
{{- end }}

{{- /* Prepare and print output */}}
{{- if $objectErrors }}
  {{- $output := list -}}
  {{- range $name, $errors := $objectErrors }}
    {{- $output = append $output (printf "Errors for Java agent '%s':\n%s" $name (join "\n" $errors)) -}}
  {{- end }}
  {{- printf "\nCONFIGURATION CHECKS:\n%s" (join "\n\n" $output) | fail -}}
{{- end -}}
{{- end -}}
