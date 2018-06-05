# OONI Collector

The OONI Collector is responsible for gathering measurements coming from probes.

## Usage

```
ooni-collector start
```

## Configuration

The skeleton of the configuration file is the following:

```
[core]
log-level = "INFO"
data-root = "/var/ooni-collector"
is-dev = false

[api]
port = 8080
address = "127.0.0.1"
admin-password = "changeme"

[aws]
access-key-id = "XXX"
secret-access-key = "XXX"
```

`core.data-root`: defines what is the working directory for the OONI Collector.
The user running the collector service must have write permissions to this
directory. The default path is: `/var/ooni-collector`. Report files will be
written to `/var/ooni-collector/reports/`.

`api.admin-password`: sets the basic auth password for the user `admin` when
accessing the API endpoint at path `/admin/report-files` and
`/admin/report-file/:filename`. This API endpoint is used to retrieve report
files and to delete them.
