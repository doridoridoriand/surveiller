#!/usr/bin/env python3
"""Generate grafana/dashboards/surveiller.json from the real metric set.

Real metrics exported by internal/metrics/metrics.go (verified against source):
  aggregated: surveiller_targets_{total,ok,warn,down,unknown}
  per-target: surveiller_target_up{target,address,group},
              surveiller_target_rtt_ms{target,address,group}
"""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "grafana" / "dashboards" / "surveiller.json"

DS = {"type": "prometheus", "uid": "${DS_PROMETHEUS}"}

PANEL_ID = [0]


def pid():
    PANEL_ID[0] += 1
    return PANEL_ID[0]


def expr_agg(name):
    return f"sum({name})"


def stat(title, expr, color, unit="short", decimals=0, thresholds=None, span=2):
    th = thresholds or [{"color": color, "value": None}]
    return {
        "id": pid(),
        "type": "stat",
        "title": title,
        "datasource": DS,
        "gridPos": {"h": 4, "w": span, "x": 0, "y": 0},  # layout fixed later
        "targets": [{"expr": expr, "legendFormat": title, "refId": "A"}],
        "options": {
            "colorMode": "background",
            "graphMode": "none",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {"calcs": ["lastNotNull"], "fields": "", "values": False},
            "textMode": "auto",
            "wideLayout": True,
        },
        "fieldConfig": {
            "defaults": {
                "color": {"mode": "fixed", "color": color},
                "mappings": [],
                "thresholds": {"mode": "absolute", "steps": th},
                "unit": unit,
                "decimals": decimals,
            },
            "overrides": [],
        },
    }


def panel_ts_rtt():
    return {
        "id": pid(),
        "type": "timeseries",
        "title": "RTT by target (ms)",
        "datasource": DS,
        "targets": [
            {
                "expr": 'surveiller_target_rtt_ms{instance=~"${instance}", target=~"${target}", group=~"${group}"}',
                "legendFormat": "{{target}}",
                "refId": "A",
            }
        ],
        "fieldConfig": {
            "defaults": {
                "unit": "ms",
                "custom": {
                    "drawStyle": "line",
                    "lineWidth": 2,
                    "fillOpacity": 8,
                    "showPoints": "never",
                    "spanNulls": True,
                    "pointSize": 5,
                    "axisPlacement": "auto",
                    "stacking": {"mode": "none"},
                    "thresholdsStyle": {"mode": "off"},
                },
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
                "decimals": 0,
            },
            "overrides": [],
        },
        "options": {
            "legend": {"displayMode": "table", "placement": "right", "calcs": ["lastNotNull", "max", "mean"], "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"},
            "tooltipOptions": {"sort": "desc"},
        },
    }


def panel_group_rtt():
    return {
        "id": pid(),
        "type": "barchart",
        "title": "Avg RTT by group (current)",
        "datasource": DS,
        "targets": [
            {
                "expr": 'avg by (group) (surveiller_target_rtt_ms{instance=~"${instance}", target=~"${target}", group=~"${group}"})',
                "legendFormat": "{{group}}",
                "refId": "A",
            }
        ],
        "fieldConfig": {
            "defaults": {
                "unit": "ms",
                "color": {"mode": "palette-classic"},
                "custom": {
                    "axisPlacement": "auto",
                    "fillOpacity": 80,
                    "gradientMode": "none",
                    "lineWidth": 1,
                    "fillSize": 1,
                },
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
            },
            "overrides": [],
        },
        "options": {
            "orientation": "horizontal",
            "barRadius": 0.1,
            "barWidth": 0.8,
            "groupWidth": 0.7,
            "showValue": "auto",
            "legend": {"displayMode": "list", "placement": "right", "showLegend": False},
            "tooltip": {"mode": "single", "sort": "desc"},
            "xTickLabelRotation": 0,
            "xTickLabelSpacing": 0,
        },
    }


