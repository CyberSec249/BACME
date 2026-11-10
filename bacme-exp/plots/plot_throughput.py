#!/usr/bin/env python3
"""
Throughput vs. inter-node RTT (same style as Fig. 9 of the paper)

Usage:
    python3 plot_throughput.py                                  # reads ./throughput.csv by default
    python3 plot_throughput.py --csv v2/throughput.csv --out figs
    python3 plot_throughput.py --no-errorbar                    # do not draw standard-deviation error bars
    python3 plot_throughput.py --include-stalls                 # do not exclude runs with suspected view changes

Output: <out>/fig_throughput_rtt.pdf and .png
x-axis: inter-node RTT (ms); y-axis: throughput of successful transactions (query throughput for verify);
one line per number of consensus nodes N.
"""
import argparse
import csv
import os
import statistics
from collections import defaultdict

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
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
    "pdf.fonttype": 42,          # embed TrueType fonts so submission systems render them correctly
    "ps.fonttype": 42,
})

# node count -> (color, marker), same color order as BACME / Baseline 1 / Baseline 2 in the paper
SERIES_STYLE = {
    10: ("#0072BD", "^"),   # blue triangle
    7: ("#9467BD", "s"),    # purple square
    4: ("#FF7F0E", "D"),    # orange diamond
}
FALLBACK_STYLES = [("#17BECF", "v"), ("#D62728", "o"), ("#2CA02C", "p")]

