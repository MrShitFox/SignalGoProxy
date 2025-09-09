#!/bin/bash

# SignalGoProxy Installer for Debian 12/13
# This script automates the installation and uninstallation of SignalGoProxy.

# --- Configuration ---
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="signalgoproxy"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
GITHUB_REPO="MrShitFox/SignalGoProxy"

# --- Helper Functions ---

# Function to print messages in color
print_msg() {
    COLOR=$1
    MSG=$2
    case "$COLOR" in
        "red") echo -e "\e[31m${MSG}\e[0m" ;;
        "green") echo -e "\e[32m${MSG}\e[0m" ;;
        "yellow") echo -e "\e[33m${MSG}\e[0m" ;;
        *) echo "${MSG}" ;;
    esac
}

# --- Main Functions ---

install_proxy() {
    print_msg "green" "Starting SignalGoProxy installation..."

    # 1. Check dependencies
    print_msg "yellow" "Checking dependencies..."
    DEPS="curl jq unzip"
    MISSING_DEPS=()
    for dep in $DEPS; do
        if ! command -v "$dep" &> /dev/null; then
            MISSING_DEPS+=("$dep")
        fi
    done

    if [ ${#MISSING_DEPS[@]} -ne 0 ]; then
        print_msg "red" "The following dependencies are missing: ${MISSING_DEPS[*]}. Please install them first."
        print_msg "yellow" "You can install them using: sudo apt-get update && sudo apt-get install -y ${MISSING_DEPS[*]}"
        exit 1
    fi
    print_msg "green" "Dependencies are satisfied."

    # 2. Get the latest release
    print_msg "yellow" "Fetching the latest release from GitHub..."
    LATEST_RELEASE_URL=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | jq -r '.assets[] | select(.name | contains("linux-amd64.zip")) | .browser_download_url')

    if [ -z "$LATEST_RELEASE_URL" ] || [ "$LATEST_RELEASE_URL" == "null" ]; then
        print_msg "red" "Could not find the latest linux-amd64 release for SignalGoProxy. Exiting."
        exit 1
    fi
    print_msg "green" "Found latest release: $LATEST_RELEASE_URL"

    # Download and extract
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR" || exit
    print_msg "yellow" "Downloading..."
    curl -sL "$LATEST_RELEASE_URL" -o signalgoproxy.zip
    if [ $? -ne 0 ]; then
        print_msg "red" "Download failed. Please check your internet connection or the URL."
        rm -rf "$TMP_DIR"
        exit 1
    fi

    print_msg "yellow" "Extracting..."
    unzip -q signalgoproxy.zip
    if [ ! -f "signalgoproxy" ]; then
        print_msg "red" "The zip file did not contain the 'signalgoproxy' binary."
        rm -rf "$TMP_DIR"
        exit 1
    fi

    print_msg "yellow" "Installing binary to $INSTALL_DIR..."
    sudo mv signalgoproxy "$INSTALL_DIR/"
    sudo chmod +x "${INSTALL_DIR}/signalgoproxy"
    rm -rf "$TMP_DIR"
    print_msg "green" "Binary installed successfully."


    # 3. Get user input
    read -rp "Enter your domain: " DOMAIN
    while [ -z "$DOMAIN" ]; do
        print_msg "red" "Domain cannot be empty."
        read -rp "Enter your domain: " DOMAIN
    done

    print_msg "yellow" "Select a masking mode:"
    select MASK_MODE in "Nginx" "Apache" "proxy"; do
        case $MASK_MODE in
            Nginx|Apache)
                PROXY_URL=""
                break
                ;;
            proxy)
                read -rp "Enter the full proxy URL (e.g., https://ya.ru): " PROXY_URL
                while [[ -z "$PROXY_URL" || ! "$PROXY_URL" =~ ^https?:// ]]; do
                    print_msg "red" "Invalid URL. It must start with http:// or https://"
                    read -rp "Enter the full proxy URL (e.g., https://ya.ru): " PROXY_URL
                done
                break
                ;;
            *)
                print_msg "red" "Invalid option. Please select 1, 2, or 3."
                ;;
        esac
    done

    # 4. Create and install systemd service
    print_msg "yellow" "Creating systemd service..."
    SERVICE_CONTENT="[Unit]
Description=SignalGoProxy Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=${INSTALL_DIR}/signalgoproxy -domain ${DOMAIN} -stealth-mode ${MASK_MODE,,}"

    if [ "$MASK_MODE" == "proxy" ]; then
        SERVICE_CONTENT="${SERVICE_CONTENT} -proxy-url ${PROXY_URL}"
    fi

    SERVICE_CONTENT="${SERVICE_CONTENT}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target"

    echo "$SERVICE_CONTENT" | sudo tee "$SERVICE_FILE" > /dev/null

    print_msg "yellow" "Reloading systemd daemon and starting the service..."
    sudo systemctl daemon-reload
    sudo systemctl enable "${SERVICE_NAME}"
    sudo systemctl start "${SERVICE_NAME}"

    # 5. Final message
    print_msg "green" "Installation complete!"
    print_msg "green" "SignalGoProxy is now running."
    echo "--------------------------------------------------"
    echo "To manage the service, use the following commands:"
    print_msg "yellow" "sudo systemctl status ${SERVICE_NAME}"
    print_msg "yellow" "sudo systemctl stop ${SERVICE_NAME}"
    print_msg "yellow" "sudo systemctl start ${SERVICE_NAME}"
    print_msg "yellow" "sudo systemctl restart ${SERVICE_NAME}"
    echo "--------------------------------------------------"
}

uninstall_proxy() {
    print_msg "green" "Starting SignalGoProxy uninstallation..."

    # Stop and disable the service
    if [ -f "$SERVICE_FILE" ]; then
        print_msg "yellow" "Stopping and disabling systemd service..."
        sudo systemctl stop "${SERVICE_NAME}"
        sudo systemctl disable "${SERVICE_NAME}"
        sudo rm "$SERVICE_FILE"
        sudo systemctl daemon-reload
        print_msg "green" "Service removed."
    else
        print_msg "yellow" "Service file not found, skipping."
    fi

    # Remove the binary
    if [ -f "${INSTALL_DIR}/signalgoproxy" ]; then
        print_msg "yellow" "Removing binary..."
        sudo rm "${INSTALL_DIR}/signalgoproxy"
        print_msg "green" "Binary removed."
    else
        print_msg "yellow" "Binary not found, skipping."
    fi

    print_msg "green" "Uninstallation complete!"
}


# --- Script Entry Point ---

main() {
    if [ "$(id -u)" -eq 0 ]; then
        print_msg "red" "This script should not be run as root. It will ask for sudo permissions when needed."
        exit 1
    fi

    clear
    echo "SignalGoProxy Installer for Debian"
    echo "------------------------------------"
    echo "1. Install SignalGoProxy"
    echo "2. Uninstall SignalGoProxy"
    echo "3. Exit"
    echo "------------------------------------"
    read -rp "Select an option [1-3]: " choice

    case "$choice" in
        1)
            install_proxy
            ;;
        2)
            uninstall_proxy
            ;;
        3)
            print_msg "yellow" "Exiting."
            exit 0
            ;;
        *)
            print_msg "red" "Invalid choice. Exiting."
            exit 1
            ;;
    esac
}

main "$@"