def panel_availability():
    return {
        "id": pid(),
        "type": "barchart",
        "title": "Availability by target (up fraction, last 5m) — lowest first",
        "description": "Approximation: mean of the surveiller_target_up gauge over the window (no success/failure counters are exported yet).",
        "datasource": DS,
        "targets": [
            {
                "expr": 'avg_over_time(surveiller_target_up{instance=~"${instance}", target=~"${target}", group=~"${group}"}[5m])',
                "legendFormat": "{{target}}",
                "refId": "A",
            }
        ],
        "fieldConfig": {
            "defaults": {
                "unit": "percentunit",
                "max": 1,
                "min": 0,
                "color": {"mode": "thresholds"},
                "custom": {
                    "axisPlacement": "auto",
                    "fillOpacity": 90,
                    "gradientMode": "none",
                    "lineWidth": 1,
                    "fillSize": 1,
                },
                "thresholds": {
                    "mode": "absolute",
                    "steps": [
                        {"color": "red", "value": None},
                        {"color": "orange", "value": 0.95},
                        {"color": "green", "value": 1},
                    ],
                },
                "decimals": 2,
            },
            "overrides": [],
        },
        "options": {
            "orientation": "horizontal",
            "barRadius": 0.1,
            "barWidth": 0.8,
            "groupWidth": 0.7,
            "showValue": "auto",
            "legend": {"displayMode": "list", "placement": "right", "showLegend": False},
            "tooltip": {"mode": "single", "sort": "asc"},
            "xTickLabelRotation": 0,
            "xTickLabelSpacing": 0,
        },
    }


def panel_status_timeseries():
    return {
        "id": pid(),
        "type": "timeseries",
        "title": "Target status over time",
        "datasource": DS,
        "targets": [
            {
                "expr": 'sum(surveiller_target_up{instance=~"${instance}", target=~"${target}", group=~"${group}"})',
                "legendFormat": "UP",
                "refId": "A",
            },
            {
                "expr": "sum(surveiller_targets_warn)",
                "legendFormat": "WARN",
                "refId": "B",
            },
            {
                "expr": "sum(surveiller_targets_down)",
                "legendFormat": "DOWN",
                "refId": "C",
            },
        ],
        "fieldConfig": {
            "defaults": {
                "unit": "short",
                "decimals": 0,
                "custom": {
                    "drawStyle": "line",
                    "lineWidth": 2,
                    "fillOpacity": 15,
                    "showPoints": "never",
                    "spanNulls": True,
                    "pointSize": 5,
                    "axisPlacement": "auto",
                    "stacking": {"mode": "none"},
                    "thresholdsStyle": {"mode": "off"},
                },
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
            },
            "overrides": [
                {"matcher": {"id": "byName", "options": "DOWN"}, "properties": [{"id": "color", "value": {"mode": "fixed", "color": "red"}}]},
                {"matcher": {"id": "byName", "options": "WARN"}, "properties": [{"id": "color", "value": {"mode": "fixed", "color": "orange"}}]},
            ],
        },
        "options": {
            "legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"},
        },
    }


def panel_status_stacked():
    return {
        "id": pid(),
        "type": "timeseries",
        "title": "Status composition (stacked)",
        "datasource": DS,
        "targets": [
            {"expr": "sum(surveiller_targets_ok)", "legendFormat": "OK", "refId": "A"},
            {"expr": "sum(surveiller_targets_warn)", "legendFormat": "WARN", "refId": "B"},
            {"expr": "sum(surveiller_targets_down)", "legendFormat": "DOWN", "refId": "C"},
            {"expr": "sum(surveiller_targets_unknown)", "legendFormat": "UNKNOWN", "refId": "D"},
        ],
        "fieldConfig": {
            "defaults": {
                "unit": "short",
                "decimals": 0,
                "color": {"mode": "palette-classic"},
                "custom": {
                    "drawStyle": "line",
                    "lineWidth": 1,
                    "fillOpacity": 80,
                    "showPoints": "never",
                    "spanNulls": True,
                    "pointSize": 5,
                    "axisPlacement": "auto",
                    "stacking": {"mode": "normal", "group": "A"},
                    "thresholdsStyle": {"mode": "off"},
                },
                "thresholds": {"mode": "absolute", "steps": [{"color": "green", "value": None}]},
            },
            "overrides": [],
        },
        "options": {
            "legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
            "tooltip": {"mode": "multi", "sort": "desc"},
        },
    }


