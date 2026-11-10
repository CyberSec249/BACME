# BACME Experiment Toolkit: Network Latency x Number of Nodes (Single Host)

Experiment design: inter-node RTT ∈ {0, 20, 50, 100, 200} ms x number of consensus nodes N ∈ {4, 7, 10} (f = 1, 2, 3),
measuring the TPS and confirmation latency (p50 / p95 / p99) of each operation. The toolkit also contains the
large certificate state experiment (see `SCALE_NOTES.md`).

## Files

| File | Purpose |
|---|---|
| `netem.sh` | Uses tc netem to add delay only to inter-node P2P ports (one-way delay D -> RTT ≈ 2D) |
| `exp_nodes.sh` | Shows node status, prints removeNode / addObserver / addSealer commands, starts or stops nodes, shows block heights |
| `run_exp.sh` | Iterates over RTT levels x operations x repetitions and appends the results to a CSV file |
| `seed_scale.sh` | Pre-populates certificate records on chain for the large certificate state experiment |
| `summarize.py` | Aggregates mean ± standard deviation by (nodes, RTT, operation) into `summary.csv` |
| `collect_env.sh` | Collects hardware, OS and FISCO consensus parameters into `env.txt` |
| `make_report.py` | Generates the Markdown report `EXPERIMENT_RESULTS.md` from `throughput.csv`, `latency.csv` and `env.txt` |
| `perf-patch/java-sdk-demo-1.0.0.jar` | Patched benchmark program; replaces the file of the same name in `performanceTest/java-sdk-demo/dist/apps/` |
| `perf-patch/*.diff`, `perf-patch/src/` | Source code and diffs of the patches, for review and version control |
| `perf-patch/src/PerformanceDomainValidation.java` | New: benchmark of the on-chain stages of domain control validation (BACME and the two baselines); included in the jar |
| `perf-patch/src/PerformanceScale.java` | New: benchmark with complete state updates for the large certificate state experiment; included in the jar |
| `plots/plot_throughput_network_latency.py`, `plots/plot_latency_network_latency.py` | Throughput / latency vs. RTT figures used in the paper (one series per node count) |
| `plots/plot_scale.py` | Figure of the large certificate state experiment |
| `plots/plot_throughput.py`, `plots/plot_latency.py` | Earlier versions of the RTT figures, kept for reference |
| `plots/plot_dcv.py` | Domain control validation: comparison of BACME and the two baselines |
| `SCALE_NOTES.md` | Notes that must be stated when reporting the large certificate state experiment |

## Changes to the benchmark program (benchmark logic unchanged)

1. `PerformanceDpkiAcme`: the contract address is read from the environment variable `DPKIACME_ADDR` (falls back to the original hard-coded address if unset).
2. `PerformanceCollector`:
   - records the latency of every **successful** request and additionally prints p50 / p95 / p99 / max (nearest-rank method) at the end;
   - additionally prints a `RESULT_CSV,...` line for the scripts to parse;
   - fixes a race in the original code where the main thread could exit before the results were printed (the last request is printed before it is counted);
   - uses the total number of requests as the denominator of the error rate (it was off by one under the new counting).

All original outputs (TPS, average latency, latency distribution buckets, etc.) are kept unchanged.

Note: `PerformanceDpkiAcme` generates a random account for each run. This account has not passed domain control
validation, so apply / update / revoke transactions go through transaction submission, PBFT consensus and contract
invocation, but terminate at the authorization check of the smart contract. `PerformanceScale` uses a validated
account and a distinct certificate hash per transaction, so every write transaction executes the complete state update.

## Experiment steps (overview)

1. Power off the VM and increase its resources (at least 8 cores and 8 GB of memory recommended); record the configuration.
2. Build a 10-node FISCO 2.x chain with `build_chain.sh -l 127.0.0.1:10 -p 30300,20200,8545`.
3. Reset the Truora database, connect it to the new chain, and redeploy the DpkiAcme contract.
4. Replace the benchmark jar, copy the new chain's certificates to `dist/conf/` of the benchmark program, and install the 2.x console.
5. In the order 10 -> 7 -> 4: adjust the consensus nodes -> `bash run_exp.sh --nodes N`.
6. Run `bash collect_env.sh` to collect the environment and `python3 make_report.py` to generate the report.
