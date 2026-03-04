# MKTXP

**Mikrotik Traffic Exporter for Prometheus** - A Go rewrite of the Python [mktxp](https://github.com/akpw/mktxp) project.

MKTXP collects metrics from Mikrotik RouterOS devices and exports them in Prometheus format.

## Features

- **Prometheus Exporter**: Expose RouterOS metrics via HTTP `/metrics` endpoint
- **Multi-Router Support**: Monitor multiple Mikrotik devices from a single exporter
- **Secure Connections**: Support for TLS/SSL with certificate verification
- **Automatic Reconnection**: Built-in backoff and retry logic for failed connections
- **YAML Configuration**: Easy configuration with YAML files and template support

## Installation

### From Source

```bash
git clone https://github.com/eleboucher/mktxp-go.git
cd mktxp-go
make build
./build/mktxp --help
```

### Using Docker

```bash
docker build -t mktxp-go:latest .
docker run -d -p 49090:49090 -v ~/mktxp:/home/mktxp mktxp-go:latest
```

## Quick Start

1. **Initialize Configuration**

```bash
./build/mktxp info
```

This creates default configuration files in `~/mktxp/`:
- `mktxp.yaml` - Router configuration
- `_mktxp.yaml` - System settings

2. **Edit Configuration**

Edit `~/mktxp/mktxp.yaml` to add your RouterOS devices:

```yaml
routers:
  MyRouter:
    hostname: 192.168.88.1
    username: admin
    password: your-password
    enabled: true
```

3. **Check Configuration**

```bash
./build/mktxp show
```

4. **Start the Exporter**

```bash
./build/mktxp export
```

Metrics are now available at `http://localhost:49090/metrics`

## CLI Commands

### `mktxp info`

Display MKTXP version and configuration information.

```bash
./build/mktxp info
```

Output:
```
MKTXP - Mikrotik RouterOS Prometheus Exporter

Version:        dev
Git Commit:     abc123
Build Date:     2026-03-03T17:30:00Z

Config Dir:     /home/user/mktxp
Main Config:    /home/user/mktxp/mktxp.yaml
System Config:  /home/user/mktxp/_mktxp.yaml

Listen Address: 0.0.0.0:49090
Socket Timeout: 2s
Routers:        2 configured

Configured Routers:
  - Router1 (192.168.88.1) [enabled]
  - Router2 (192.168.88.2) [disabled]
```

### `mktxp show`

Display configured router entries.

```bash
# Show all routers
./build/mktxp show

# Show specific router
./build/mktxp show --entry-name Router1

# Show config file paths
./build/mktxp show --config
```

Flags:
- `-e, --entry-name string` - Show specific router entry
- `-c, --config` - Show configuration file paths

### `mktxp print`

Print metrics from a specific router to stdout.

```bash
# Print in Prometheus format
./build/mktxp print --entry-name Router1

# Print in JSON format (coming soon)
./build/mktxp print --entry-name Router1 --format json
```

Flags:
- `-e, --entry-name string` - Router entry name (required)
- `-f, --format string` - Output format: prometheus, json (default "prometheus")

### `mktxp export`

Start the Prometheus exporter server.

```bash
# Start with default settings
./build/mktxp export

# Override listen address
./build/mktxp export --listen 0.0.0.0:9090

# Enable debug logging
./build/mktxp export --verbose
```

Flags:
- `--listen string` - Override listen address (default from config)
- `--socket-timeout int` - Override socket timeout in seconds
- `--max-scrape-duration int` - Override per-router scrape timeout in seconds
- `--total-max-scrape-duration int` - Override total scrape timeout in seconds
- `-v, --verbose` - Enable verbose/debug logging

Global Flags:
- `--cfg-dir string` - Configuration directory (default `~/mktxp`)

## Configuration

### System Configuration (`_mktxp.yaml`)

```yaml
mktxp:
  listen: "0.0.0.0:49090"          # Listen address(es), space-separated for multiple
  socket_timeout: 2                # RouterOS API socket timeout (seconds)

  initial_delay_on_failure: 120    # Backoff delay after connection failure
  max_delay_on_failure: 900        # Maximum backoff delay
  delay_inc_div: 5                 # Backoff growth divisor

  verbose_mode: false              # Enable debug logging
  fetch_routers_in_parallel: false # Scrape routers concurrently
  max_worker_threads: 5            # Max concurrent scrapes

  persistent_router_connection_pool: true  # Keep connections open
  persistent_dhcp_cache: true              # Cache DHCP leases
```

### Router Configuration (`mktxp.yaml`)

```yaml
default:
  enabled: true
  port: 8728                    # RouterOS API port (8729 for SSL)
  username: admin
  password: password
  plaintext_login: true         # Use plaintext auth (RouterOS 6.43+)

  # SSL/TLS settings
  use_ssl: false
  no_ssl_certificate: false
  ssl_certificate_verify: false
  ssl_check_hostname: true

  # Feature flags
  health: true
  interface: true
  system: true
  dhcp: true
  connections: true
  route: true
  firewall: true
  # ... (see config file for full list)

routers:
  Router1:
    hostname: 192.168.88.1
    username: monitor
    password: secret123
    custom_labels:
      site: datacenter
      rack: A1

  Router2:
    hostname: 192.168.88.2
    use_ssl: true
    ssl_certificate_verify: true
```

## Endpoints

When running in export mode:

- `/` - Welcome page with endpoint list
- `/metrics` - Prometheus metrics for all configured routers
- `/probe?target=<router-name>` - Metrics for a specific router (multi-target pattern)



## Contributing

Contributions are welcome! Priority areas:

1. **Metric Collectors** - Implement collectors for interface, system, DHCP, etc.
2. **Testing** - Add unit and integration tests
3. **Documentation** - Improve docs and examples

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Credits

- Original Python project: [mktxp](https://github.com/akpw/mktxp) by Arseniy Kuznetsov

## Prometheus Configuration

Example Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: 'mktxp'
    static_configs:
      - targets: ['localhost:49090']

  # For multi-target probing pattern
  - job_name: 'mktxp-probe'
    metrics_path: /probe
    static_configs:
      - targets:
        - Router1
        - Router2
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: localhost:49090
```
