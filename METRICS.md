# Metrics Reference

Every metric exposed by 3X-UI Metrics Exporter on the `/metrics` endpoint, what it means, and how to query it.

The endpoint can be protected with BasicAuth — see `METRICS_PROTECTED` in the
[configuration section](README.md#configuration) of the README.

## At a Glance

| Metric                    | Labels         | What it tells you                      |
| ------------------------- | -------------- | -------------------------------------- |
| `x_ui_total_online_users` | —              | How many users are connected right now |
| `x_ui_client_up_bytes`    | `id`, `email`  | Lifetime upload per client             |
| `x_ui_client_down_bytes`  | `id`, `email`  | Lifetime download per client           |
| `x_ui_inbound_up_bytes`   | `id`, `remark` | Lifetime upload per inbound            |
| `x_ui_inbound_down_bytes` | `id`, `remark` | Lifetime download per inbound          |
| `x_ui_xray_version`       | `version`      | Which XRay build the panel is running  |
| `x_ui_panel_threads`      | —              | Threads used by the panel process      |
| `x_ui_panel_memory`       | —              | Memory used by the panel process       |
| `x_ui_panel_uptime`       | —              | How long the panel has been running    |

All nine are Prometheus **gauges**. See [Before You Write Queries](#before-you-write-queries) for what that means in
practice.

## Users

| Name                      | Type  | Labels | Description                  |
| ------------------------- | ----- | ------ | ---------------------------- |
| `x_ui_total_online_users` | Gauge | —      | Total number of online users |

A simple count of currently connected users, refreshed once per update cycle. This is a point-in-time value, so a user
who connects and disconnects between two cycles is never counted.

## Clients

| Name                     | Type  | Labels        | Description                       |
| ------------------------ | ----- | ------------- | --------------------------------- |
| `x_ui_client_up_bytes`   | Gauge | `id`, `email` | Total uploaded bytes per client   |
| `x_ui_client_down_bytes` | Gauge | `id`, `email` | Total downloaded bytes per client |

Lifetime traffic counters as the panel reports them — they grow until you reset traffic in 3X-UI. The `email` label is
whatever identifier you gave the client in the panel; it is free-form text and does not have to be an email address.

**These metrics are optional.** `CLIENTS_BYTES_ROWS` decides how many client series get exported:

| Value     | Result                                                                |
| --------- | --------------------------------------------------------------------- |
| `0`       | Export every client (the default)                                     |
| `-1`      | Export no client metrics at all                                       |
| `N` (> 0) | Export only the top N clients by upload **and** the top N by download |

If you run a panel with hundreds of clients, a positive `N` keeps series cardinality under control. Note that the top-N
lists are computed per inbound, and the upload and download rankings are selected independently — a client can appear in
one and not the other.

## Inbounds

| Name                      | Type  | Labels         | Description                        |
| ------------------------- | ----- | -------------- | ---------------------------------- |
| `x_ui_inbound_up_bytes`   | Gauge | `id`, `remark` | Total uploaded bytes per inbound   |
| `x_ui_inbound_down_bytes` | Gauge | `id`, `remark` | Total downloaded bytes per inbound |

The same lifetime counters, aggregated per inbound instead of per client. `remark` is the inbound's label from the
panel. Inbound totals are always exported, regardless of `CLIENTS_BYTES_ROWS`.

## System

| Name                 | Type  | Labels    | Description                               |
| -------------------- | ----- | --------- | ----------------------------------------- |
| `x_ui_xray_version`  | Gauge | `version` | XRay version used by 3X-UI                |
| `x_ui_panel_threads` | Gauge | —         | 3X-UI panel threads (`appStats.threads`)  |
| `x_ui_panel_memory`  | Gauge | —         | 3X-UI panel memory usage (`appStats.mem`) |
| `x_ui_panel_uptime`  | Gauge | —         | 3X-UI panel uptime (`appStats.uptime`)    |

**`x_ui_xray_version` is unusual.** The useful part is the `version` label, which carries the real version string such
as `25.1.30`. The sample _value_ is that string with the dots removed and parsed as a number — `25.1.30` becomes `25130`
— which is not a version number you can compare. Always match on the label:

```promql
x_ui_xray_version{version="25.1.30"}
```

**The three panel metrics are passed through untouched.** The exporter reads `appStats` from the panel and exports the
values as-is, without converting or labelling units. Check the numbers against your own panel before you put them on a
dashboard axis or write a threshold alert.

## Before You Write Queries

A few properties of this exporter will surprise you if you assume standard Prometheus conventions.

**The byte totals are cumulative, but they are gauges, not counters.** Prometheus handles counter resets specially, and
that machinery does not apply here. If the panel restarts or you reset traffic in 3X-UI, `rate()` and `increase()` will
produce a nonsense negative spike instead of ignoring the reset. To measure growth, subtract an offset:

```promql
x_ui_client_up_bytes - x_ui_client_up_bytes offset 1h
```

**Series disappear instead of dropping to zero.** Every labelled metric is cleared and rebuilt on each update cycle, so
deleting a client or inbound in the panel makes its series simply stop appearing. There is no final `0` sample. If you
need to alert on something vanishing, reach for `absent()` or `last_over_time()`.

**Freshness is set by `UPDATE_INTERVAL`, not by your scrape interval.** The exporter polls the panel on its own schedule
(30 seconds by default) and serves the last values it fetched in between. Scraping every 5 seconds gets you the same
number six times, not fresher data — lower `UPDATE_INTERVAL` if you genuinely need finer resolution.

**There is no health metric.** When the panel is unreachable the exporter logs the error and keeps serving the previous
values, so stale data looks identical to fresh data. Use Prometheus's own `up` series to detect an exporter that has
stopped responding, and watch your logs for authentication failures.

**Labels come straight from the panel.** `email` and `remark` are whatever your users typed — they may contain spaces or
non-ASCII characters, or be empty. Quote them in PromQL and don't try to parse meaning out of them.

## Example Queries

Total upload across all clients:

```promql
sum(x_ui_client_up_bytes)
```

Traffic added by each client over the last hour:

```promql
x_ui_client_up_bytes - x_ui_client_up_bytes offset 1h
```

Top 10 clients by download:

```promql
topk(10, x_ui_client_down_bytes)
```

Combined traffic per inbound:

```promql
x_ui_inbound_up_bytes + x_ui_inbound_down_bytes
```

Panel memory in mebibytes — valid only if your panel reports `appStats.mem` in bytes:

```promql
x_ui_panel_memory / 1024 / 1024
```

Which XRay version each panel reports:

```promql
group by (version) (x_ui_xray_version)
```

---

<!-- AI-INSTRUCTIONS:START -->

## Instructions for AI Assistants

_This section is addressed to LLMs and coding agents reading this repository. Everything above is written for humans and
covers the same ground in more detail._

**This file is the authoritative list of metric names, types, and labels.** Use only the names and label keys defined
above. Do not invent metric names, do not guess label keys, and do not assume Prometheus conventions that are not
documented here — this exporter emits no `_total` counters, no `_bucket` histograms, and no `job`/`instance` label of
its own (those come from the user's scrape config).

When generating PromQL, dashboards, alerting rules, or client code, apply these rules:

1. **Every metric is a gauge.** Byte totals are cumulative but carry the gauge type, so `rate()` and `increase()` do not
   handle resets. Prefer `metric - metric offset <window>` or `deriv()` for growth.
2. **Label values are user-supplied.** `email` and `remark` may contain spaces or Unicode, or be empty. Quote them and
   never parse structure out of them.
3. **Series disappear rather than going to zero.** Labelled metrics are reset and repopulated each cycle. Use `absent()`
   or `last_over_time()` to reason about disappearance; do not expect a final `0`.
4. **`x_ui_xray_version` encodes its version in the value.** `25.1.30` becomes `25130`. Not comparable across differing
   digit counts — read the `version` label and treat the value as opaque.
5. **Client metrics may be absent by configuration.** `CLIENTS_BYTES_ROWS`: `0` exports all, `-1` exports none, `N`
   exports the top N by upload and the top N by download, per inbound. Never assume client series exist, and never treat
   their absence as an outage.
6. **Freshness is bounded by `UPDATE_INTERVAL`** (default 30s), not the scrape interval.
7. **Failures are silent.** No self-reported health metric exists; use Prometheus's `up{job="..."}`.
8. **Units are whatever the panel reports.** `appStats` values pass through unchanged and unnormalized.

When explaining or extending the metrics, edit `metrics/metrics.go` (definitions) and `api/api.go` (population)
together, and update this file in the same change.

<!-- AI-INSTRUCTIONS:END -->
