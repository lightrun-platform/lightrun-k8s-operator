#!/bin/sh
# Script to initialize and configure the Lightrun agent
# This script:
# 1. Validates required environment variables
# 2. Sets up a working directory
# 3. Merges configuration files
# 4. Updates configuration with environment variables
# 5. Copies the final configuration to destination

set -e

# Constants
TMP_DIR="/tmp"
WORK_DIR="${TMP_DIR}/agent-workdir"
FINAL_DEST="${TMP_DIR}/agent"
CONFIG_MAP_DIR="${TMP_DIR}/cm"

# Function to validate required environment variables
validate_env_vars() {
    local missing_vars=""

    if [ -z "${LIGHTRUN_KEY}" ]; then
        missing_vars="${missing_vars} LIGHTRUN_KEY"
    fi
    if [ -z "${PINNED_CERT}" ]; then
        missing_vars="${missing_vars} PINNED_CERT"
    fi
    if [ -z "${LIGHTRUN_SERVER}" ]; then
        missing_vars="${missing_vars} LIGHTRUN_SERVER"
    fi

    if [ -n "${missing_vars}" ]; then
        echo "Error: Missing required environment variables:${missing_vars}"
        exit 1
    fi
}

# Function to setup working directory
setup_working_dir() {
    echo "Setting up working directory at ${WORK_DIR}"
    mkdir -p "${WORK_DIR}"
    cp -R /agent/* "${WORK_DIR}/"
}

# Function to merge configuration files
merge_configs() {
    echo "Merging configuration files"
    local temp_conf="${WORK_DIR}/tempconf"

    # Merge base config and mounted configmap config
    awk -F'=' '{
        if($1 in b) a[b[$1]]=$0;
        else{a[++i]=$0; b[$1]=i}
    } END{for(j=1;j<=i;j++) print a[j]}' \
        "${WORK_DIR}/agent.config" \
        "${CONFIG_MAP_DIR}/agent.config" > "${temp_conf}"

    # Replace the config in the workdir with the merged one
    cp "${temp_conf}" "${WORK_DIR}/agent.config"

    # Copy metadata from configmap to workdir
    cp "${CONFIG_MAP_DIR}/agent.metadata.json" "${WORK_DIR}/agent.metadata.json"

    rm "${temp_conf}"
}

# Function to update configuration with environment variables
update_config() {
    echo "Updating configuration with environment variables"
    local config_file="${WORK_DIR}/agent.config"
    local missing_configuration_params=""

    if sed -n "s|com.lightrun.server=.*|com.lightrun.server=https://${LIGHTRUN_SERVER}|p" "${config_file}" | grep -q .; then
        # Perform actual in-place change
        sed -i "s|com.lightrun.server=.*|com.lightrun.server=https://${LIGHTRUN_SERVER}|" "${config_file}"
    else
        missing_configuration_params="${missing_configuration_params} com.lightrun.server"
    fi
    if sed -n "s|com.lightrun.secret=.*|com.lightrun.secret=${LIGHTRUN_KEY}|p" "${config_file}" | grep -q .; then
        # Perform actual in-place change
        sed -i "s|com.lightrun.secret=.*|com.lightrun.secret=${LIGHTRUN_KEY}|" "${config_file}"
    else
        missing_configuration_params="${missing_configuration_params} com.lightrun.secret"
    fi
    if sed -n "s|pinned_certs=.*|pinned_certs=${PINNED_CERT}|p" "${config_file}" | grep -q .; then
        # Perform actual in-place change
        sed -i "s|pinned_certs=.*|pinned_certs=${PINNED_CERT}|" "${config_file}"
    else
        missing_configuration_params="${missing_configuration_params} pinned_certs"
    fi
    if [ -n "${missing_configuration_params}" ]; then
        echo "Error: Missing configuration parameters:${missing_configuration_params}"
        exit 1
    fi
}

# Function to copy final configuration
copy_final_config() {
    echo "Copying configured agent to final destination ${FINAL_DEST}"
    cp -R "${WORK_DIR}" "${FINAL_DEST}"
}

# Function to cleanup
cleanup() {
    echo "Cleaning up working directory"
    rm -rf "${WORK_DIR}"
}

# Main execution
main() {
    validate_env_vars
    setup_working_dir
    merge_configs
    update_config
    copy_final_config
    cleanup
    echo "Configuration completed successfully"
}

# Execute main function
main
