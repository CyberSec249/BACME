#!/usr/bin/env python3
"""
Confirmation latency vs. inter-node RTT (same style as Fig. 9 / Fig. 12 of the paper)

Usage:
    python3 plot_latency.py                                     # reads ./latency.csv by default
    python3 plot_latency.py --csv v2/latency.csv --out figs
    python3 plot_latency.py --include-stalls                    # do not exclude runs with suspected view changes
    python3 plot_latency.py --style lines                       # legacy: single row, solid p50 + dashed p95

--style errorbar outputs <out>/fig_latency_rtt_errorbar.pdf/.png:
top row p50, bottom row p95; each point is the mean ± standard deviation over runs, with different N slightly offset horizontally to avoid overlap.

Output: <out>/fig_latency_rtt.pdf and .png
x-axis: inter-node RTT (ms); y-axis: confirmation latency (ms). Solid = p50, dashed = p95;
one color per number of consensus nodes N.

Runs with a suspected PBFT view change (block production stalls for about consensus_timeout = 3 s) are excluded by default if either condition holds:
  1) p95 > 3 x p50 in the run (and p50 > 50 ms)
  2) for write operations, the number of blocks in the run <= 75% of the group maximum (a ~3 s stall produces several fewer blocks)
Excluded runs are printed, and their number should be reported in the paper.
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

# ---------------- Paper figure style (MATLAB default style) ----------------
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

SERIES_STYLE = {
    10: ("#0072BD", "^"),   # blue triangle
    7: ("#9467BD", "s"),    # purple square
    4: ("#FF7F0E", "D"),    # orange diamond
}
FALLBACK_STYLES = [("#17BECF", "v"), ("#D62728", "o"), ("#2CA02C", "p")]

OPS = [
    ("bacme_v1", "Latency (ms)"),       # domain control validation (on-chain stage 1: controlValidate)
    ("validation", "Latency (ms)"),
    ("apply", "Latency (ms)"),
    ("update", "Latency (ms)"),
    ("revoke", "Latency (ms)"),
    ("revokeDV", "Latency (ms)"),
    ("revokeP", "Latency (ms)"),
    ("verify", "Query latency (ms)"),
]
OP_NAMES = {
    "bacme_v1": "Domain control validation",
    "validation": "Domain control validation",
    "apply": "Certificate application",
    "update": "Certificate renewal",
    "revoke": "Certificate revocation (management account)",
    "revokeDV": "Certificate revocation (domain control)",
    "revokeP": "Certificate revocation (private key proof)",
    "verify": "Certificate verification",
}


# Subplot titles (shown below each subplot, e.g., "(a) Domain control validation")
SUBPLOT_TITLES = {
    "bacme_v1": "Domain control validation",
    "validation": "Domain control validation",
    "apply": "Certificate application",
    "update": "Certificate renewal",
    "revoke": "Certificate revocation",
    "revokeDV": "Revocation (domain control)",
    "revokeP": "Revocation (private key)",
    "verify": "Certificate verification",
}


def load(path, include_stalls):
    groups = defaultdict(list)   # (op, nodes, rtt) -> [row, ...]
    with open(path, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f):
            if row["total"] == "FAILED" or not row.get("p50_ms"):
                continue
            groups[(row["op"], int(row["nodes"]), int(row["rtt_target_ms"]))].append(row)

    data, excluded = {}, []
    for key, rows in groups.items():
        max_blocks = max(int(r["blocks"]) for r in rows)
        kept = []
        for r in rows:
            p50, p95, blocks = float(r["p50_ms"]), float(r["p95_ms"]), int(r["blocks"])
            stall = (p95 > 3 * p50 and p50 > 50) or \
                    (key[0] != "verify" and max_blocks > 0 and blocks <= 0.75 * max_blocks)
            if stall and not include_stalls:
                excluded.append((key, r["run"], p50, p95, blocks))
            else:
                kept.append(r)
        if kept:
            p50s = [float(r["p50_ms"]) for r in kept]
            p95s = [float(r["p95_ms"]) for r in kept]
            data[key] = {
                "p50": statistics.mean(p50s),
                "p95": statistics.mean(p95s),
                "p50_sd": statistics.stdev(p50s) if len(p50s) > 1 else 0.0,
                "p95_sd": statistics.stdev(p95s) if len(p95s) > 1 else 0.0,
                "runs": len(kept),
                "total": len(rows),
            }
    return data, excluded


def style_axis(ax, use_offset):
    ax.grid(True, color="#D9D9D9", linewidth=0.5)
    ax.set_axisbelow(True)
    ax.tick_params(top=False, right=False)
    if use_offset:
        fmt = ScalarFormatter(useMathText=True)
        fmt.set_powerlimits((0, 0))
        ax.yaxis.set_major_formatter(fmt)
        ax.yaxis.get_offset_text().set_fontsize(8)


# Error-bar version: two rows of subplots (top p50, bottom p95); each point = mean ± standard deviation over 5 runs;
# the three node counts are offset horizontally by DODGE ms so error bars and markers do not overlap.
DODGE = {10: -4.5, 7: 0.0, 4: 4.5}


def plot_errorbar(data, ops, nodes_all, rtts, out):
    n = len(ops)
    fig, axes = plt.subplots(2, n, figsize=(1.6 * n + 0.2, 3.5), squeeze=False)
    metrics = [("p50", "p50 latency (ms)", "p50 query latency (ms)"),
               ("p95", "p95 latency (ms)", "p95 query latency (ms)")]
    print(f"{'Operation':<10}{'N':>4}{'Metric':>7}" + "".join(f"{'RTT ' + str(r):>16}" for r in rtts) + "   (mean ± std, ms)")
    for row, (metric, ylab, ylab_q) in enumerate(metrics):
        for idx, (op, _) in enumerate(ops):
            ax = axes[row][idx]
            for j, nodes in enumerate(nodes_all):
                color, marker = SERIES_STYLE.get(nodes, FALLBACK_STYLES[j % len(FALLBACK_STYLES)])
                pts = [(r, data[(op, nodes, r)]) for r in rtts if (op, nodes, r) in data]
                if not pts:
                    continue
                dx = DODGE.get(nodes, 0.0)
                xs = [p[0] + dx for p in pts]
                ys = [p[1][metric] for p in pts]
                es = [p[1][metric + "_sd"] for p in pts]
                ax.errorbar(xs, ys, yerr=es, fmt="none", ecolor=color, capsize=1.8,
                            elinewidth=0.7, capthick=0.7)
                ax.plot(xs, ys, color=color, marker=marker, markersize=4.0, markerfacecolor="white",
                        markeredgewidth=1.0, linewidth=1.1, label=f"BACME (N={nodes})")
                print(f"{op:<10}{nodes:>4}{metric:>5}" + "".join(f"{y:>9.0f}±{e:<6.0f}" for y, e in zip(ys, es)))
            style_axis(ax, use_offset=(op != "verify"))
            step = 50 if max(rtts) >= 150 else 20
            ax.set_xticks(list(range(0, int(max(rtts)) + 1, step)))
            ax.set_xlim(min(rtts) - 14, max(rtts) + 14)
            ax.set_ylabel(ylab_q if op == "verify" else ylab, labelpad=1)
            lo, hi = ax.get_ylim()
            ax.set_ylim(0, hi * (1.9 if op == "verify" else 1.5))
            ax.legend(loc="upper left", frameon=True, fancybox=False, edgecolor="black",
                      framealpha=1, handlelength=1.6, borderpad=0.3, labelspacing=0.2)
            if row == 1:
                ax.set_xlabel("Inter-node RTT (ms)", labelpad=1)
                ax.text(0.5, -0.36, f"({chr(ord('a') + idx)}) {SUBPLOT_TITLES.get(op, op)}",
                        transform=ax.transAxes, ha="center", va="top", fontsize=8.5, fontweight="bold")
            else:
                ax.tick_params(labelbottom=True)
    fig.tight_layout(pad=0.3, w_pad=0.8, h_pad=0.9)
    fig.subplots_adjust(bottom=0.15)
    os.makedirs(out, exist_ok=True)
    base = os.path.join(out, "fig_latency_rtt_errorbar")
    fig.savefig(base + ".pdf", bbox_inches="tight")
    fig.savefig(base + ".png", dpi=300, bbox_inches="tight")
    return base


# Grouped bar chart: x-axis is the RTT level (evenly spaced categories), with 3 bars per group for N=10/7/4;
# the solid part ends at p50, and the light hatched part on top extends to p95 (the tail-latency increment from p50 to p95).
from matplotlib.patches import Patch


def plot_bar(data, ops, nodes_all, rtts, out):
    n = len(ops)
    fig, axes = plt.subplots(1, n, figsize=(1.75 * n + 0.2, 2.0), squeeze=False)
    axes = axes[0]
    k = len(nodes_all)
    width = 0.8 / k
    print(f"{'Operation':<10}{'N':>4}" + "".join(f"{'RTT ' + str(r):>16}" for r in rtts) + "   (p50 / p95, ms)")
    for idx, (ax, (op, ylab)) in enumerate(zip(axes, ops)):
        for j, nodes in enumerate(nodes_all):
            color, _ = SERIES_STYLE.get(nodes, FALLBACK_STYLES[j % len(FALLBACK_STYLES)])
            xs, p50, p95 = [], [], []
            for i, r in enumerate(rtts):
                v = data.get((op, nodes, r))
                if v:
                    xs.append(i + (j - (k - 1) / 2) * width)
                    p50.append(v["p50"]); p95.append(v["p95"])
            if not xs:
                continue
            ax.bar(xs, p50, width=width, color=color, edgecolor="black", linewidth=0.4, zorder=3)
            ax.bar(xs, [b - a for a, b in zip(p50, p95)], bottom=p50, width=width,
                   color=color, alpha=0.35, hatch="////", edgecolor="black", linewidth=0.4, zorder=3)
            print(f"{op:<10}{nodes:>4}" + "".join(f"{a:>9.0f} / {b:<5.0f}" for a, b in zip(p50, p95)))
        style_axis(ax, use_offset=(op != "verify"))
        ax.grid(True, axis="y", color="#D9D9D9", linewidth=0.5)
        ax.grid(False, axis="x")
        ax.set_xticks(range(len(rtts)))
        ax.set_xticklabels([str(r) for r in rtts])
        ax.set_xlim(-0.6, len(rtts) - 0.4)
        ax.set_xlabel("Inter-node RTT (ms)", labelpad=1)
        ax.set_ylabel(ylab, labelpad=1)
        lo, hi = ax.get_ylim()
        ax.set_ylim(0, hi * 2.0)
        handles = [Patch(facecolor=SERIES_STYLE.get(nd, FALLBACK_STYLES[0])[0], edgecolor="black",
                         linewidth=0.4, label=f"BACME (N={nd})") for nd in nodes_all]
        handles += [Patch(facecolor="#BBBBBB", edgecolor="black", linewidth=0.4, label="Solid top: p50"),
                    Patch(facecolor="#BBBBBB", alpha=0.35, hatch="////", edgecolor="black",
                          linewidth=0.4, label="Hatched top: p95")]
        ax.legend(handles=handles, loc="upper left", ncol=1, frameon=True, fancybox=False,
                  edgecolor="black", framealpha=1, handlelength=1.2, handleheight=0.8,
                  borderpad=0.3, labelspacing=0.2, columnspacing=0.6, fontsize=5.8)
        ax.text(0.5, -0.31, f"({chr(ord('a') + idx)}) {SUBPLOT_TITLES.get(op, op)}",
                transform=ax.transAxes, ha="center", va="top", fontsize=8.5, fontweight="bold")
    fig.tight_layout(pad=0.3, w_pad=0.8)
    fig.subplots_adjust(bottom=0.27)
    os.makedirs(out, exist_ok=True)
    base = os.path.join(out, "fig_latency_rtt_bar")
    fig.savefig(base + ".pdf", bbox_inches="tight")
    fig.savefig(base + ".png", dpi=300, bbox_inches="tight")
    return base


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--csv", default="latency.csv")
    ap.add_argument("--out", default=".")
    ap.add_argument("--include-stalls", action="store_true")
    ap.add_argument("--style", choices=["bar", "errorbar", "lines"], default="bar",
                    help="bar (default): grouped bars, solid=p50, hatched=p95; errorbar: two rows with error bars; lines: single row with solid/dashed lines")
    args = ap.parse_args()

    data, excluded = load(args.csv, args.include_stalls)
    ops = [(op, ylab) for op, ylab in OPS if any(k[0] == op for k in data)]
    nodes_all = sorted({k[1] for k in data}, reverse=True)
    rtts = sorted({k[2] for k in data})

    if args.style == "bar":
        base = plot_bar(data, ops, nodes_all, rtts, args.out)
        print(f"\nSaved: {base}.pdf / .png")
        if excluded:
            print(f"\nExcluded {len(excluded)} runs with suspected view changes")
        print("Suggested caption: Confirmation latency of certificate management operations under different "
              "inter-node RTTs at a fixed sending rate (solid: p50; hatched: p95): " +
              "; ".join(f"({chr(ord('a') + i)}) {OP_NAMES[op]}" for i, (op, _) in enumerate(ops)) + ".")
        return
    if args.style == "errorbar":
        base = plot_errorbar(data, ops, nodes_all, rtts, args.out)
        print(f"\nSaved: {base}.pdf / .png")
        if excluded:
            print(f"\nExcluded {len(excluded)} runs with suspected view changes:")
            for (op, nodes, rtt), run, p50, p95, blocks in excluded:
                print(f"  N={nodes} RTT={rtt}ms {op} run {run}: p50={p50:.0f} p95={p95:.0f} blocks={blocks}")
        print("Suggested caption: Confirmation latency of certificate management operations under different "
              "inter-node RTTs at a fixed sending rate (top: p50, bottom: p95; mean ± standard deviation "
              "over repeated runs; points of different N are slightly offset horizontally for readability): " +
              "; ".join(f"({chr(ord('a') + i)}) {OP_NAMES[op]}" for i, (op, _) in enumerate(ops)) + ".")
        return

    n = len(ops)
    fig, axes = plt.subplots(1, n, figsize=(1.6 * n + 0.2, 1.9))
    if n == 1:
        axes = [axes]

    print(f"{'Operation':<10}{'N':>4}" + "".join(f"{'RTT ' + str(r):>16}" for r in rtts) + "   (p50 / p95, ms)")
    for idx, (ax, (op, ylab)) in enumerate(zip(axes, ops)):
        handles = []
        for j, nodes in enumerate(nodes_all):
            color, marker = SERIES_STYLE.get(nodes, FALLBACK_STYLES[j % len(FALLBACK_STYLES)])
            pts = [(r, data[(op, nodes, r)]) for r in rtts if (op, nodes, r) in data]
            if not pts:
                continue
            xs = [p[0] for p in pts]
            ax.plot(xs, [p[1]["p50"] for p in pts], color=color, marker=marker, markersize=4.5,
                    markerfacecolor="white", markeredgewidth=1.0, linewidth=1.2, linestyle="-")
            ax.plot(xs, [p[1]["p95"] for p in pts], color=color, marker=marker, markersize=4.5,
                    markerfacecolor="white", markeredgewidth=1.0, linewidth=1.2, linestyle="--")
            handles.append(Line2D([], [], color=color, marker=marker, markersize=4.5,
                                  markerfacecolor="white", linewidth=1.2, label=f"BACME (N={nodes})"))
            print(f"{op:<10}{nodes:>4}" + "".join(
                f"{p[1]['p50']:>9.0f} / {p[1]['p95']:<5.0f}" for p in pts))

        # Distinguish p50 / p95 by line style
        handles += [Line2D([], [], color="black", linewidth=1.0, linestyle="-", label="p50"),
                    Line2D([], [], color="black", linewidth=1.0, linestyle="--", label="p95")]

        style_axis(ax, use_offset=(op != "verify"))
        # Ticks at 0/50/100/150/200; data points stay at the true RTT (20 ms has a point but no tick label, to avoid crowding next to 0)
        step = 50 if max(rtts) >= 150 else 20
        ax.set_xticks(list(range(0, int(max(rtts)) + 1, step)))
        ax.set_xlim(min(rtts) - 12, max(rtts) + 12)
        ax.set_xlabel("Inter-node RTT (ms)", labelpad=1)
        ax.set_ylabel(ylab, labelpad=1)
        lo, hi = ax.get_ylim()
        ax.set_ylim(0, hi * (2.3 if op == "verify" else 1.75))  # leave room for the upper-left legend (5 rows) so it does not cover the curves
        ax.legend(handles=handles, loc="upper left", frameon=True, fancybox=False,
                  edgecolor="black", framealpha=1, handlelength=1.6, borderpad=0.3,
                  labelspacing=0.2)
        ax.text(0.5, -0.31, f"({chr(ord('a') + idx)}) {SUBPLOT_TITLES.get(op, op)}",
                transform=ax.transAxes, ha="center", va="top", fontsize=8.5, fontweight="bold")

    fig.tight_layout(pad=0.3, w_pad=0.8)
    fig.subplots_adjust(bottom=0.27)
    os.makedirs(args.out, exist_ok=True)
    base = os.path.join(args.out, "fig_latency_rtt")
    fig.savefig(base + ".pdf", bbox_inches="tight")
    fig.savefig(base + ".png", dpi=300, bbox_inches="tight")

    total_runs = sum(v["total"] for v in data.values()) + sum(
        1 for e in excluded if e[0] not in data)
    print(f"\nSaved: {base}.pdf / .png")
    if excluded:
        print(f"\nExcluded {len(excluded)} runs with suspected view changes (out of {total_runs}):")
        for (op, nodes, rtt), run, p50, p95, blocks in excluded:
            print(f"  N={nodes} RTT={rtt}ms {op} run {run}: p50={p50:.0f} p95={p95:.0f} blocks={blocks}")
    print("Suggested caption: Confirmation latency (p50 solid, p95 dashed) of certificate management "
          "operations under different inter-node RTTs at a fixed sending rate: " +
          "; ".join(f"({chr(ord('a') + i)}) {OP_NAMES[op]}" for i, (op, _) in enumerate(ops)) + ".")


if __name__ == "__main__":
    main()
