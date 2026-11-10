#!/usr/bin/env python3
"""
Large certificate state experiment: throughput and confirmation latency of BACME operations under different numbers of on-chain certificate records (grouped bar chart).

Usage:
    python3 plot_scale.py --dir bacme-exp-results/scale --out figs
    python3 plot_scale.py --dir ... --levels s0 s1m --labels "<0.4 M records" "~1.4 M records"

Reads <dir>/throughput_<level>.csv and <dir>/latency_<level>.csv (run_exp.sh output format).
Outputs <out>/fig_scale.pdf / .png, with two subplots by default (only operations that write to the smart contract):
  (a) throughput of application/renewal/revocation (mean ± standard deviation)
  (b) confirmation latency of application/renewal/revocation (solid = p50, hatched = p95)
With --with-verify, a subplot for the throughput of certificate verification (read-only query) is added in the middle.
The latency of certificate verification is not plotted (the first-level verify latency was measured during background record insertion and was excluded); it is reported in the text.

Exclusion rules are the same as in the network latency experiment:
  Throughput: a write run whose duration >= group median + max(2 s, 10% x median) is treated as a view change and excluded
  Latency: excluded if p95 > 3 x p50 (and p50 > 50 ms), or if an application run produced <= 75% of the group's maximum number of blocks
       (the block count of renewal/revocation includes the untimed preparation phase, so the block rule does not apply to them)
"""
import argparse
import csv
import os
import statistics
from collections import defaultdict

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
from matplotlib.patches import Patch

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

LEVEL_COLORS = ["#0072BD", "#FF7F0E", "#9467BD", "#2CA02C"]
WRITE_OPS = [("scale_apply", "Application"), ("scale_update", "Renewal"), ("scale_revoke", "Revocation")]
VERIFY_OP = ("scale_verify", "Verification")


def read(path):
    if not os.path.exists(path):
        return defaultdict(list)
    g = defaultdict(list)
    with open(path, newline="", encoding="utf-8") as f:
        for r in csv.DictReader(f):
            if r["total"] == "FAILED" or not r.get("tps_ok"):
                continue
            g[r["op"]].append(r)
    return g


def throughput(rows, op, excluded, tag):
    if not rows:
        return None
    med = statistics.median(float(r["time_ms"]) for r in rows)
    vals = []
    for r in rows:
        if op != "scale_verify" and float(r["time_ms"]) >= med + max(2000, 0.1 * med):
            excluded.append(f"{tag} {op} throughput run {r['run']} TPS={float(r['tps_ok']):.0f}")
            continue
        vals.append(float(r["tps_ok"]))
    return statistics.mean(vals), (statistics.stdev(vals) if len(vals) > 1 else 0.0)


