#!/bin/bash
# =============================================================
# Automated performance experiments: network latency x number of nodes (single host)
#
# For the current number of consensus nodes, sets each inter-node RTT level in turn and repeats the benchmark for each operation;
# results are appended to a CSV file. Run this script once per node count.
#
# Usage:
#   bash run_exp.sh --nodes 10
#   bash run_exp.sh --nodes 7 --rtts "0 20 50 100 200" --ops "apply update verify revoke" \
#                   --count 10000 --qps 10000 --repeat 3
#
# Options (defaults in parentheses):
#   --nodes N       current number of consensus nodes, checked against the chain (required)
#   --rtts  "..."   inter-node RTT levels in ms ("0 20 50 100 200")
#   --ops   "..."   operations to benchmark ("apply update verify revoke")
#                   domain control validation: bacme_v1 bacme_v2 dacme_v1 dacme_v2 dcv_v1 dcv_v2
#                   (_v1 = stage 1, request randomness; _v2 = stage 2, start HTTP validation; requires DACME_ADDR / DCV_ADDR)
#                   large certificate state experiment: scale_apply scale_update scale_revoke scale_verify
#                   (complete execution path; requires ACCOUNT_PEM / SCALE_DOMAIN, see PerformanceScale.java)
#   --count N       total requests per run (10000)
#   --qps   N       maximum sending rate (10000)
#   --repeat N      repetitions per combination (3)
#   --interval S    seconds between runs (10)
#   --warmup S      seconds to wait after switching the RTT (15)
#   --warmup-run 0|1 run one discarded warm-up benchmark after switching the RTT (1)
#   --out FILE      result file (~/exp-results/results.csv)
# =============================================================

NODES=""
RTTS="0 20 50 100 200"
OPS="apply update verify revoke"
COUNT=10000
QPS=10000
REPEAT=3
INTERVAL=10
WARMUP=15
WARMUP_RUN=1
OUT="$HOME/exp-results/results.csv"

PERF_DIR="$HOME/DPKIACME/performanceTest/java-sdk-demo/dist"
NETEM="$HOME/netem.sh"
P2P_PORTS="30300-30309"
RPC="http://127.0.0.1:8545"
RUN_TIMEOUT=1800   # per-run timeout (s), prevents the script from hanging if transactions are never confirmed

export JAVA_HOME="/usr/lib/jvm/java-8-openjdk-amd64"
export PATH="$JAVA_HOME/bin:$PATH"

while [ $# -gt 0 ]; do
    case "$1" in
        --nodes)    NODES=$2; shift 2 ;;
        --rtts)     RTTS=$2; shift 2 ;;
        --ops)      OPS=$2; shift 2 ;;
        --count)    COUNT=$2; shift 2 ;;
        --qps)      QPS=$2; shift 2 ;;
        --repeat)   REPEAT=$2; shift 2 ;;
        --interval) INTERVAL=$2; shift 2 ;;
        --warmup)   WARMUP=$2; shift 2 ;;
        --warmup-run) WARMUP_RUN=$2; shift 2 ;;
        --out)      OUT=$2; shift 2 ;;
        *) sed -n '3,24p' "$0" | sed 's/^# \{0,1\}//'; exit 1 ;;
    esac
done
[ -z "$NODES" ] && { echo "Please specify the current number of consensus nodes with --nodes"; exit 1; }

# Contract address: use the environment variable if set, otherwise read the backend configuration
if [ -z "$DPKIACME_ADDR" ]; then
    DPKIACME_ADDR=$(tr -d ' \n\r' < "$HOME/DPKIACME/acmeClient/conf/contractAddress")
fi
export DPKIACME_ADDR

LOG_DIR="$(dirname "$OUT")/logs"
mkdir -p "$LOG_DIR"

rpc() {
    curl -s -m 5 -X POST -H 'Content-Type: application/json' \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"$1\",\"params\":[1],\"id\":1}" "$RPC"
}
block_number() { rpc getBlockNumber | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'],16))" 2>/dev/null; }
sealer_count() { rpc getSealerList | python3 -c "import sys,json; print(len(json.load(sys.stdin)['result']))" 2>/dev/null; }

# Measure the actual RTT of the node P2P port via TCP handshake time (average of 5, in ms)
measure_rtt() {
    local sum=0 t
    for _ in 1 2 3 4 5; do
        t=$(curl -s -o /dev/null -m 3 -w '%{time_connect}' "telnet://127.0.0.1:30300" </dev/null)
        sum=$(python3 -c "print($sum + $t)")
    done
    python3 -c "print(round($sum / 5 * 1000, 3))"
}

# ---------- Pre-checks ----------
# Domain control validation experiments require the baseline contract addresses
for OP in $OPS; do
    case "$OP" in
        dacme_*) [ -z "$DACME_ADDR" ] && { echo "[ERROR] Running $OP requires export DACME_ADDR=..."; exit 1; } ;;
        dcv_*)   [ -z "$DCV_ADDR" ]   && { echo "[ERROR] Running $OP requires export DCV_ADDR=..."; exit 1; } ;;
        scale_*) [ -z "$ACCOUNT_PEM" ] && { echo "[ERROR] Running $OP requires export ACCOUNT_PEM=..."; exit 1; }
                 [ "$OP" != "scale_verify" ] && [ -z "$SCALE_DOMAIN" ] && { echo "[ERROR] Running $OP requires export SCALE_DOMAIN=..."; exit 1; } ;;
    esac
