#!/bin/bash
# =============================================================
# Helper for changing the number of consensus nodes (FISCO BCOS 2.x, 10-node chain on a single host)
#
# Usage:
#   bash exp_nodes.sh show            # show the status and NodeID of each node
#   bash exp_nodes.sh sealers         # query the current number of consensus nodes (sealers)
#   bash exp_nodes.sh remove-cmds N   # print the console commands to shrink to N consensus nodes
#   bash exp_nodes.sh stop-from N     # stop node processes with index >= N
#   bash exp_nodes.sh start-upto N    # start node processes with index < N (for scaling up)
#   bash exp_nodes.sh heights         # show the block height of each running node (to check that new nodes have synced)
#   bash exp_nodes.sh add-cmds A B    # print the console commands to grow from A to B consensus nodes
#
# Scaling up (e.g., 4 -> 7): start-upto 7 -> run addObserver in the console -> check heights until synced
#                          -> run addSealer in the console -> check that sealers shows 7
#
# Scaling down (important): run removeNode in the console first, then stop the processes.
# If the processes are stopped first, the remaining nodes cannot form the 2f+1 PBFT quorum and the chain halts.
#   10 -> 7: remove-cmds 7  ->  run the printed commands in the console  ->  stop-from 7
#    7 -> 4: remove-cmds 4  ->  run the printed commands in the console  ->  stop-from 4
# =============================================================

NODES_DIR="$HOME/fisco2/nodes/127.0.0.1"
RPC="http://127.0.0.1:8545"
MAX_NODES=10

rpc() {
    curl -s -m 5 -X POST -H 'Content-Type: application/json' \
        --data "{\"jsonrpc\":\"2.0\",\"method\":\"$1\",\"params\":[1],\"id\":1}" "$RPC"
}

node_running() { pgrep -f "$NODES_DIR/node$1/" >/dev/null; }

case "$1" in
show)
    for ((i = 0; i < MAX_NODES; i++)); do
        [ -d "$NODES_DIR/node$i" ] || continue
        if node_running "$i"; then st="running"; else st="stopped"; fi
        printf "node%-2s %-6s %s\n" "$i" "$st" "$(cat "$NODES_DIR/node$i/conf/node.nodeid")"
    done
    ;;
sealers)
    rpc getSealerList | python3 -c "import sys,json; r=json.load(sys.stdin)['result']; print('Current number of consensus nodes:', len(r))"
    ;;
remove-cmds)
    N=$2
    [ -z "$N" ] && { echo "Usage: bash $0 remove-cmds N"; exit 1; }
    echo "# Run the following commands one by one in the FISCO 2.x console (wait for success before running the next):"
    for ((i = MAX_NODES - 1; i >= N; i--)); do
        [ -f "$NODES_DIR/node$i/conf/node.nodeid" ] || continue
        echo "removeNode $(cat "$NODES_DIR/node$i/conf/node.nodeid")"
    done
    echo "getSealerList"
    ;;
stop-from)
    N=$2
    [ -z "$N" ] && { echo "Usage: bash $0 stop-from N"; exit 1; }
    for ((i = N; i < MAX_NODES; i++)); do
        if node_running "$i"; then
            bash "$NODES_DIR/node$i/stop.sh" >/dev/null 2>&1
            echo "node$i stopped"
        fi
    done
    ;;
start-upto)
    N=$2
    [ -z "$N" ] && { echo "Usage: bash $0 start-upto N"; exit 1; }
    for ((i = 0; i < N; i++)); do
        if ! node_running "$i"; then
            bash "$NODES_DIR/node$i/start.sh" >/dev/null 2>&1
            echo "node$i started"
        fi
    done
    ;;
heights)
    for ((i = 0; i < MAX_NODES; i++)); do
        node_running "$i" || continue
        h=$(curl -s -m 3 -X POST --data '{"jsonrpc":"2.0","method":"getBlockNumber","params":[1],"id":1}' \
            "http://127.0.0.1:$((8545 + i))" | python3 -c "import sys,json; print(int(json.load(sys.stdin)['result'],16))" 2>/dev/null)
        echo "node$i block height: ${h:-query failed}"
    done
    ;;
add-cmds)
    A=$2; B=$3
    [ -z "$B" ] && { echo "Usage: bash $0 add-cmds <current node count> <target node count>"; exit 1; }
    echo "# Step 1: run these one by one in the console (add as observer nodes, which sync blocks but do not take part in consensus):"
    for ((i = A; i < B; i++)); do echo "addObserver $(cat "$NODES_DIR/node$i/conf/node.nodeid")"; done
    echo
    echo "# Step 2: exit the console and run bash $0 heights; once the new nodes reach the same block height as node0,"
    echo "#         return to the console and run these one by one (promote to consensus nodes):"
    for ((i = A; i < B; i++)); do echo "addSealer $(cat "$NODES_DIR/node$i/conf/node.nodeid")"; done
    echo "getSealerList"
    ;;
*)
    sed -n '3,20p' "$0" | sed 's/^# \{0,1\}//'
    ;;
esac