def latency(rows, op, excluded, tag):
    if not rows:
        return None
    max_blocks = max(int(r["blocks"]) for r in rows)
    p50s, p95s = [], []
    for r in rows:
        p50, p95, b = float(r["p50_ms"]), float(r["p95_ms"]), int(r["blocks"])
        if (p95 > 3 * p50 and p50 > 50) or (op == "scale_apply" and max_blocks > 0 and b <= 0.75 * max_blocks):
            excluded.append(f"{tag} {op} latency run {r['run']} p50={p50:.0f} p95={p95:.0f} blocks={b}")
            continue
        p50s.append(p50)
        p95s.append(p95)
    return statistics.mean(p50s), statistics.mean(p95s)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--dir", default=".")
    ap.add_argument("--out", default=".")
    ap.add_argument("--levels", nargs="+", default=["s0", "s1m", "s2m"])
    ap.add_argument("--labels", nargs="+", default=["< 0.4 M records", "~ 1.4 M records", "~ 2.8 M records"])
    ap.add_argument("--with-verify", action="store_true",
                    help="also plot the throughput of certificate verification (read-only query); by default only write operations are plotted")
    args = ap.parse_args()
    levels = args.levels
    labels = (args.labels + levels)[:len(levels)]

    excluded = []
    T, L = {}, {}
    for lv, lab in zip(levels, labels):
        tp = read(os.path.join(args.dir, f"throughput_{lv}.csv"))
        lt = read(os.path.join(args.dir, f"latency_{lv}.csv"))
        for op, _ in WRITE_OPS + [VERIFY_OP]:
            T[(lv, op)] = throughput(tp.get(op, []), op, excluded, lv)
            L[(lv, op)] = latency(lt.get(op, []), op, excluded, lv)

    # ------- Print the value table -------
    print(f"{'Operation':<14}" + "".join(f"{lv:>26}" for lv in levels))
    for op, name in WRITE_OPS + [VERIFY_OP]:
        row = f"{name:<14}"
        for lv in levels:
            t = T[(lv, op)]
            row += f"{('%.0f ± %.0f' % t) if t else '-':>26}"
        print(row + "   (throughput)")
    for op, name in WRITE_OPS + [VERIFY_OP]:
        row = f"{name:<14}"
        for lv in levels:
            l = L[(lv, op)]
            row += f"{('p50 %.0f / p95 %.0f' % l) if l else '-':>26}"
        print(row + "   (latency, ms)")
    if len(levels) >= 2:
        a, b = levels[0], levels[-1]
        print(f"\nChange of {b} relative to {a}:")
        for op, name in WRITE_OPS + [VERIFY_OP]:
            if T[(a, op)] and T[(b, op)]:
                msg = f"  {name:<13} throughput {100 * (T[(b, op)][0] / T[(a, op)][0] - 1):+.1f}%"
                if L[(a, op)] and L[(b, op)]:
                    msg += f"   p50 {100 * (L[(b, op)][0] / L[(a, op)][0] - 1):+.1f}%   p95 {100 * (L[(b, op)][1] / L[(a, op)][1] - 1):+.1f}%"
                print(msg)

    # ------- Plot -------
    k = len(levels)
    width = 0.8 / k
    if args.with_verify:
        fig, axs = plt.subplots(1, 3, figsize=(7.2, 2.1), gridspec_kw={"width_ratios": [3, 1.35, 3]})
        ax_tp, ax_q, ax_lat = axs
    else:
        fig, axs = plt.subplots(1, 2, figsize=(6.0, 2.1))
        ax_tp, ax_lat = axs
        ax_q = None
    panel = iter(range(3))

    def finish(ax, ylab, idx, title, ymax_factor=1.45):
        ax.grid(True, axis="y", color="#D9D9D9", linewidth=0.5)
        ax.set_axisbelow(True)
        ax.tick_params(top=False, right=False)
        ax.set_ylabel(ylab, labelpad=1)
        lo, hi = ax.get_ylim()
        ax.set_ylim(0, hi * ymax_factor)
        ax.text(0.5, -0.24, f"({chr(ord('a') + idx)}) {title}", transform=ax.transAxes,
                ha="center", va="top", fontsize=8.5, fontweight="bold")

    level_handles = [Patch(facecolor=LEVEL_COLORS[j % 4], edgecolor="black", linewidth=0.4, label=labels[j])
                     for j in range(k)]

    # (a) Write throughput
    ax = ax_tp
    for j, lv in enumerate(levels):
        xs, ys, es = [], [], []
        for i, (op, _) in enumerate(WRITE_OPS):
            t = T[(lv, op)]
            if t:
                xs.append(i + (j - (k - 1) / 2) * width); ys.append(t[0]); es.append(t[1])
        ax.bar(xs, ys, width=width, color=LEVEL_COLORS[j % 4], edgecolor="black", linewidth=0.4, zorder=3,
               yerr=es, error_kw=dict(elinewidth=0.7, capsize=1.8, capthick=0.7, ecolor="black"))
    ax.set_xticks(range(len(WRITE_OPS)))
    ax.set_xticklabels([n for _, n in WRITE_OPS])
    finish(ax, "Throughput (tx/s)", next(panel), "Throughput")
    ax.legend(handles=level_handles, loc="upper left", frameon=True, fancybox=False, edgecolor="black",
              framealpha=1, handlelength=1.2, borderpad=0.3, labelspacing=0.2)

    # (b) Verification throughput (optional)
    ax = ax_q
    for j, lv in enumerate(levels if ax is not None else []):
        t = T[(lv, VERIFY_OP[0])]
        if t:
            ax.bar([(j - (k - 1) / 2) * width], [t[0]], width=width, color=LEVEL_COLORS[j % 4],
                   edgecolor="black", linewidth=0.4, zorder=3, yerr=[t[1]],
                   error_kw=dict(elinewidth=0.7, capsize=1.8, capthick=0.7, ecolor="black"))
    if ax is not None:
        ax.set_xticks([0])
        ax.set_xticklabels([VERIFY_OP[1]])
        ax.set_xlim(-0.75, 0.75)
        finish(ax, "Throughput (queries/s)", next(panel), "Query throughput")

    # (c) Write confirmation latency
    ax = ax_lat
    for j, lv in enumerate(levels):
        xs, p50, p95 = [], [], []
        for i, (op, _) in enumerate(WRITE_OPS):
            l = L[(lv, op)]
            if l:
                xs.append(i + (j - (k - 1) / 2) * width); p50.append(l[0]); p95.append(l[1])
        c = LEVEL_COLORS[j % 4]
        ax.bar(xs, p50, width=width, color=c, edgecolor="black", linewidth=0.4, zorder=3)
        ax.bar(xs, [b - a for a, b in zip(p50, p95)], bottom=p50, width=width, color=c, alpha=0.35,
               hatch="////", edgecolor="black", linewidth=0.4, zorder=3)
    ax.set_xticks(range(len(WRITE_OPS)))
    ax.set_xticklabels([n for _, n in WRITE_OPS])
    finish(ax, "Latency (ms)", next(panel), "Confirmation latency", ymax_factor=1.75)
    ax.legend(handles=level_handles + [
        Patch(facecolor="#BBBBBB", edgecolor="black", linewidth=0.4, label="Solid top: p50"),
        Patch(facecolor="#BBBBBB", alpha=0.35, hatch="////", edgecolor="black", linewidth=0.4,
              label="Hatched top: p95")],
        loc="upper left", ncol=2, frameon=True, fancybox=False, edgecolor="black", framealpha=1,
        handlelength=1.2, borderpad=0.3, labelspacing=0.2, columnspacing=0.6, fontsize=6)

    fig.tight_layout(pad=0.3, w_pad=1.0)
    fig.subplots_adjust(bottom=0.24)
    os.makedirs(args.out, exist_ok=True)
    base = os.path.join(args.out, "fig_scale")
    fig.savefig(base + ".pdf", bbox_inches="tight")
    fig.savefig(base + ".png", dpi=300, bbox_inches="tight")
    print(f"\nSaved: {base}.pdf / .png")
    if excluded:
        print(f"\nExcluded {len(excluded)} runs:")
        for e in excluded:
            print("  " + e)


if __name__ == "__main__":
    main()
