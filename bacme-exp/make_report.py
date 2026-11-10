#!/usr/bin/env python3
"""
Generate a Markdown experiment report from the results produced by run_exp.sh (ready to commit to GitHub).

Usage:
    python3 make_report.py                       # reads files under ~/exp-results by default
    python3 make_report.py --dir ~/exp-results -o ~/exp-results/EXPERIMENT_RESULTS.md

Input files (whichever exist are used):
    throughput.csv   throughput experiment (high sending rate, maximum TPS)
    latency.csv      latency experiment (fixed low sending rate, p50/p95 confirmation latency)
    env.txt          environment information collected by collect_env.sh
"""
import argparse
import csv
import os
import statistics
from collections import defaultdict

OP_ORDER = ["validation", "bacme_v1", "dacme_v1", "dcv_v1", "bacme_v2", "dacme_v2", "dcv_v2",
            "apply", "update", "verify", "revoke", "revokeDV", "revokeP"]


def load(path):
    """Group by (nodes, RTT, operation); return {key: {metric: [values...]}} and the number of failed runs."""
    groups = defaultdict(lambda: defaultdict(list))
    failed = defaultdict(int)
    if not os.path.exists(path):
        return groups, failed
    with open(path, newline="", encoding="utf-8") as f:
        for row in csv.DictReader(f):
            key = (int(row["nodes"]), int(row["rtt_target_ms"]), row["op"])
            if row["total"] == "FAILED":
                failed[key] += 1
                continue
            # A p95 far above p50 (> 3x) within a run usually indicates a PBFT view change (block production stalls for about consensus_timeout)
            try:
                if float(row["p95_ms"]) > 3 * max(float(row["p50_ms"]), 1) and float(row["p50_ms"]) > 50:
                    groups[key]["suspect_runs"].append(int(row["run"]))
            except (ValueError, KeyError):
                pass
            for m in ("tps_ok", "p50_ms", "p95_ms", "p99_ms", "avg_ms", "err_rate_pct",
                      "rtt_measured_ms", "qps", "count", "blocks"):
                if row.get(m) not in ("", None):
                    groups[key][m].append(float(row[m]))
    return groups, failed


def ms(values, digits=1):
    if not values:
        return "-"
    m = statistics.mean(values)
    s = statistics.stdev(values) if len(values) > 1 else 0.0
    return f"{m:.{digits}f} ± {s:.{digits}f}"


def mean(values):
    return statistics.mean(values) if values else None


def op_rank(op):
    return (OP_ORDER.index(op) if op in OP_ORDER else 99, op)


def sorted_ops(ops):
    return sorted(ops, key=op_rank)


def pivot_table(groups, failed, nodes, cell, flag=None):
    """Rows = operations, columns = RTT. cell(g) returns the cell text; flag(g) returns a note to annotate."""
    keys = [k for k in set(groups) | set(failed) if k[0] == nodes]
    rtts = sorted({k[1] for k in keys})
    ops = sorted_ops({k[2] for k in keys})
    lines = ["| Operation | " + " | ".join(f"RTT {r} ms" for r in rtts) + " |",
             "|---|" + "---|" * len(rtts)]
    notes = []
    for op in ops:
        cells = []
        for r in rtts:
            g = groups.get((nodes, r, op))
            nf = failed.get((nodes, r, op), 0)
            text = cell(g) if g else "-"
            if nf:
                text += f" ({nf} runs failed)"
            if g and flag:
                mark = flag(g)
                if mark:
                    text += " ⚠"
                    notes.append(f"{op} @ RTT {r} ms: {mark}")
            cells.append(text)
        lines.append(f"| {op} | " + " | ".join(cells) + " |")
    return lines, notes


def measured_rtt_line(groups_list, nodes):
    per_rtt = defaultdict(list)
    for groups in groups_list:
        for (n, r, _), g in groups.items():
            if n == nodes:
                per_rtt[r] += g.get("rtt_measured_ms", [])
    if not per_rtt:
        return None
    parts = [f"{r} ms → {statistics.mean(v):.2f} ms" for r, v in sorted(per_rtt.items()) if v]
    return "Target RTT -> measured TCP handshake RTT: " + "; ".join(parts)