# operation -> y-axis label. Subplot order follows Fig. 9 of the paper
OPS = [
    ("bacme_v1", "Throughput (tx/s)"),       # domain control validation (on-chain stage 1: controlValidate)
    ("validation", "Throughput (tx/s)"),
    ("apply", "Throughput (tx/s)"),
    ("update", "Throughput (tx/s)"),
    ("revoke", "Throughput (tx/s)"),
    ("revokeDV", "Throughput (tx/s)"),
    ("revokeP", "Throughput (tx/s)"),
    ("verify", "Throughput (queries/s)"),
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


def load(path, include_stalls=False):
    """Return {(op, nodes, rtt): [tps_ok, ...]} and the excluded runs.

    If a PBFT view change occurs under full load, block production stalls for about consensus_timeout (3 s), noticeably lengthening the run.
    By default, write runs with duration >= group median + max(2 s, 10% x median) are excluded (verify does not go through consensus and is never excluded).
    DCV runs take 40-90 s and naturally vary by several seconds, so the threshold scales with run duration; for 10-17 s runs such as apply it remains 2 s."""
    groups = defaultdict(list)
    with open(path, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f):
            if row["total"] == "FAILED" or not row.get("tps_ok"):
                continue
            groups[(row["op"], int(row["nodes"]), int(row["rtt_target_ms"]))].append(row)
    data, excluded = defaultdict(list), []
    for key, rows in groups.items():
        med_time = statistics.median(float(r["time_ms"]) for r in rows)
        for r in rows:
            stall = key[0] != "verify" and float(r["time_ms"]) >= med_time + max(2000, 0.1 * med_time)
            if stall and not include_stalls:
                excluded.append((key, r["run"], float(r["tps_ok"]), float(r["time_ms"]) - med_time))
            else:
                data[key].append(float(r["tps_ok"]))
    return data, excluded


def style_axis(ax):
    ax.grid(True, color="#D9D9D9", linewidth=0.5)
    ax.set_axisbelow(True)
    for side in ("top", "right"):
        ax.spines[side].set_visible(True)
    ax.tick_params(top=False, right=False)
    fmt = ScalarFormatter(useMathText=True)
    fmt.set_powerlimits((0, 0))            # always show as x10^n, consistent with MATLAB
    ax.yaxis.set_major_formatter(fmt)
    ax.yaxis.get_offset_text().set_fontsize(8)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--csv", default="throughput.csv")
    ap.add_argument("--out", default=".")
    ap.add_argument("--no-errorbar", action="store_true")
    ap.add_argument("--include-stalls", action="store_true", help="do not exclude runs with suspected view changes")
    args = ap.parse_args()

    data, excluded = load(args.csv, args.include_stalls)
    ops = [(op, ylab) for op, ylab in OPS if any(k[0] == op for k in data)]
    nodes_all = sorted({k[1] for k in data}, reverse=True)
    rtts = sorted({k[2] for k in data})

    n = len(ops)
    fig, axes = plt.subplots(1, n, figsize=(1.6 * n + 0.2, 1.9))
    if n == 1:
        axes = [axes]

    print(f"{'Operation':<10}{'N':>4}" + "".join(f"{'RTT ' + str(r):>16}" for r in rtts))
    for idx, (ax, (op, ylab)) in enumerate(zip(axes, ops)):
        for j, nodes in enumerate(nodes_all):
            color, marker = SERIES_STYLE.get(nodes, FALLBACK_STYLES[j % len(FALLBACK_STYLES)])
            xs, ys, es = [], [], []
            for r in rtts:
                vals = data.get((op, nodes, r))
                if vals:
                    xs.append(r)
                    ys.append(statistics.mean(vals))
                    es.append(statistics.stdev(vals) if len(vals) > 1 else 0.0)
            if not xs:
                continue
            if not args.no_errorbar:
                ax.errorbar(xs, ys, yerr=es, fmt="none", ecolor=color, capsize=1.8,
                            elinewidth=0.7, capthick=0.7)
            # The legend shows only line + marker (as in the paper); error bars are drawn separately
            ax.plot(xs, ys, color=color, marker=marker, markersize=4.5, markerfacecolor="white",
                    markeredgewidth=1.0, linewidth=1.2, label=f"BACME (N={nodes})")
            print(f"{op:<10}{nodes:>4}" + "".join(
                f"{m:>9.1f}±{s:<6.1f}" for m, s in zip(ys, es)))

        style_axis(ax)
        # Ticks at 0/50/100/150/200; data points stay at the true RTT (20 ms has a point but no tick label, to avoid crowding next to 0)
        step = 50 if max(rtts) >= 150 else 20
        ax.set_xticks(list(range(0, int(max(rtts)) + 1, step)))
        ax.set_xlim(min(rtts) - 12, max(rtts) + 12)
        ax.set_xlabel("Inter-node RTT (ms)", labelpad=1)
        ax.set_ylabel(ylab, labelpad=1)
        lo, hi = ax.get_ylim()
        ax.set_ylim(0, hi * 1.5)             # start at 0 and leave room for the upper-right legend (avoids covering the N=4 curve in the DCV subplot)
        # verify data sit near the top, so its legend goes to the lower right
        ax.legend(loc="lower right" if op == "verify" else "upper right",
                  frameon=True, fancybox=False, edgecolor="black",
                  framealpha=1, handlelength=1.6, borderpad=0.3, labelspacing=0.2)
        ax.text(0.5, -0.31, f"({chr(ord('a') + idx)})", transform=ax.transAxes,
                ha="center", va="top", fontsize=9)

    fig.tight_layout(pad=0.3, w_pad=0.8)
    fig.subplots_adjust(bottom=0.27)
    os.makedirs(args.out, exist_ok=True)
    base = os.path.join(args.out, "fig_throughput_rtt")
    fig.savefig(base + ".pdf", bbox_inches="tight")
    fig.savefig(base + ".png", dpi=300, bbox_inches="tight")
    print(f"\nSaved: {base}.pdf / .png")
    if excluded:
        print(f"\nExcluded {len(excluded)} runs with suspected view changes (duration exceeds the group median by the threshold):")
        for (op, nodes, rtt), run, tps, extra in excluded:
            print(f"  N={nodes} RTT={rtt}ms {op} run {run}: TPS={tps:.0f}, {extra/1000:.1f} s longer")
    print("Suggested caption: Throughput of certificate management operations under different "
          "inter-node RTTs: " + "; ".join(f"({chr(ord('a') + i)}) {OP_NAMES[op]}"
                                         for i, (op, _) in enumerate(ops)) + ".")


if __name__ == "__main__":
    main()
