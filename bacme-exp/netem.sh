#!/bin/bash
# =============================================================
# Network delay emulation script (based on Linux tc netem)
#
# Two modes:
#   1) Single-host mode (default): all nodes run on 127.0.0.1; delay is added only to the
#      inter-node P2P ports (default 30300-30303), while client<->node RPC/channel
#      traffic is unaffected. This emulates consortium members deployed across regions, with clients connecting to their local organization's node.
#   2) Multi-host mode (--dev <interface> --all): for one node per machine;
#      adds delay to all egress traffic of the interface. Run it once on every machine.
#
# Delay conversion:
#   In single-host mode, packets in both directions (A->B and B->A) traverse lo once, so
#   a one-way delay of D ms yields an inter-node RTT of about 2D. E.g., to emulate RTT=100ms, use --delay 50.
#   In multi-host mode, setting D on both machines gives RTT ≈ 2D; setting it on only one gives RTT ≈ D.
#
# Examples:
#   sudo bash netem.sh apply --delay 50                         # single host, inter-node RTT ≈ 100ms
#   sudo bash netem.sh apply --delay 50 --jitter 10 --loss 0.1  # add jitter and packet loss
#   sudo bash netem.sh apply --delay 25 --rate 100mbit          # add a bandwidth limit
#   sudo bash netem.sh apply --delay 50 --ports 30300-30306     # 7 nodes
#   sudo bash netem.sh apply --dev ens33 --all --delay 50       # multi-host mode
#   sudo bash netem.sh show                                     # show current rules
#   sudo bash netem.sh verify                                   # measure the actual RTT of node ports
#   sudo bash netem.sh clear                                    # remove all rules
# =============================================================

DEV="lo"
DELAY=0
JITTER=0
LOSS=0
RATE=""
PORT_RANGE="30300-30303"
ALL=false
LIMIT=100000   # netem queue length; too small a value causes extra drops under high concurrency

usage() { sed -n '3,26p' "$0" | sed 's/^# \{0,1\}//'; exit 1; }

[ $# -lt 1 ] && usage
CMD=$1; shift

while [ $# -gt 0 ]; do
    case "$1" in
        --dev)    DEV=$2; shift 2 ;;
        --delay)  DELAY=$2; shift 2 ;;
        --jitter) JITTER=$2; shift 2 ;;
        --loss)   LOSS=$2; shift 2 ;;
        --rate)   RATE=$2; shift 2 ;;
        --ports)  PORT_RANGE=$2; shift 2 ;;
        --all)    ALL=true; shift ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

if [ "$(id -u)" -ne 0 ]; then
    echo "Please run with sudo: sudo bash $0 $CMD ..."
    exit 1
fi

clear_rules() {
    tc qdisc del dev "$DEV" root 2>/dev/null
}

netem_args() {
    local args="delay ${DELAY}ms"
    [ "$JITTER" != "0" ] && args="$args ${JITTER}ms distribution normal"
    [ "$LOSS" != "0" ] && args="$args loss ${LOSS}%"
    [ -n "$RATE" ] && args="$args rate ${RATE}"
    echo "$args limit $LIMIT"
}

apply_rules() {
    clear_rules
    local NETEM
    NETEM=$(netem_args)

    if $ALL; then
        # Delay all traffic of the interface (multi-host mode)
        tc qdisc add dev "$DEV" root handle 1: netem $NETEM || exit 1
        echo "[OK] All egress traffic on $DEV: $NETEM"
    else
        # prio qdisc: bands 0-2 use the default path (no delay); band 3 (1:4) is attached to netem
        tc qdisc add dev "$DEV" root handle 1: prio bands 4 \
            priomap 1 2 2 2 1 2 0 0 1 1 1 1 1 1 1 1 || exit 1
        tc qdisc add dev "$DEV" parent 1:4 handle 40: netem $NETEM || exit 1

        local START=${PORT_RANGE%-*} END=${PORT_RANGE#*-}
        for ((p = START; p <= END; p++)); do
            # Delay packets both destined to and sent from the P2P ports
            tc filter add dev "$DEV" parent 1:0 protocol ip prio 1 u32 \
                match ip dport "$p" 0xffff flowid 1:4
            tc filter add dev "$DEV" parent 1:0 protocol ip prio 1 u32 \
                match ip sport "$p" 0xffff flowid 1:4
        done
        echo "[OK] Traffic of P2P ports $PORT_RANGE on $DEV: $NETEM"
        echo "     Inter-node RTT is about $((DELAY * 2)) ms (client<->node traffic is unaffected)"
    fi
}

show_rules() {
    echo "== qdisc =="; tc qdisc show dev "$DEV"
    echo "== filter =="; tc filter show dev "$DEV" | grep -E "match|flowid" | head -40
}

verify_rtt() {
    # Measure the actual RTT of the node P2P port via TCP handshake time (SYN and SYN-ACK are each delayed once)
    local START=${PORT_RANGE%-*}
    echo "Measuring TCP handshake time to 127.0.0.1:$START (5 times):"
    for _ in 1 2 3 4 5; do
        curl -s -o /dev/null -m 3 -w '  connect: %{time_connect}s\n' "telnet://127.0.0.1:$START" </dev/null
    done
}

case "$CMD" in
    apply)  apply_rules ;;
    clear)  clear_rules; echo "[OK] All tc rules on $DEV removed" ;;
    show)   show_rules ;;
    verify) verify_rtt ;;
    *) usage ;;
esac
