#!/usr/bin/env python3
"""
Summarize the results produced by run_exp.sh: group by (nodes, RTT, operation) and compute the mean and standard deviation of each metric.

Usage:
    python3 summarize.py                         # reads ~/exp-results/results.csv by default
    python3 summarize.py results.csv -o summary.csv
"""
import argparse
import csv
import os
import statistics
from collections import defaultdict

METRICS = ["tps_ok", "tps_all", "avg_ms", "p50_ms", "p95_ms", "p99_ms", "err_rate_pct", "blocks"]


def mean_std(values):
    if not values:
        return "", ""
    m = statistics.mean(values)
    s = statistics.stdev(values) if len(values) > 1 else 0.0
    return round(m, 2), round(s, 2)


def main():
    default_in = os.path.expanduser("~/exp-results/results.csv")
    ap = argparse.ArgumentParser()
    ap.add_argument("input", nargs="?", default=default_in)
    ap.add_argument("-o", "--output", default=None)
    args = ap.parse_args()
    out = args.output or os.path.join(os.path.dirname(args.input) or ".", "summary.csv")

    groups = defaultdict(lambda: defaultdict(list))
    failed = defaultdict(int)
    measured_rtt = defaultdict(list)
    with open(args.input, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f):
            key = (int(row["nodes"]), int(row["rtt_target_ms"]), row["op"])
            if row["total"] == "FAILED":
                failed[key] += 1
                continue
            for m in METRICS:
                if row.get(m, "") not in ("", None):
                    groups[key][m].append(float(row[m]))
            if row.get("rtt_measured_ms"):
                measured_rtt[key].append(float(row["rtt_measured_ms"]))

    keys = sorted(set(groups) | set(failed))
    header = ["nodes", "f", "rtt_target_ms", "rtt_measured_ms", "op", "runs_ok", "runs_failed"]
    for m in METRICS:
        header += [m + "_mean", m + "_std"]

    rows = []
    for key in keys:
        nodes, rtt, op = key
        g = groups.get(key, {})
        runs_ok = len(g.get("tps_ok", []))
        mr = measured_rtt.get(key, [])
        row = [nodes, (nodes - 1) // 3, rtt, round(statistics.mean(mr), 3) if mr else "", op,
               runs_ok, failed.get(key, 0)]
        for m in METRICS:
            row += list(mean_std(g.get(m, [])))
        rows.append(row)

    with open(out, "w", newline="", encoding="utf-8") as f:
        w = csv.writer(f)
        w.writerow(header)
        w.writerows(rows)

    # Print a compact table to the terminal
    print(f"{'nodes':>5} {'f':>2} {'RTT':>5} {'op':<9} {'runs':>4} "
          f"{'TPS(ok)':>16} {'p50(ms)':>14} {'p95(ms)':>14} {'err%':>7}")
    idx = {h: i for i, h in enumerate(header)}
    for r in rows:
        def ms(name):
            m, s = r[idx[name + "_mean"]], r[idx[name + "_std"]]
            return f"{m}±{s}" if m != "" else "-"
        err = r[idx["err_rate_pct_mean"]]
        print(f"{r[0]:>5} {r[1]:>2} {r[2]:>5} {r[4]:<9} {r[5]:>4} "
              f"{ms('tps_ok'):>16} {ms('p50_ms'):>14} {ms('p95_ms'):>14} {err if err != '' else '-':>7}")
    print(f"\nSummary written to: {out}")


if __name__ == "__main__":
    main()
