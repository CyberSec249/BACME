#!/usr/bin/env python3
"""
On-chain stages of domain control validation (DCV): comparison of BACME and the two baselines (same style as Fig. 9 of the paper)

Usage:
    python3 plot_dcv.py --throughput v2/throughput.csv --latency v2/latency.csv --nodes 10 --out figs

Output (two figures, each with two subplots: (a) stage 1, randomness request; (b) stage 2, start of HTTP validation):
    <out>/fig_dcv_throughput_rtt_n<N>.pdf/.png   y-axis: throughput of successful transactions (tx/s)
    <out>/fig_dcv_latency_rtt_n<N>.pdf/.png      y-axis: confirmation latency (ms), solid p50, dashed p95
x-axis: inter-node RTT (ms); three lines: BACME, Baseline 1, Baseline 2.
The latency figure excludes runs with suspected PBFT view changes by default (same rule as plot_latency.py); disable with --include-stalls.
"""
import argparse
import csv
import os
import statistics
from collections import defaultdict

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.lines import Line2D
from matplotlib.ticker import ScalarFormatter

plt.rcParams.update({
    "font.family": "serif",
    "font.serif": ["Times New Roman", "Times", "Liberation Serif", "DejaVu Serif"],
    "mathtext.fontset": "stix",
    "font.size": 8,
    "axes.labelsize": 8,
    "axes.linewidth": 0.6,
    "xtick.labelsize": 8,
    "ytick.labelsize": 8,
    "xtick.direction": "in",
    "ytick.direction": "in",
    "xtick.major.size": 2.5,
    "ytick.major.size": 2.5,
    "legend.fontsize": 6.5,
    "pdf.fonttype": 42,
    "ps.fonttype": 42,
})

# scheme -> (legend label, color, marker), consistent with the paper
SCHEMES = [
    ("bacme", "BACME", "#0072BD", "^"),
    ("dacme", "Baseline 1", "#9467BD", "s"),
    ("dcv", "Baseline 2", "#FF7F0E", "D"),
]
STAGES = [("v1", "Stage 1: randomness request"), ("v2", "Stage 2: HTTP validation request")]


def read_rows(path, nodes):
    rows = []
    if not path or not os.path.exists(path):
        return rows
    with open(path, newline="", encoding="utf-8") as f:
        for r in csv.DictReader(f):
            if r["total"] != "FAILED" and r.get("tps_ok") and int(r["nodes"]) == nodes \
                    and r["op"].endswith(("_v1", "_v2")):
                rows.append(r)
    return rows


def group(rows):
    g = defaultdict(list)
    for r in rows:
        scheme, stage = r["op"].rsplit("_", 1)
        g[(scheme, stage, int(r["rtt_target_ms"]))].append(r)
    return g


def style_axis(ax, rtts):
    ax.grid(True, color="#D9D9D9", linewidth=0.5)
    ax.set_axisbelow(True)
    ax.tick_params(top=False, right=False)
    fmt = ScalarFormatter(useMathText=True)
    fmt.set_powerlimits((0, 0))
    ax.yaxis.set_major_formatter(fmt)
    ax.yaxis.get_offset_text().set_fontsize(8)
    step = 50 if max(rtts) >= 150 else 20
    ax.set_xticks(list(range(0, int(max(rtts)) + 1, step)))
    ax.set_xlim(min(rtts) - 12, max(rtts) + 12)
    ax.set_xlabel("Inter-node RTT (ms)", labelpad=1)


def finish(fig, axes, out, name):
    for idx, ax in enumerate(axes):
        ax.text(0.5, -0.31, f"({chr(ord('a') + idx)})", transform=ax.transAxes,
                ha="center", va="top", fontsize=9)
    fig.tight_layout(pad=0.3, w_pad=1.0)
    fig.subplots_adjust(bottom=0.27)
    os.makedirs(out, exist_ok=True)
    base = os.path.join(out, name)
    fig.savefig(base + ".pdf", bbox_inches="tight")
    fig.savefig(base + ".png", dpi=300, bbox_inches="tight")
    print(f"Saved: {base}.pdf / .png")


