#!/bin/bash

GREEN='\033[1;32m'
PURPLE='\033[1;35m'
NC='\033[0m'

# GitHub repository to install from (this fork). Override to use another fork.
REPO="${REPO:-fffedor/3x-ui-exporter}"

step() {
  echo -e "\n${GREEN}[$1/8] $2${NC}"
}

# Prompt to continue after a non-fatal validation issue, or abort the install.
confirm_continue_or_abort() {
  read -p "Continue anyway? (y/N): " CONTINUE
  if [ "$CONTINUE" != "y" ] && [ "$CONTINUE" != "Y" ]; then
    echo "Installation aborted."
    exit 1
  fi
}

# Check if script is run as root
if [ "$(id -u)" -ne 0 ]; then
    echo "Error: This script must be run as root (sudo)."
    exit 1
fi

# Determine system architecture
ARCH=$(uname -m)
case ${ARCH} in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

# Get latest release tag
echo "Fetching latest release information..."
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest")
if [ $? -ne 0 ] || [ -z "$LATEST_RELEASE" ]; then
    echo "Failed to fetch release information. Installation aborted."
    exit 1
fi

VERSION=$(echo "${LATEST_RELEASE}" | grep -Po '"tag_name": "\K.*?(?=")')
echo -e "\n${PURPLE}✨ Starting 3X-UI Exporter $VERSION automated install wizard...\033[0m"

# Create dedicated system user for running the service
step 1 "Creating x-ui-exporter user"
if ! id -u x-ui-exporter > /dev/null 2>&1; then
    useradd -r -s /bin/false x-ui-exporter
    if [ $? -ne 0 ]; then
        echo "Failed to create user. Installation aborted."
        exit 1
    fi
fi

# Download the appropriate archive
TEMP_DIR=$(mktemp -d)
ARCHIVE_NAME="3x-ui-exporter-${VERSION}-linux-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

step 2 "Downloading binary from: ${DOWNLOAD_URL}"
curl -L -o "${TEMP_DIR}/${ARCHIVE_NAME}" "${DOWNLOAD_URL}"
if [ $? -ne 0 ]; then
    echo "Failed to download binary. Installation aborted."
    rm -rf "${TEMP_DIR}"
    exit 1
fi

# Extract binary
step 3 "Extracting binary..."
tar -xzf "${TEMP_DIR}/${ARCHIVE_NAME}" -C "${TEMP_DIR}"
if [ $? -ne 0 ]; then
    echo "Failed to extract binary. Installation aborted."
    rm -rf "${TEMP_DIR}"
    exit 1
fi

# Force remove old binary if exists
if [ -f /usr/local/bin/x-ui-exporter ]; then
    rm -f /usr/local/bin/x-ui-exporter
    if [ $? -ne 0 ]; then
        echo "Failed to remove old binary. Installation aborted."
        rm -rf "${TEMP_DIR}"
        exit 1
    fi
fi

# Install binary to /usr/local/bin
step 4 "Installing binary to /usr/local/bin..."
cp "${TEMP_DIR}/x-ui-exporter" /usr/local/bin/
if [ $? -ne 0 ]; then
    echo "Failed to install binary. Installation aborted."
    rm -rf "${TEMP_DIR}"
    exit 1
fi

# Clean up and set permissions
rm -rf "${TEMP_DIR}"
chmod 755 /usr/local/bin/x-ui-exporter

# Create config directory
step 5 "Creating configuration directory..."
mkdir -p /etc/x-ui-exporter/
if [ $? -ne 0 ]; then
    echo "Failed to create config directory. Installation aborted."
    exit 1
fi

# Check if config file already exists
CONFIG_FILE="/etc/x-ui-exporter/config.yaml"
SKIP_CONFIG_SETUP=0
if [ -f "$CONFIG_FILE" ]; then
    echo "Configuration file already exists at $CONFIG_FILE"
    while true; do
        read -p "Do you want to overwrite the existing config? (y/N): " yn
        case $yn in
            [Yy]* )
                echo "Overwriting existing configuration..."
                break
                ;;
            * )
                echo "Skipping config setup."
                SKIP_CONFIG_SETUP=1
                ;;
        esac
        [ $SKIP_CONFIG_SETUP -eq 1 ] && break
    done
fi

