#!/bin/bash
# =============================================================
# Pre-populate certificate records on chain until the total reaches the target (for the large certificate state experiment)
#
# Usage: bash seed_scale.sh TARGET [BATCH] [QPS]
#   TARGET  target total number of pre-populated records, e.g., 1000000
#   BATCH   records per batch (default 100000; one Java process per batch to limit memory usage)
#   QPS     maximum sending rate (default 3000; in-flight transactions are additionally limited by SCALE_INFLIGHT)
#
# Required environment variables: DPKIACME_ADDR, ACCOUNT_PEM, SCALE_DOMAIN, SCALE_STATE (absolute path recommended)
# Progress is stored in the SCALE_STATE file; rerunning after an interruption resumes from there.
# =============================================================
set -u
TARGET=${1:?Usage: bash seed_scale.sh TARGET [BATCH] [QPS]}
BATCH=${2:-100000}
QPS=${3:-3000}
PERF_DIR=${PERF_DIR:-$HOME/DPKIACME/performanceTest/java-sdk-demo/dist}
: "${DPKIACME_ADDR:?Please export DPKIACME_ADDR=... first}"
: "${ACCOUNT_PEM:?Please export ACCOUNT_PEM=... first}"
: "${SCALE_DOMAIN:?Please export SCALE_DOMAIN=... first}"
export SCALE_STATE=${SCALE_STATE:-$HOME/scale_state.txt}
export DPKIACME_ADDR ACCOUNT_PEM SCALE_DOMAIN
LOG_DIR=${LOG_DIR:-$HOME/scale-seed-logs}
mkdir -p "$LOG_DIR"

current() { [ -f "$SCALE_STATE" ] && tr -d '[:space:]' < "$SCALE_STATE" || echo 0; }

cd "$PERF_DIR" || { echo "[ERROR] $PERF_DIR not found"; exit 1; }
while true; do
    CUR=$(current); CUR=${CUR:-0}
    if [ "$CUR" -ge "$TARGET" ]; then
        echo "[DONE] $CUR records pre-populated (target $TARGET)"
        break
    fi
    N=$(( TARGET - CUR < BATCH ? TARGET - CUR : BATCH ))
    FREE=$(df -BG --output=avail / | tail -n 1 | tr -dc '0-9')
    if [ "${FREE:-0}" -lt 5 ]; then
        echo "[STOP] Only ${FREE}G of disk space left, stopping"; exit 1
    fi
    LOG="$LOG_DIR/seed_${CUR}_$((CUR + N)).log"
    echo "[$(date '+%H:%M:%S')] Writing records $CUR ~ $((CUR + N)) (${FREE}G disk available)..."
    java -cp 'conf/:lib/*:apps/*' org.fisco.bcos.sdk.demo.perf.PerformanceScale scale_seed "$N" "$QPS" 1 > "$LOG" 2>&1
    RES=$(grep -a "^SEED_DONE" "$LOG" | tail -n 1)
    echo "           $RES"
    if [ -z "$RES" ] || ! echo "$RES" | grep -q " err=0 "; then
        echo "[ERROR] This batch failed or did not complete; stopped. See log: $LOG"
        grep -a "\[error\]\|failed\|Exception" "$LOG" | head -n 5
        exit 1
    fi
done