def plot_throughput(g, nodes, out, errorbar, exclude=True):
    rtts = sorted({k[2] for k in g})
    fig, axes = plt.subplots(1, 2, figsize=(3.6, 1.9))
    print(f"\nThroughput (successful TPS, N={nodes})")
    for ax, (stage, title) in zip(axes, STAGES):
        for scheme, label, color, marker in SCHEMES:
            pts = []
            for r in rtts:
                rows = g.get((scheme, stage, r), [])
                if not rows:
                    continue
                med = statistics.median(float(x["time_ms"]) for x in rows)
                vals = [float(x["tps_ok"]) for x in rows
                        if not (exclude and float(x["time_ms"]) >= med + 2000)]
                pts.append((r, vals or [float(x["tps_ok"]) for x in rows]))
            if not pts:
                continue
            xs = [p[0] for p in pts]
            ys = [statistics.mean(p[1]) for p in pts]
            es = [statistics.stdev(p[1]) if len(p[1]) > 1 else 0.0 for p in pts]
            if errorbar:
                ax.errorbar(xs, ys, yerr=es, fmt="none", ecolor=color, capsize=1.8,
                            elinewidth=0.7, capthick=0.7)
            ax.plot(xs, ys, color=color, marker=marker, markersize=4.5, markerfacecolor="white",
                    markeredgewidth=1.0, linewidth=1.2, label=label)
            print(f"  {stage} {label:<11}" + "".join(f"{x:>6}ms:{y:>8.1f}" for x, y in zip(xs, ys)))
        style_axis(ax, rtts)
        ax.set_ylabel("Throughput (tx/s)", labelpad=1)
        lo, hi = ax.get_ylim()
        ax.set_ylim(0, hi * 1.3)
        ax.legend(loc="upper right", frameon=True, fancybox=False, edgecolor="black",
                  framealpha=1, handlelength=1.6, borderpad=0.3, labelspacing=0.2)
    finish(fig, axes, out, f"fig_dcv_throughput_rtt_n{nodes}")


def plot_latency(g, nodes, out, include_stalls):
    rtts = sorted({k[2] for k in g})
    fig, axes = plt.subplots(1, 2, figsize=(3.6, 1.9))
    excluded = []
    print(f"\nConfirmation latency (p50 / p95 ms, N={nodes})")
    for ax, (stage, title) in zip(axes, STAGES):
        handles = []
        for scheme, label, color, marker in SCHEMES:
            pts = []
            for r in rtts:
                rows = g.get((scheme, stage, r), [])
                if not rows:
                    continue
                kept = []
                for x in rows:
                    p50, p95 = float(x["p50_ms"]), float(x["p95_ms"])
                    if not include_stalls and p95 > 3 * p50 and p50 > 50:
                        excluded.append((scheme, stage, r, x["run"], p50, p95))
                    else:
                        kept.append(x)
                if kept:
                    pts.append((r, statistics.mean(float(x["p50_ms"]) for x in kept),
                                statistics.mean(float(x["p95_ms"]) for x in kept)))
            if not pts:
                continue
            xs = [p[0] for p in pts]
            for idx, ls in ((1, "-"), (2, "--")):
                ax.plot(xs, [p[idx] for p in pts], color=color, marker=marker, markersize=4.5,
                        markerfacecolor="white", markeredgewidth=1.0, linewidth=1.2, linestyle=ls)
            handles.append(Line2D([], [], color=color, marker=marker, markersize=4.5,
                                  markerfacecolor="white", linewidth=1.2, label=label))
            print(f"  {stage} {label:<11}" + "".join(f"{p[0]:>6}ms:{p[1]:>6.0f}/{p[2]:<6.0f}" for p in pts))
        handles += [Line2D([], [], color="black", linewidth=1.0, linestyle="-", label="p50"),
                    Line2D([], [], color="black", linewidth=1.0, linestyle="--", label="p95")]
        style_axis(ax, rtts)
        ax.set_ylabel("Latency (ms)", labelpad=1)
        lo, hi = ax.get_ylim()
        ax.set_ylim(0, hi * 1.6)
        ax.legend(handles=handles, loc="upper left", frameon=True, fancybox=False,
                  edgecolor="black", framealpha=1, handlelength=1.6, borderpad=0.3, labelspacing=0.2)
    finish(fig, axes, out, f"fig_dcv_latency_rtt_n{nodes}")
    if excluded:
        print(f"Excluded {len(excluded)} runs with suspected view changes:")
        for e in excluded:
            print(f"  {e[0]}_{e[1]} RTT={e[2]}ms run {e[3]}: p50={e[4]:.0f} p95={e[5]:.0f}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--throughput", default="throughput.csv")
    ap.add_argument("--latency", default="latency.csv")
    ap.add_argument("--nodes", type=int, default=10)
    ap.add_argument("--out", default=".")
    ap.add_argument("--no-errorbar", action="store_true")
    ap.add_argument("--include-stalls", action="store_true")
    args = ap.parse_args()

    tg = group(read_rows(args.throughput, args.nodes))
    lg = group(read_rows(args.latency, args.nodes))
    if not tg and not lg:
        print(f"No DCV data found for N={args.nodes} (operation names should be bacme_v1 / dacme_v1 / dcv_v1 / *_v2)")
        return
    if tg:
        plot_throughput(tg, args.nodes, args.out, not args.no_errorbar, not args.include_stalls)
    if lg:
        plot_latency(lg, args.nodes, args.out, args.include_stalls)
    print("\nSuggested caption: On-chain performance of domain control validation under different inter-node RTTs "
          f"(N = {args.nodes}): (a) Stage 1 (randomness request); (b) Stage 2 (HTTP validation request). "
          "Oracle-side processing is identical for all schemes and is excluded.")


if __name__ == "__main__":
    main()