if [ $SKIP_CONFIG_SETUP -eq 0 ]; then
    # Download example config file
    echo "Downloading example config from GitHub..."
    curl -s -o "$CONFIG_FILE" "https://raw.githubusercontent.com/${REPO}/main/config-example.yaml"
    if [ $? -ne 0 ]; then
        echo "Failed to download config file. Installation aborted."
        exit 1
    fi

    # Interactive configuration
    echo "Provide your 3X-UI panel details:"

    # Get Panel URL
    while true; do
        read -p "Enter Panel URL (e.g., http://example.com:54321): " PANEL_URL
        # Remove trailing slash if present
        PANEL_URL=${PANEL_URL%/}

        if [ -z "$PANEL_URL" ]; then
            echo "Error: Panel URL cannot be empty. Please try again."
        elif [[ ! "$PANEL_URL" =~ ^https?:// ]]; then
            echo "Error: Panel URL must start with http:// or https://. Please try again."
        else
            break
        fi
    done

    # Get credentials
    while true; do
        read -p "Enter Panel Username: " PANEL_USERNAME
        if [ -z "$PANEL_USERNAME" ]; then
            echo "Error: Panel Username cannot be empty. Please try again."
        else
            break
        fi
    done

    while true; do
        read -s -p "Enter Panel Password: " PANEL_PASSWORD
        echo ""
        if [ -z "$PANEL_PASSWORD" ]; then
            echo "Error: Panel Password cannot be empty. Please try again."
        else
            break
        fi
    done

    # Validate connection to panel (3X-UI v3.0+ CSRF-aware login, mirroring the exporter)
    echo "Validating connection to panel..."
    TEMP_RESPONSE=$(mktemp)
    COOKIE_JAR=$(mktemp)

    # 1) Mint a CSRF token (v3.0+); the session cookie is stored in COOKIE_JAR.
    CSRF_TOKEN=$(curl -s --max-time 15 -c "$COOKIE_JAR" -H "Accept: application/json" \
        "${PANEL_URL}/csrf-token" | grep -Po '"obj"\s*:\s*"\K[^"]*')

    # 2) Log in carrying the CSRF token + session cookie, form-urlencoded like the exporter.
    CSRF_HEADER=()
    [ -n "$CSRF_TOKEN" ] && CSRF_HEADER=(-H "X-CSRF-Token: ${CSRF_TOKEN}")
    LOGIN_RESULT=$(curl -s --max-time 15 -w "%{http_code}" -o "$TEMP_RESPONSE" -X POST "${PANEL_URL}/login" \
        -b "$COOKIE_JAR" \
        "${CSRF_HEADER[@]}" \
        --data-urlencode "username=${PANEL_USERNAME}" \
        --data-urlencode "password=${PANEL_PASSWORD}")
    CURL_EXIT_CODE=$?

    LOGIN_BODY=$(cat "$TEMP_RESPONSE" 2>/dev/null)
    LOGIN_MSG=$(echo "$LOGIN_BODY" | grep -Po '"msg"\s*:\s*"\K[^"]*')
    rm -f "$TEMP_RESPONSE" "$COOKIE_JAR"

    if [ $CURL_EXIT_CODE -ne 0 ]; then
        echo "Failed to connect to panel. Network error (curl exit code: $CURL_EXIT_CODE)"
        echo "Please check if the panel URL is correct and the server is reachable."
        confirm_continue_or_abort
    elif [ -z "$CSRF_TOKEN" ]; then
        echo "Panel returned no CSRF token from ${PANEL_URL}/csrf-token."
        echo "This exporter requires 3X-UI v3.0+; older panels are not supported."
        confirm_continue_or_abort
    elif echo "$LOGIN_BODY" | grep -qE '"success"[[:space:]]*:[[:space:]]*true'; then
        echo "Successfully connected to panel!"
    else
        echo "Authentication failed (HTTP ${LOGIN_RESULT})${LOGIN_MSG:+: ${LOGIN_MSG}}."
        echo "Please verify your panel username and password."
        confirm_continue_or_abort
    fi

    # Update the config file with user input
    echo "Updating configuration file with provided details..."
    # Escape special characters in variables for sed
    PANEL_URL_ESCAPED=$(echo "$PANEL_URL" | sed 's/[\/&]/\\&/g')
    PANEL_USERNAME_ESCAPED=$(echo "$PANEL_USERNAME" | sed 's/[\/&]/\\&/g')
    PANEL_PASSWORD_ESCAPED=$(echo "$PANEL_PASSWORD" | sed 's/[\/&]/\\&/g')

    sed -i "s|panel-base-url:.*|panel-base-url: \"${PANEL_URL_ESCAPED}\"|" "$CONFIG_FILE"
    sed -i "s|panel-username:.*|panel-username: \"${PANEL_USERNAME_ESCAPED}\"|" "$CONFIG_FILE"
    sed -i "s|panel-password:.*|panel-password: \"${PANEL_PASSWORD_ESCAPED}\"|" "$CONFIG_FILE"
else
    echo "Using existing configuration file without changes."
fi

chmod 644 "$CONFIG_FILE"
chown -R x-ui-exporter:x-ui-exporter /etc/x-ui-exporter

# Create systemd service file
step 6 "Downloading systemd service file from GitHub..."
curl -s -o /etc/systemd/system/x-ui-exporter.service "https://raw.githubusercontent.com/${REPO}/main/x-ui-exporter.service"

if [ $? -ne 0 ]; then
    echo "Failed to create service file. Installation aborted."
    exit 1
fi

sed -i "s|^Description=\(.*\)|Description=\1 ${VERSION}|" /etc/systemd/system/x-ui-exporter.service
chmod 644 /etc/systemd/system/x-ui-exporter.service

# Reload systemd to recognize the new service
step 7 "Reloading systemd daemon..."
systemctl daemon-reload
if [ $? -ne 0 ]; then
    echo "Failed to reload systemd. Installation aborted."
    exit 1
fi

# Enable and start (or restart) the service
step 8 "Enabling and starting x-ui-exporter service..."
if systemctl is-active --quiet x-ui-exporter.service; then
    echo "Service is already running. Restarting..."
    systemctl restart x-ui-exporter.service
    if [ $? -ne 0 ]; then
        echo "Failed to restart service. Installation aborted."
        exit 1
    fi
else
    systemctl enable x-ui-exporter.service
    if [ $? -ne 0 ]; then
        echo "Failed to enable service. Installation aborted."
        exit 1
    fi

    systemctl start x-ui-exporter.service
    if [ $? -ne 0 ]; then
        echo "Failed to start service. Installation aborted."
        exit 1
    fi
fi

sudo systemctl status x-ui-exporter --no-pager

echo -e "\n${PURPLE}✅ 3X-UI Exporter is installed!"
echo -e "${GREEN}\nCheck status:      ${NC}sudo systemctl status x-ui-exporter --no-pager"
echo -e "${GREEN}Binary path:       ${NC}/usr/local/bin/x-ui-exporter"
echo -e "${GREEN}Config path:       ${NC}$CONFIG_FILE"
echo ""
echo -e "You can view logs with: journalctl -u x-ui-exporter.service"
echo -e "Support the project: \033[1;33mhttps://pay.cloudtips.ru/p/67507843${NC}"
echo ""