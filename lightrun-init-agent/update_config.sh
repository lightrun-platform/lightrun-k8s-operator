#!/bin/sh
# Script to initialize and configure the Lightrun agent
# This script:
# 1. Validates required environment variables and files
# 2. Sets up a working directory
# 3. Merges configuration files
# 4. Updates configuration with values from files
# 5. Copies the final configuration to destination
# 6. Optionally copies libstdc++ libraries for Alpine-based images

set -e

# Constants
TMP_DIR="/tmp"
WORK_DIR="${TMP_DIR}/agent-workdir"
FINAL_DEST="${TMP_DIR}/agent"
CONFIG_MAP_DIR="${TMP_DIR}/cm"
SECRET_DIR="/etc/lightrun/secret"
LIBSTDC_DIR="${TMP_DIR}/libstdc"

# Function to get value from either environment variable or file
get_value() {
    local env_var=$1
    local file_path=$2
    local value=""

    # First try environment variable
    eval "value=\$${env_var}"
    if [ -n "${value}" ]; then
        echo "[WARNING] Using environment variable ${env_var}. Please upgrade to the latest operator version for better security and management." >&2
        echo "[INFO] Using value from environment variable ${env_var}" >&2
        echo "${value}"
        return 0
    fi

    # Then try file
    if [ -f "${file_path}" ]; then
        value=$(cat "${file_path}")
        echo "[INFO] Using value from file ${file_path}" >&2
        echo "${value}"
        return 0
    fi

    echo ""
    return 1
}

# Function to validate required files and environment variables
validate_env_vars() {
    local missing_requirements=""

    # Check for LIGHTRUN_SERVER (required in both old and new versions)
    if [ -z "${LIGHTRUN_SERVER}" ]; then
        missing_requirements="${missing_requirements} LIGHTRUN_SERVER"
    fi

    # Check for lightrun_key (either env var or file)
    local lightrun_key=$(get_value "LIGHTRUN_KEY" "${SECRET_DIR}/lightrun_key")
    if [ -z "${lightrun_key}" ]; then
        missing_requirements="${missing_requirements} LIGHTRUN_KEY"
    fi

    # Check for pinned_cert (either env var or file)
    local pinned_cert=$(get_value "PINNED_CERT" "${SECRET_DIR}/pinned_cert_hash")
    if [ -z "${pinned_cert}" ]; then
        missing_requirements="${missing_requirements} PINNED_CERT"
    fi

    if [ -n "${missing_requirements}" ]; then
        echo "Error: Missing required environment variables or files:${missing_requirements}"
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

# Function to update configuration with values from files
update_config() {
    echo "Updating configuration with values from files"
    local config_file="${WORK_DIR}/agent.config"
    local missing_configuration_params=""

    if [ ! -f "${config_file}" ]; then
        echo "[ERROR] Config file not found at ${config_file}"
        exit 1
    fi

    # Get values from either environment variables or files
    local lightrun_key=$(get_value "LIGHTRUN_KEY" "${SECRET_DIR}/lightrun_key")
    local pinned_cert=$(get_value "PINNED_CERT" "${SECRET_DIR}/pinned_cert_hash")

    if sed -n "s|com.lightrun.server=.*|com.lightrun.server=https://${LIGHTRUN_SERVER}|p" "${config_file}" | grep -q .; then
        # Perform actual in-place change
        sed -i "s|com.lightrun.server=.*|com.lightrun.server=https://${LIGHTRUN_SERVER}|" "${config_file}"
    else
        missing_configuration_params="${missing_configuration_params} com.lightrun.server"
    fi
    if sed -n "s|com.lightrun.secret=.*|com.lightrun.secret=${lightrun_key}|p" "${config_file}" | grep -q .; then
        # Perform actual in-place change
        sed -i "s|com.lightrun.secret=.*|com.lightrun.secret=${lightrun_key}|" "${config_file}"
    else
        missing_configuration_params="${missing_configuration_params} com.lightrun.secret"
    fi
    if sed -n "s|pinned_certs=.*|pinned_certs=${pinned_cert}|p" "${config_file}" | grep -q .; then
        # Perform actual in-place change
        sed -i "s|pinned_certs=.*|pinned_certs=${pinned_cert}|" "${config_file}"
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

# Function to copy libstdc++ libraries
copy_libstdc() {
    if [ "${MOUNT_LIBSTDC}" = "true" ]; then
        echo "========================================="
        echo "COPYING LIBSTDC++ LIBRARIES"
        echo "========================================="
        
        # Create the directory
        mkdir -p "${LIBSTDC_DIR}"
        echo "Created directory: ${LIBSTDC_DIR}"
        echo ""
        
        # Search for and copy libstdc++ libraries
        echo "Searching for libstdc++ libraries..."
        echo "Running as user: $(id)"
        echo ""
        
        # List of paths to search
        for search_path in /usr/lib /lib /usr/lib64 /lib64 /usr/lib/aarch64-linux-gnu /lib/aarch64-linux-gnu /usr/lib/x86_64-linux-gnu /lib/x86_64-linux-gnu; do
            if [ -d "${search_path}" ]; then
                echo "Checking ${search_path}..."
                # Use ls instead of find for simpler logic
                lib_files=$(ls "${search_path}"/libstdc++.so* 2>/dev/null || true)
                if [ -n "${lib_files}" ]; then
                    for lib_file in ${lib_files}; do
                        if [ -f "${lib_file}" ] || [ -L "${lib_file}" ]; then
                            echo "  Found: ${lib_file}"
                            # Try to copy following symlinks first, then regular copy
                            if cp -L "${lib_file}" "${LIBSTDC_DIR}/" 2>/dev/null; then
                                echo "    ✓ Copied (following symlink)"
                            elif cp "${lib_file}" "${LIBSTDC_DIR}/" 2>/dev/null; then
                                echo "    ✓ Copied (as-is)"
                            else
                                echo "    ✗ Failed to copy"
                            fi
                        fi
                    done
                fi
            fi
        done
        
        echo ""
        echo "----------------------------------------"
        echo "Contents of ${LIBSTDC_DIR}:"
        if ls -lah "${LIBSTDC_DIR}/" 2>/dev/null; then
            echo "----------------------------------------"
        else
            echo "  (empty or not accessible)"
            echo "----------------------------------------"
        fi
        
        # Count files (excluding . and ..)
        lib_count=$(find "${LIBSTDC_DIR}" -maxdepth 1 -type f -o -type l | wc -l)
        if [ "${lib_count}" -gt 0 ]; then
            echo "✓ Successfully copied ${lib_count} libstdc++ library file(s)"
        else
            echo "⚠ WARNING: No libstdc++ libraries were copied!"
            echo "  Debugging information:"
            echo "  - Checking if libstdc++ is installed:"
            apk info libstdc++ 2>/dev/null || echo "    libstdc++ package not found"
            echo "  - Looking for any libstdc++ files:"
            find /usr /lib -name '*libstdc++*' 2>/dev/null | head -10 || echo "    No files found"
        fi
        
        echo "========================================="
        echo ""
    else
        echo "MOUNT_LIBSTDC not set to 'true', skipping libstdc++ library copying"
    fi
}

# Main execution
main() {
    copy_libstdc
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