panels = [
    stat("Targets (total)", expr_agg("surveiller_targets_total"), "blue"),
    stat("Targets OK", expr_agg("surveiller_targets_ok"), "green"),
    stat("Targets WARN", expr_agg("surveiller_targets_warn"), "orange", thresholds=[{"color": "blue", "value": None}, {"color": "orange", "value": 1}]),
    stat("Targets DOWN", expr_agg("surveiller_targets_down"), "red", thresholds=[{"color": "green", "value": None}, {"color": "red", "value": 1}]),
    stat("Targets UNKNOWN", expr_agg("surveiller_targets_unknown"), "purple"),
    stat("Up ratio", "sum(surveiller_target_up{instance=~\"${instance}\", target=~\"${target}\", group=~\"${group}\"}) / sum(surveiller_targets_total)", "green", unit="percentunit", decimals=1,
         thresholds=[{"color": "red", "value": None}, {"color": "orange", "value": 0.9}, {"color": "green", "value": 1}]),
    panel_ts_rtt(),
    panel_group_rtt(),
    panel_availability(),
    panel_status_timeseries(),
    panel_status_stacked(),
]

# --- layout: stat row on top (6 x w=4), then big panels (full width / half+half) ---
x = 0
for p in panels[:6]:
    p["gridPos"] = {"h": 4, "w": 4, "x": x, "y": 0}
    x += 4
panels[6]["gridPos"] = {"h": 8, "w": 24, "x": 0, "y": 4}  # rtt timeseries
panels[7]["gridPos"] = {"h": 8, "w": 12, "x": 0, "y": 12}  # group rtt
panels[8]["gridPos"] = {"h": 8, "w": 12, "x": 12, "y": 12}  # availability
panels[9]["gridPos"] = {"h": 8, "w": 12, "x": 0, "y": 20}  # status timeseries
panels[10]["gridPos"] = {"h": 8, "w": 12, "x": 12, "y": 20}  # status stacked

dashboard = {
    "annotations": {"list": []},
    "description": "Surveiller ping monitor — Prometheus metrics from surveiller (internal/metrics). Works with metrics.mode=both (recommended); per-target panels need per-target mode, stat panels need aggregated mode.",
    "editable": True,
    "fiscalYearStartMonth": 0,
    "graphTooltip": 1,
    "id": None,
    "links": [],
    "liveNow": False,
    "panels": panels,
    "refresh": "10s",
    "schemaVersion": 39,
    "tags": ["surveiller", "ping", "prometheus"],
    "templating": {
        "list": [
            {
                "name": "DS_PROMETHEUS",
                "label": "Prometheus",
                "type": "datasource",
                "query": "prometheus",
                "refresh": 1,
                "hide": 0,
                "current": {},
                "options": [],
                "regex": "",
            },
            {
                "name": "instance",
                "label": "Instance",
                "type": "query",
                "datasource": DS,
                "query": "label_values(surveiller_targets_total, instance)",
                "refresh": 2,
                "hide": 0,
                "includeAll": True,
                "multi": True,
                "current": {"text": "All", "value": "$__all"},
                "options": [],
                "regex": "",
                "sort": 1,
            },
            {
                "name": "target",
                "label": "Target",
                "type": "query",
                "datasource": DS,
                "query": "label_values(surveiller_target_up, target)",
                "refresh": 2,
                "hide": 0,
                "includeAll": True,
                "allValue": ".*",
                "multi": True,
                "current": {"text": "All", "value": "$__all"},
                "options": [],
                "regex": "",
                "sort": 1,
            },
            {
                "name": "group",
                "label": "Group",
                "type": "query",
                "datasource": DS,
                "query": "label_values(surveiller_target_up, group)",
                "refresh": 2,
                "hide": 0,
                "includeAll": True,
                "allValue": ".*",
                "multi": True,
                "current": {"text": "All", "value": "$__all"},
                "options": [],
                "regex": "",
                "sort": 1,
            },
        ]
    },
    "time": {"from": "now-1h", "to": "now"},
    "timepicker": {},
    "timezone": "browser",
    "title": "Surveiller / Ping Monitor",
    "uid": "surveiller",
    "version": 1,
    "weekStart": "",
}

OUT.parent.mkdir(parents=True, exist_ok=True)
OUT.write_text(json.dumps(dashboard, indent=2, ensure_ascii=False) + "\n")
print(f"wrote {OUT} ({OUT.stat().st_size} bytes), {len(panels)} panels")
