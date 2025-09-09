# SignalGoProxy

[![Go Version](https://img.shields.io/badge/Go-1.24.6%2B-blue)](https://golang.org/) [![License: GPLv3](https://img.shields.io/badge/License-GPLv3-red)](LICENSE)

A high-performance, stealthy Signal proxy written in Go, designed to provide resilient access to the Signal network.

## The Problem and The Solution

In some regions, access to secure communication platforms like Signal is restricted or blocked. The official [Signal TLS Proxy](https://github.com/signalapp/Signal-TLS-Proxy) was created to help users bypass these restrictions.

**SignalGoProxy** is a powerful, alternative implementation of that concept. We ported the core mechanics of the official proxy to Go a language known for its performance and concurrency and added unique features to make the proxy more robust, efficient, and harder for censors to detect. It is fully compatible with the official Signal clients.

## Key Features

  - **🚀 High Performance:** Built with Go, it's lightweight, fast, and can handle a large number of concurrent connections with minimal resource usage.
  - **🛡️ Stealth Modes:** Camouflage your proxy traffic as a standard web server. If someone (or a censor's bot) visits your proxy's domain in a browser, they will see a generic webpage instead of revealing the proxy.
      - **Nginx Mode:** Simulates a default Nginx welcome page.
      - **Apache Mode:** Simulates a default Apache welcome page.
      - **Proxy Mode:** Forwards all non-Signal traffic to a legitimate website of your choice, making your server appear completely unrelated to Signal.
  - **🔒 Automatic TLS:** Integrates with Let's Encrypt to automatically provision and renew TLS certificates for your domain, ensuring all traffic is securely encrypted.
  - **🧩 Zero Dependencies:** Distributed as a single, static binary with no external dependencies required.
  - **💨 Easy Deployment:** Get your proxy up and running in minutes with a simple installation script.

## How It Works

SignalGoProxy intelligently inspects incoming connections to differentiate between legitimate Signal traffic and other activity. This is achieved by examining the **Server Name Indication (SNI)** in the initial TLS handshake.

  - **If the SNI matches a known Signal server** (e.g., `chat.signal.org`), the connection is transparently forwarded to the actual Signal server.
  - **If the SNI does not match, or if the request is plain HTTP**, the proxy serves a decoy response based on the configured "Stealth Mode".

This mechanism ensures that only genuine Signal traffic reaches its destination, while probes from network censors are diverted, protecting the proxy from being easily discovered and blocked.

## Installation and Setup

### Requirements

  - A VPS (Virtual Private Server) running a modern Linux distribution (like Debian 12/13).
  - A domain name (e.g., `signal.mydomain.com`) with its A record pointing to your VPS's IP address.
  - Ports **80** and **443** must be open on your server's firewall.

### Automatic Installation (Recommended)

An easy-to-use script automates the entire installation process, including downloading the binary, setting up the configuration, and creating a systemd service.

```bash
bash -c "$(curl -sL https://raw.githubusercontent.com/MrShitFox/SignalGoProxy/main/debian_x86_Installer.sh)"
```

The script will guide you through the configuration.

### Manual Installation

1.  **Download the Binary:**
    Go to the [Releases page](https://github.com/MrShitFox/SignalGoProxy/releases/latest/) and download the latest `SignalGoProxy-v1.1.0-linux-amd64.zip` archive.

2.  **Install the Binary:**

    ```bash
    # Unzip the archive
    unzip SignalGoProxy-v1.1.0-linux-amd64.zip

    # Move the binary to a system path
    sudo mv signalgoproxy /usr/local/bin/

    # Make it executable
    sudo chmod +x /usr/local/bin/signalgoproxy
    ```

3.  **Run the Proxy:**
    Run the following command, replacing the arguments with your values.

    ```bash
    /usr/local/bin/signalgoproxy -domain YOUR_DOMAIN -stealth-mode nginx
    ```

### Configuration

You can configure SignalGoProxy using command-line flags:

  - `-domain` (Required): Your domain name for the TLS certificate.
  - `-stealth-mode`: The camouflage mode. Options are `nginx` (default), `apache`, `proxy`, or `none`.
  - `-proxy-url`: If using `proxy` stealth mode, this is the full URL to which non-Signal traffic will be forwarded.

**Examples:**

  - **Nginx Mode:**
    ```bash
    signalgoproxy -domain my.domain.com -stealth-mode nginx
    ```
  - **Proxy to another website:**
    ```bash
    signalgoproxy -domain my.domain.com -stealth-mode proxy -proxy-url https://example.com
    ```

### Running as a systemd Service

To ensure the proxy runs automatically on boot, create a systemd service file at `/etc/systemd/system/signalgoproxy.service`:

```ini
[Unit]
Description=SignalGoProxy Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/signalgoproxy -domain YOUR_DOMAIN -stealth-mode nginx
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Reload systemd and enable the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable signalgoproxy
sudo systemctl start signalgoproxy
```

## Signal Client Configuration

Once your proxy is running, configure your Signal client to use it:

1.  Open Signal.
2.  Go to **Settings** \> **Data and storage**.
3.  Scroll down to **Use proxy** click and enable it.
4.  Enter your full domain: `https://your.domain.com`
5.  Save the settings. Signal will automatically connect through your proxy.

## License

This project is licensed under the **GNU General Public License v3.0**. See the [LICENSE](https://www.gnu.org/licenses/gpl-3.0.en.html) file for details.