done
export DACME_ADDR DCV_ADDR DV_DOMAINS ACCOUNT_PEM SCALE_DOMAIN SCALE_STATE
ACTUAL=$(sealer_count)
if [ "$ACTUAL" != "$NODES" ]; then
    echo "[ERROR] The chain has '$ACTUAL' consensus nodes, which does not match --nodes $NODES. Adjust it with exp_nodes.sh first."
    exit 1
fi
[ -f "$NETEM" ] || { echo "[ERROR] $NETEM not found"; exit 1; }
[ -d "$PERF_DIR" ] || { echo "[ERROR] Benchmark directory $PERF_DIR not found"; exit 1; }

echo "sudo privileges are required to configure network delay:"
sudo -v || exit 1
# Keep the sudo credential alive in the background so long experiments do not prompt for the password again
( while true; do sudo -n true; sleep 60; done ) 2>/dev/null &
KEEPALIVE=$!

cleanup() {
    sudo bash "$NETEM" clear >/dev/null 2>&1
    kill "$KEEPALIVE" 2>/dev/null
    echo "[INFO] Network delay rules cleared"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

if [ ! -f "$OUT" ]; then
    echo "timestamp,nodes,sealers,rtt_target_ms,rtt_measured_ms,op,run,count,qps,total,ok,err,time_ms,tps_all,tps_ok,avg_ms,p50_ms,p95_ms,p99_ms,max_ms,err_rate_pct,blocks" > "$OUT"
fi

echo "========== Starting experiment: $NODES consensus nodes =========="
echo "Contract address: $DPKIACME_ADDR    Result file: $OUT"

for RTT in $RTTS; do
    DELAY=$((RTT / 2))   # one-way delay = RTT / 2 (see netem.sh)
    if [ "$DELAY" -eq 0 ]; then
        sudo bash "$NETEM" clear >/dev/null
    else
        sudo bash "$NETEM" apply --delay "$DELAY" --ports "$P2P_PORTS" >/dev/null || exit 1
    fi
    echo "[INFO] RTT level ${RTT}ms set, waiting ${WARMUP}s to stabilize..."
    sleep "$WARMUP"
    MEASURED=$(measure_rtt)
    echo "[INFO] Measured inter-node RTT ≈ ${MEASURED}ms"

    if [ "$WARMUP_RUN" = "1" ]; then
        # Warm-up: removes the effect of JVM startup and cold node caches on the first run; not written to the CSV
        WCOUNT=$(( COUNT < 2000 ? COUNT : 2000 ))
        echo "[INFO] Warm-up run (not recorded)..."
        (cd "$PERF_DIR" && timeout "$RUN_TIMEOUT" java -cp 'conf/:lib/*:apps/*' \
            org.fisco.bcos.sdk.demo.perf.PerformanceDpkiAcme apply "$WCOUNT" "$QPS" 1) \
            > "$LOG_DIR/n${NODES}_q${QPS}_rtt${RTT}_warmup.log" 2>&1
        sleep "$INTERVAL"
    fi

    for OP in $OPS; do
        for ((RUN = 1; RUN <= REPEAT; RUN++)); do
            TAG="n${NODES}_q${QPS}_rtt${RTT}_${OP}_r${RUN}"
            LOG="$LOG_DIR/$TAG.log"
            H1=$(block_number)
            echo "[RUN ] $TAG ..."
            # Domain control validation (*_v1 / *_v2) uses PerformanceDomainValidation; scale_* uses PerformanceScale; other operations use PerformanceDpkiAcme
            case "$OP" in
                scale_*)   PERF_CLASS=org.fisco.bcos.sdk.demo.perf.PerformanceScale ;;
                *_v1|*_v2) PERF_CLASS=org.fisco.bcos.sdk.demo.perf.PerformanceDomainValidation ;;
                *)         PERF_CLASS=org.fisco.bcos.sdk.demo.perf.PerformanceDpkiAcme ;;
            esac
            (cd "$PERF_DIR" && timeout "$RUN_TIMEOUT" java -cp 'conf/:lib/*:apps/*' \
                "$PERF_CLASS" "$OP" "$COUNT" "$QPS" 1) > "$LOG" 2>&1
            H2=$(block_number)
            LINE=$(grep -a "^RESULT_CSV," "$LOG" | tail -n 1)
            TS=$(date '+%Y-%m-%d %H:%M:%S')
            if [ -n "$LINE" ]; then
                echo "$TS,$NODES,$ACTUAL,$RTT,$MEASURED,$OP,$RUN,$COUNT,$QPS,${LINE#RESULT_CSV,},$((H2 - H1))" >> "$OUT"
                echo "       $(echo "$LINE" | awk -F, '{printf "TPS(success)=%s  p50=%sms  p95=%sms  error rate=%s%%", $7, $9, $10, $13}')"
            else
                echo "$TS,$NODES,$ACTUAL,$RTT,$MEASURED,$OP,$RUN,$COUNT,$QPS,FAILED,,,,,,,,,,,,$((H2 - H1))" >> "$OUT"
                echo "       [FAILED] No result obtained, see $LOG"
            fi
            sleep "$INTERVAL"
        done
    done
done

echo "========== Finished: $NODES consensus nodes =========="
