<p align="center">
  <img src="https://raw.githubusercontent.com/hteppl/3x-ui-exporter/main/.github/images/logo.png" alt="logo">
</p>

# 3X-UI Metrics Exporter

[![Release](https://img.shields.io/github/v/release/hteppl/3x-ui-exporter.svg)](https://github.com/hteppl/3x-ui-exporter/releases)
[![DockerHub](https://img.shields.io/badge/DockerHub-x--ui--exporter-blue)](https://hub.docker.com/r/hteppl/x-ui-exporter/)
[![Build](https://img.shields.io/github/actions/workflow/status/hteppl/3x-ui-exporter/release.yaml.svg)](https://github.com/hteppl/3x-ui-exporter/actions)
[![GO Version](https://img.shields.io/github/go-mod/go-version/hteppl/3x-ui-exporter.svg)]()
[![Downloads](https://img.shields.io/github/downloads/hteppl/3x-ui-exporter/total.svg)](https://github.com/hteppl/3x-ui-exporter/releases/latest)
[![License](https://img.shields.io/badge/license-GNU%20AGPLv3-blue.svg?longCache=true)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hteppl/3x-ui-exporter)](https://goreportcard.com/report/github.com/hteppl/3x-ui-exporter)

3X-UI Metrics Exporter is a comprehensive tool designed to collect and export metrics from
the [3X-UI Web Panel](https://github.com/MHSanaei/3x-ui). This exporter provides detailed monitoring capabilities for
various aspects of your 3X-UI, including node status, traffic flow, system performance, and user activity, making all
data readily
available for integration with the Prometheus monitoring system.

> **Compatibility:** This exporter targets the **3X-UI v3.0+** API (CSRF-authenticated login). Panels older than
> v3.0 are not supported.

## Features

- **Online Monitoring**: Tracks the number of online users across your 3X-UI instance.
- **Traffic Metrics**: Monitors total uploaded and downloaded bytes per client or inbound.
- **3X-UI Monitoring**: Provides detailed XRay version information and additional operational metrics from 3X-UI.
- **Version and Start Time Information**: Delivers core version information and confirms whether the core service has
  started
  successfully.
- **Flexible Configuration Options**: Supports customization through environment variables, command-line arguments, and
  YAML configuration files,
  providing maximum flexibility for different deployment scenarios.
- **Multi-Architecture Support**: Features Docker images for multiple architectures, including AMD64 and ARM64,
  ensuring compatibility across diverse deployment environments.
- **Enhanced Security**: Offers optional BasicAuth protection for the metrics endpoint, providing an
  additional layer of security for sensitive monitoring data.
- **Seamless Prometheus Integration**: Designed to work flawlessly with Prometheus, enabling straightforward setup and
  configuration for comprehensive 3X-UI panel monitoring.
- **Comprehensive VPN Monitoring**: Simplifies the monitoring and management of VPN services by providing a rich set of
  metrics,
  significantly improving visibility into system performance and user activity.

## Metrics

Below is a table of the metrics provided by 3X-UI Metrics Exporter.

### Users

Users metrics, such as online:

| Name                      | Description                  |
|---------------------------|------------------------------|
| `x_ui_total_online_users` | Total number of online users |

### Clients

Clients metrics (params: `id`, `email`):

| Name                     | Description                       |
|--------------------------|-----------------------------------|
| `x_ui_client_up_bytes`   | Total uploaded bytes per client   |
| `x_ui_client_down_bytes` | Total downloaded bytes per client |

### Inbounds

Inbounds metrics (params: `id`, `remark`):

| Name                      | Description                        |
|---------------------------|------------------------------------|
| `x_ui_inbound_up_bytes`   | Total uploaded bytes per inbound   |
| `x_ui_inbound_down_bytes` | Total downloaded bytes per inbound |

### System

System metrics (`version` param for `x_ui_xray_version`):

| Name                 | Description                |
|----------------------|----------------------------|
| `x_ui_xray_version`  | XRay version used by 3X-UI |
| `x_ui_panel_threads` | 3X-UI panel threads        |
| `x_ui_panel_memory`  | 3X-UI panel memory usage   |
| `x_ui_panel_uptime`  | 3X-UI panel uptime         |

## Configuration

3X-UI Metrics Exporter can be configured using environment variables, command-line arguments, or a YAML configuration
file. These are alternative methods of configuration, and you should choose one approach for your deployment.

Below is a table of configuration options:

| Variable Name          | Command-Line Argument    | Required | Default Value              | Description                                                               |
|------------------------|--------------------------|----------|----------------------------|---------------------------------------------------------------------------|
| `CONFIG_FILE`          | `--config-file`          | No       | N/A                        | Path to YAML configuration file. When provided, CLI flags are ignored     |
| `PANEL_BASE_URL`       | `--panel-base-url`       | Yes      | `https://<your-panel-url>` | URL of the 3X-UI management panel                                         |
| `PANEL_USERNAME`       | `--panel-username`       | Yes      | `<your-panel-username>`    | Username for the 3X-UI panel                                              |
| `PANEL_PASSWORD`       | `--panel-password`       | Yes      | `<your-panel-password>`    | Password for the 3X-UI panel                                              | 
| `INSECURE_SKIP_VERIFY` | `--insecure-skip-verify` | No       | `false`                    | Skip SSL certificate verification (INSECURE)                              |
| `METRICS_IP`           | `--metrics-ip`           | No       | `0.0.0.0`                  | IP address for the metrics server                                         |
| `METRICS_PORT`         | `--metrics-port`         | No       | `9090`                     | Port for the metrics server                                               |
| `CLIENTS_BYTES_ROWS`   | `--clients-bytes-rows`   | No       | `0`                        | Limit rows for clients up/down bytes (0=all; -1=disable; else top N rows) |
| `METRICS_PROTECTED`    | `--metrics-protected`    | No       | `false`                    | Enable BasicAuth protection for metrics endpoint                          |
| `METRICS_USERNAME`     | `--metrics-username`     | No       | `metricsUser`              | Username for BasicAuth, effective if `METRICS_PROTECTED` is `true`        |
| `METRICS_PASSWORD`     | `--metrics-password`     | No       | `MetricsVeryHardPassword`  | Password for BasicAuth, effective if `METRICS_PROTECTED` is `true`        |
| `UPDATE_INTERVAL`      | `--update-interval`      | No       | `30`                       | Interval (in seconds) for metrics update                                  |
| `TIMEZONE`             | `--timezone`             | No       | `UTC`                      | Timezone for correct time display                                         |

### YAML Configuration

You can use a YAML configuration file to configure the exporter by providing the `--config-file` flag. When using a
configuration file, all settings are read from the file and any command-line arguments are ignored. The YAML
configuration file should contain the same parameters as the command-line arguments, but in YAML format.

A sample configuration file `config-example.yaml` is provided with the project, which you can use as a template for your
own configuration. The structure of the YAML file matches the command-line arguments.

Example YAML configuration:

```yaml
# 3X-UI panel connection details
panel-base-url: "https://your-panel-url"
panel-username: "your-panel-username"
panel-password: "your-panel-password"
insecure-skip-verify: false

# General settings
update-interval: 30
timezone: "UTC"

# Metrics server configuration
metrics-ip: "0.0.0.0"
metrics-port: 9090
clients-bytes-rows: 0
metrics-protected: false
metrics-username: "metricsUser"
metrics-password: "MetricsVeryHardPassword"
```

> **Note:** When using a configuration file with the `--config-file` flag, all settings
> are taken from the configuration file, and any other command-line arguments are ignored.

## Installation

There are several ways to install and run the 3X-UI Metrics Exporter, each tailored to different environments and
deployment preferences. Select the installation method that aligns best with your infrastructure requirements:

### Automatic Installation Script (Recommended)

The easiest way to install the exporter is using automatic installation script:

```bash
bash <(curl -fsSL raw.githubusercontent.com/fffedor/3x-ui-exporter/main/install.sh)
```

During installation, you'll be prompted to enter:

1. Your 3X-UI panel URL
2. Admin username
3. Admin password

> **Note:** The script will validate your credentials to ensure they work with your panel.

After installation, the service will be running automatically. You can manage it with:

```bash
sudo systemctl status x-ui-exporter    # Check status
sudo systemctl restart x-ui-exporter   # Restart service
sudo systemctl stop x-ui-exporter      # Stop service
```

### Manual CLI Installation

If you prefer manual installation, download the latest binary from
the [releases page](https://github.com/hteppl/3x-ui-exporter/releases) for your architecture.

#### Running with command-line arguments:

```bash
./x-ui-exporter --panel-base-url="https://your-panel-url" \
                --panel-username="your-panel-username" \
                --panel-password="your-panel-password"
```

#### Running with a configuration file:

1. Create a `config.yaml` file based on the example configuration
2. Run the exporter:

```bash
./x-ui-exporter --config-file=config.yaml
```

> **Important:** The configuration file approach and command-line arguments cannot be combined.
> When using a configuration file, any command-line arguments are ignored.

### Docker Installation

Running with Docker provides an optimal solution for containerized environments, offering simplified deployment and
streamlined updates.

#### Using Docker Run:

```bash
docker run -d \
  --name x-ui-exporter \
  -e PANEL_BASE_URL="https://your-panel-url" \
  -e PANEL_USERNAME="your-panel-username" \
  -e PANEL_PASSWORD="your-panel-password" \
  -p 9090:9090 \
  hteppl/x-ui-exporter
```

#### Using Docker Compose:

Create a `docker-compose.yml` file:

```yaml
version: "3"
services:
  x-ui-exporter:
    image: hteppl/x-ui-exporter
    container_name: x-ui-exporter
    restart: unless-stopped
    environment:
      - PANEL_BASE_URL=https://your-panel-url
      - PANEL_USERNAME=your-panel-username
      - PANEL_PASSWORD=your-panel-password
      # Optional settings
      # - METRICS_PORT=9090
      # - UPDATE_INTERVAL=30
    ports:
      - "9090:9090"
```

Then run:

```bash
docker-compose up -d
```

> **Security Recommendation:** For production deployments, it's strongly advised to enable metrics authentication by
> setting `METRICS_PROTECTED=true` and configuring a secure custom metrics username and password.

### Docker build

You can build the Docker image locally for both AMD and ARM architectures using Docker Buildx:

```bash
docker buildx create --name multiarch-builder --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg GIT_TAG=$(git describe --tags --always) \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t <registry_name>:<tag> \
  --push .
```

#### Building for a Single Architecture

To build for a specific architecture only:

```bash
docker buildx build --platform linux/amd64 -t ksusonic/3x-ui-exporter:latest .
```

## Integration with Prometheus

To collect metrics with Prometheus, add the exporter to your prometheus.yml configuration file:

```yaml
scrape_configs:
  - job_name: "x-ui_exporter"
    static_configs:
      - targets: [ "<exporter-ip>:9090" ]
```

Ensure to replace `<your-panel-url>`, `<your-panel-username>`, `<your-panel-password>`, and `<exporter-ip>`
with your actual information.

## Contribute

Contributions to 3X-UI Metrics Exporter are warmly welcomed. Whether it's bug fixes, new features, or documentation
improvements, your input helps make this project better. Here's a quick guide to contributing:

1. **Fork & Branch**: Fork this repository and create a branch for your work.
2. **Implement Changes**: Work on your feature or fix, keeping code clean and well-documented.
3. **Test**: Ensure your changes maintain or improve current functionality, adding tests for new features.
4. **Commit & PR**: Commit your changes with clear messages, then open a pull request detailing your work.
5. **Feedback**: Be prepared to engage with feedback and further refine your contribution.

Happy contributing! If you're new to this, GitHub's guide
on [Creating a pull request](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request)
is an excellent resource.
