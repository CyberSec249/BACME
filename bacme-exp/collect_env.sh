#!/bin/bash
# =============================================================
# Collect experiment environment information (hardware, OS, FISCO BCOS consensus parameters) into env.txt
# Usage: bash collect_env.sh [output file, default ~/exp-results/env.txt]
# =============================================================

export LC_ALL=C   # force English output; on a localized system the lscpu/free field names would not match
OUT=${1:-$HOME/exp-results/env.txt}
NODE_DIR="$HOME/fisco2/nodes/127.0.0.1"
mkdir -p "$(dirname "$OUT")"

{
echo "Collected at: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo
echo "== Hardware =="
lscpu | grep -E "^Model name|^CPU\(s\)|^Thread\(s\) per core|^Core\(s\) per socket|^Socket\(s\)|^Hypervisor vendor|^Virtualization type|^CPU max MHz"
echo "Memory: $(free -h | awk '/^Mem:/{print $2}')"
echo "Disk: $(df -h "$HOME" | awk 'NR==2{print $2" total, "$4" available"}')"
echo
echo "== OS and software =="
echo "OS: $(. /etc/os-release && echo "$PRETTY_NAME")"
echo "Kernel: $(uname -r)"
echo "FISCO BCOS: $("$NODE_DIR/fisco-bcos" -v 2>/dev/null | grep -i "Version" | head -n 1 | sed 's/^ *//')"
echo "Java: $(java -version 2>&1 | head -n 1)"
echo "Go: $(go version 2>/dev/null || /usr/local/go/bin/go version 2>/dev/null)"
echo
echo "== Deployment =="
echo "All nodes and the benchmark client are deployed on the same virtual machine; inter-node traffic goes through the loopback interface (127.0.0.1)"
echo "Number of node directories: $(ls -d "$NODE_DIR"/node* 2>/dev/null | wc -l)"
echo "Running node processes: $(pgrep -c -x fisco-bcos)"
echo "Baseline inter-node RTT (ping 127.0.0.1, 20 times): $(ping -c 20 -i 0.2 127.0.0.1 2>/dev/null | tail -n 1)"
echo
echo "== Consensus parameters (node0/conf/group.1.genesis) =="
grep -vE "^\s*;|^\s*$|node\.[0-9]+=" "$NODE_DIR/node0/conf/group.1.genesis"
echo
echo "== Group runtime parameters (consensus / tx_pool / sync sections of node0/conf/group.1.ini) =="
awk '/^\[consensus\]|^\[tx_pool\]|^\[sync\]/{p=1; print; next} /^\[/{p=0} p && !/^\s*;/ && NF' "$NODE_DIR/node0/conf/group.1.ini"
} > "$OUT" 2>&1

echo "Environment information written to: $OUT"
cat "$OUT"