def main():
    default_dir = os.path.expanduser("~/exp-results")
    ap = argparse.ArgumentParser()
    ap.add_argument("--dir", default=default_dir)
    ap.add_argument("-o", "--output", default=None)
    args = ap.parse_args()
    out = args.output or os.path.join(args.dir, "EXPERIMENT_RESULTS.md")

    tp, tp_failed = load(os.path.join(args.dir, "throughput.csv"))
    lat, lat_failed = load(os.path.join(args.dir, "latency.csv"))
    env_path = os.path.join(args.dir, "env.txt")
    all_nodes = sorted({k[0] for k in list(tp) + list(tp_failed) + list(lat) + list(lat_failed)},
                       reverse=True)

    md = ["# BACME Performance Results: Network Latency x Number of Consensus Nodes", ""]

    md += ["## 1. Methodology", "",
           "- **Deployment**: FISCO BCOS 2.x; all consensus nodes and the benchmark client run on the same virtual machine.",
           "- **Network delay emulation**: Linux `tc netem` injects a one-way delay D only on inter-node P2P port traffic, "
           "so the inter-node round-trip time is RTT ≈ 2D; client-to-node traffic is not delayed.",
           "- **Number of nodes**: the number of consensus nodes N and the number of tolerated Byzantine nodes f satisfy N = 3f + 1. "
           "The number of nodes is adjusted with the console command `removeNode` (10 -> 7 -> 4).",
           "- **Throughput experiment**: transactions are sent at a rate above the system capacity, and the TPS of successful transactions is measured.",
           "- **Latency experiment**: transactions are sent at a fixed rate below the system capacity, and the confirmation latency "
           "(from submission to receipt) is measured as p50 / p95.",
           "- **verify** is a read-only contract call (`view` function) executed locally on a single node without consensus, "
           "so its results are query throughput (QPS) and query latency.",
           "- **The benchmark account** is randomly generated in each run and has not passed domain control validation, so apply / update / revoke "
           "transactions return early inside the contract at the authorization check; they still go through PBFT consensus and on-chain commitment.",
           "- **Domains per certificate**: all tests use single-domain certificates (one domain per certificate).",
           "- **Domain control validation (DCV)**: only the on-chain stages are measured. `*_v1` is stage 1 (VRF randomness request: BACME `controlValidate`, "
           "Baseline 1 `RegisterDomain`, Baseline 2 `registerDomain`); `*_v2` is stage 2 (start of HTTP validation: "
           "`startValidation` / `StartValidation`). Each transaction uses a new account and a unique domain; stage 2 first completes stage 1 (not recorded). "
           "Oracle callbacks are identical for the three schemes and are not included. The blocks column of `*_v2` rows includes the blocks of stage 1.",
           "- Each combination is repeated several times; values are **mean ± standard deviation**.",
           ""]

    if os.path.exists(env_path):
        with open(env_path, encoding="utf-8") as f:
            md += ["## 2. Experimental Environment", "", "```", f.read().rstrip(), "```", ""]
    else:
        md += ["## 2. Experimental Environment", "", "(env.txt not found; run collect_env.sh first)", ""]

    sec = 3
    for nodes in all_nodes:
        f_val = (nodes - 1) // 3
        md += [f"## {sec}. Number of consensus nodes N = {nodes} (f = {f_val})", ""]
        rtt_line = measured_rtt_line([tp, lat], nodes)
        if rtt_line:
            md += [rtt_line, ""]

        if any(k[0] == nodes for k in list(tp) + list(tp_failed)):
            qps = next((mean(g["qps"]) for k, g in tp.items() if k[0] == nodes and g.get("qps")), None)
            cnt = next((mean(g["count"]) for k, g in tp.items() if k[0] == nodes and g.get("count")), None)
            md += [f"### {sec}.1 Throughput (TPS of successful transactions)",
                   "", f"Maximum sending rate {qps:.0f} tx/s, {cnt:.0f} transactions per run." if qps else "", ""]
            lines, notes = pivot_table(
                tp, tp_failed, nodes,
                cell=lambda g: ms(g.get("tps_ok", [])),
                flag=lambda g: (f"error rate {mean(g['err_rate_pct']):.2f}%"
                                if mean(g.get("err_rate_pct", [])) and mean(g["err_rate_pct"]) > 1 else None))
            md += lines + [""]
            if notes:
                md += ["Note: " + "; ".join(notes), ""]

        if any(k[0] == nodes for k in list(lat) + list(lat_failed)):
            qps = next((mean(g["qps"]) for k, g in lat.items() if k[0] == nodes and g.get("qps")), None)
            cnt = next((mean(g["count"]) for k, g in lat.items() if k[0] == nodes and g.get("count")), None)
            md += [f"### {sec}.2 Confirmation latency (ms, p50 / p95)",
                   "", f"Fixed sending rate {qps:.0f} tx/s, {cnt:.0f} transactions per run." if qps else "", ""]

            def lat_cell(g):
                p50, p95 = mean(g.get("p50_ms", [])), mean(g.get("p95_ms", []))
                if p50 is None:
                    return "-"
                return f"{p50:.0f} / {p95:.0f}"

            def lat_flag(g):
                msgs = []
                # A successful TPS clearly below the sending rate indicates possible saturation; the latency then includes queuing time
                tps, q = mean(g.get("tps_ok", [])), mean(g.get("qps", []))
                if tps is not None and q and tps < 0.8 * q:
                    msgs.append(f"successful TPS {tps:.0f} is clearly below the sending rate {q:.0f}; possibly saturated, latency includes queuing")
                runs = g.get("suspect_runs", [])
                if runs:
                    msgs.append(f"run(s) {','.join(map(str, runs))}: p95 exceeds 3x p50, suspected PBFT view change")
                return "; ".join(msgs) if msgs else None

            lines, notes = pivot_table(lat, lat_failed, nodes, cell=lat_cell, flag=lat_flag)
            md += lines + [""]
            if notes:
                md += ["Note: " + "; ".join(notes), ""]

            md += [f"<details><summary>N = {nodes} latency details (mean ± standard deviation)</summary>", "",
                   "| Operation | RTT (ms) | Runs | Successful TPS | Mean latency | p50 | p95 | p99 | Error rate % |",
                   "|---|---|---|---|---|---|---|---|---|"]
            for key in sorted((k for k in lat if k[0] == nodes), key=lambda k: (op_rank(k[2]), k[1])):
                g = lat[key]
                md.append(f"| {key[2]} | {key[1]} | {len(g.get('tps_ok', []))} | {ms(g.get('tps_ok', []))} | "
                          f"{ms(g.get('avg_ms', []))} | {ms(g.get('p50_ms', []))} | {ms(g.get('p95_ms', []))} | "
                          f"{ms(g.get('p99_ms', []))} | {ms(g.get('err_rate_pct', []), 2)} |")
            md += ["", "</details>", ""]
        sec += 1

    md += ["## Raw Data", "",
           "- `throughput.csv`, `latency.csv`: raw results of every benchmark run",
           "- `env.txt`: experimental environment and consensus parameters",
           "- The full output of each run is in `logs/` next to the result files", ""]

    with open(out, "w", encoding="utf-8") as f:
        f.write("\n".join(md))
    print(f"Report generated: {out}")


if __name__ == "__main__":
    main()
