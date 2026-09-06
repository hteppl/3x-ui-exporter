<p align="center">
  <img src="https://raw.githubusercontent.com/hteppl/3x-ui-exporter/main/.github/images/logo.png" alt="logo">
</p>

# 3X-UI Metrics Exporter

[![Release](https://img.shields.io/github/v/release/hteppl/3x-ui-exporter?style=flat-square&logo=github&label=release&color=blue)](https://github.com/hteppl/3x-ui-exporter/releases/latest)
[![Build](https://img.shields.io/github/actions/workflow/status/hteppl/3x-ui-exporter/release.yaml?style=flat-square&logo=githubactions&logoColor=white&label=build)](https://github.com/hteppl/3x-ui-exporter/actions/workflows/release.yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/hteppl/3x-ui-exporter?style=flat-square&logo=go&logoColor=white&label=go)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/hteppl/3x-ui-exporter?style=flat-square)](https://goreportcard.com/report/github.com/hteppl/3x-ui-exporter)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square&logo=gnu&logoColor=white)](LICENSE)
[![GitHub Downloads](https://img.shields.io/github/downloads/hteppl/3x-ui-exporter/total?style=flat-square&logo=github&label=github%20downloads&color=brightgreen)](https://github.com/hteppl/3x-ui-exporter/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/hteppl/x-ui-exporter?style=flat-square&logo=docker&logoColor=white&label=docker%20pulls&color=066da5)](https://hub.docker.com/r/hteppl/x-ui-exporter)

3X-UI Metrics Exporter is a comprehensive tool designed to collect and export metrics from the
[3X-UI Web Panel](https://github.com/MHSanaei/3x-ui). This exporter provides detailed monitoring capabilities for
various aspects of your 3X-UI, including node status, traffic flow, system performance, and user activity, making all
data readily available for integration with the Prometheus monitoring system.

> **Compatibility:** This exporter targets the **3X-UI v3.0+** API (CSRF-authenticated login). Panels older than v3.0
> are not supported.

## Features

- **Online Monitoring**: Tracks the number of online users across your 3X-UI instance.
- **Traffic Metrics**: Monitors total uploaded and downloaded bytes per client or inbound.
- **3X-UI Monitoring**: Provides detailed XRay version information and additional operational metrics from 3X-UI.
- **Version and Start Time Information**: Delivers core version information and confirms whether the core service has
  started successfully.
- **Flexible Configuration Options**: Supports customization through environment variables, `.env` files, and
  command-line arguments, providing maximum flexibility for different deployment scenarios.
- **Multi-Architecture Support**: Features Docker images for multiple architectures, including AMD64 and ARM64, ensuring
  compatibility across diverse deployment environments.
- **Enhanced Security**: Offers optional BasicAuth protection for the metrics endpoint, providing an additional layer of
  security for sensitive monitoring data.
- **Seamless Prometheus Integration**: Designed to work flawlessly with Prometheus, enabling straightforward setup and
  configuration for comprehensive 3X-UI panel monitoring.
- **Comprehensive VPN Monitoring**: Simplifies the monitoring and management of VPN services by providing a rich set of
  metrics, significantly improving visibility into system performance and user activity.

## Metrics

3X-UI Metrics Exporter exposes nine Prometheus gauges covering online users, per-client and per-inbound traffic, and
panel health.

**See [METRICS.md](METRICS.md) for the complete reference** — every metric name, type, and label, along with example
PromQL queries and the gauge-semantics caveats that matter when querying byte totals.

## Integration with Prometheus

To collect metrics with Prometheus, add the exporter to your prometheus.yml configuration file:

```yaml
scrape_configs:
  - job_name: "x-ui_exporter"
    static_configs:
      - targets: ["<exporter-ip>:9090"]
```

Ensure to replace `<your-panel-url>`, `<your-panel-username>`, `<your-panel-password>`, and `<exporter-ip>` with your
actual information.

## Configuration

3X-UI Metrics Exporter is configured with environment variables, which can be supplied directly or through a `.env`
file. Every variable also has an equivalent command-line argument.

Below is a table of configuration options:

| Variable Name          | Command-Line Argument    | Required | Default Value              | Description                                                               |
| ---------------------- | ------------------------ | -------- | -------------------------- | ------------------------------------------------------------------------- |
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

### Env File Configuration

The exporter loads a `.env` file from its working directory on startup. A sample with every option and its default is
provided as [`.env.sample`](.env.sample):

```bash
cp .env.sample .env
```

```dotenv
# 3X-UI panel connection details (required)
PANEL_BASE_URL=https://your-panel-url
PANEL_USERNAME=your-panel-username
PANEL_PASSWORD=your-panel-password

# General settings
UPDATE_INTERVAL=30
TIMEZONE=UTC

# Metrics server configuration
METRICS_IP=0.0.0.0
METRICS_PORT=9090
```

Set `ENV_FILE` to load the file from another location:

```bash
ENV_FILE=/etc/x-ui-exporter/.env ./x-ui-exporter
```

> **Note:** A missing `.env` is not an error — the exporter runs on environment variables and command-line arguments
> alone. A file named explicitly by `ENV_FILE` that does not exist _is_ an error.

Values are applied in order of increasing precedence: the `.env` file, then real environment variables, then
command-line arguments. A variable already exported in the environment (or set by Docker or systemd) overrides the file,
and a flag overrides both.

## Installation

There are several ways to install and run the 3X-UI Metrics Exporter, each tailored to different environments and
deployment preferences. Select the installation method that aligns best with your infrastructure requirements:

### Docker (Recommended)

Running with Docker is the recommended way to deploy the exporter: it needs no toolchain on the host, isolates the
exporter from the rest of your system, and makes updates a single `pull` away.

#### Using Docker Compose:

A ready-to-use [`docker-compose.yml`](docker-compose.yml) is provided with the project. It reads its configuration from
a `.env` file, so copy the provided sample and fill in your panel details:

```bash
cp .env.sample .env
```

Then run:

```bash
docker compose up -d
```

> **Security Recommendation:** For production deployments, it's strongly advised to enable metrics authentication by
> setting `METRICS_PROTECTED=true` and configuring a secure custom metrics username and password.

### Automatic Installation Script

If you would rather run the exporter directly on the host under systemd, an installation script is available:

```bash
bash <(curl -fsSL raw.githubusercontent.com/hteppl/3x-ui-exporter/main/install.sh)
```

During installation, you'll be prompted to enter:

1. Your 3X-UI panel URL
2. Admin username
3. Admin password

> **Note:** The script will validate your credentials to ensure they work with your panel.

The script installs the binary to `/usr/local/bin`, writes your settings to `/etc/x-ui-exporter/.env`, and registers a
systemd service. After installation, the service will be running automatically. You can manage it with:

```bash
sudo systemctl status x-ui-exporter    # Check status
sudo systemctl restart x-ui-exporter   # Restart service
sudo systemctl stop x-ui-exporter      # Stop service
```

### Manual CLI Installation

If you prefer manual installation, download the latest binary from the
[releases page](https://github.com/hteppl/3x-ui-exporter/releases) for your architecture.

#### Running with command-line arguments:

```bash
./x-ui-exporter --panel-base-url="https://your-panel-url" \
                --panel-username="your-panel-username" \
                --panel-password="your-panel-password"
```

#### Running with an env file:

1. Create a `.env` file based on [`.env.sample`](.env.sample)
2. Run the exporter from the same directory:

```bash
cp .env.sample .env
./x-ui-exporter
```

## Development

### Building from Source

Requires Go 1.27 or newer:

```bash
go build -o x-ui-exporter .
go test ./...
```

### Building the Docker Image

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
docker buildx build --platform linux/amd64 -t hteppl/x-ui-exporter:latest .
```

## Contribute

Contributions to 3X-UI Metrics Exporter are warmly welcomed. Whether it's bug fixes, new features, or documentation
improvements, your input helps make this project better. Here's a quick guide to contributing:

1. **Fork & Branch**: Fork this repository and create a branch for your work.
2. **Implement Changes**: Work on your feature or fix, keeping code clean and well-documented.
3. **Test**: Ensure your changes maintain or improve current functionality, adding tests for new features.
4. **Commit & PR**: Commit your changes with clear messages, then open a pull request detailing your work.
5. **Feedback**: Be prepared to engage with feedback and further refine your contribution.

Happy contributing! If you're new to this, GitHub's guide on
[Creating a pull request](https://docs.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request)
is an excellent resource.

## Credits

Maintained by [@hteppl](https://github.com/hteppl), with contributions from:

- [@fffedor](https://github.com/fffedor)
- [@ksusonic](https://github.com/ksusonic)
- [@welcomereality](https://github.com/welcomereality)

## License

This project is licensed under the **GNU Affero General Public License v3.0**. See the [LICENSE](LICENSE) file for the
full text.
