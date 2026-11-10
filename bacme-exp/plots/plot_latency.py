#!/usr/bin/env python3
"""
Confirmation latency vs. inter-node RTT (same style as Fig. 9 / Fig. 12 of the paper)

Usage:
    python3 plot_latency.py                                     # reads ./latency.csv by default
    python3 plot_latency.py --csv v2/latency.csv --out figs
    python3 plot_latency.py --include-stalls                    # do not exclude runs with suspected view changes

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
            data[key] = {
                "p50": statistics.mean(float(r["p50_ms"]) for r in kept),
                "p95": statistics.mean(float(r["p95_ms"]) for r in kept),
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


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--csv", default="latency.csv")
    ap.add_argument("--out", default=".")
    ap.add_argument("--include-stalls", action="store_true")
    args = ap.parse_args()

    data, excluded = load(args.csv, args.include_stalls)
    ops = [(op, ylab) for op, ylab in OPS if any(k[0] == op for k in data)]
    nodes_all = sorted({k[1] for k in data}, reverse=True)
    rtts = sorted({k[2] for k in data})

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
        ax.text(0.5, -0.31, f"({chr(ord('a') + idx)})", transform=ax.transAxes,
                ha="center", va="top", fontsize=9)

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
