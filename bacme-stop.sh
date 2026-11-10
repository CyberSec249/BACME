#!/bin/bash
# =============================================================
# One-click stop script for the BACME project
# Stop order (reverse of start): frontend -> backend -> IPFS -> Truora -> FISCO BCOS 2.x
# MySQL is a system service and is not stopped by default; to stop it as well: bash ~/bacme-stop.sh --all
# =============================================================

# ---------- Paths (must match the start script) ----------
FISCO_DIR="$HOME/fisco2/nodes/127.0.0.1"
TRUORA_DIR="$HOME/Truora-Service/dist"
WEB_PORT=5173

export JAVA_HOME="/usr/lib/jvm/java-8-openjdk-amd64"
export PATH="$JAVA_HOME/bin:/usr/local/bin:$PATH"

GREEN="\033[32m"; YELLOW="\033[33m"; NC="\033[0m"
info() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }

port_up() { ss -lnt 2>/dev/null | awk '{print $4}' | grep -qE "[:.]$1\$"; }

# Stop a process: terminate gracefully, then force-kill if still alive after 5 seconds
stop_proc() {
    local pattern=$1 name=$2
    if pgrep -f "$pattern" >/dev/null; then
        pkill -f "$pattern"
        for _ in 1 2 3 4 5; do
            pgrep -f "$pattern" >/dev/null || break
            sleep 1
        done
        pgrep -f "$pattern" >/dev/null && pkill -9 -f "$pattern"
        info "$name stopped"
    else
        info "$name is not running, skipping"
    fi
}

echo "================ Stopping the BACME project ================"

# ---------- 1. Frontend ----------
stop_proc "http.server $WEB_PORT" "Frontend"

# ---------- 2. Backend ----------
stop_proc "^\./acme-server" "Backend acme-server"

# ---------- 3. IPFS ----------
if port_up 5001; then
    ipfs shutdown >/dev/null 2>&1
    sleep 2
fi
stop_proc "ipfs daemon" "IPFS"

# ---------- 4. Truora-Service ----------
if port_up 5021; then
    (cd "$TRUORA_DIR" && bash stop.sh >/dev/null 2>&1)
    sleep 3
    info "Truora-Service stopped"
else
    info "Truora-Service is not running, skipping"
fi

# ---------- 5. FISCO BCOS 2.x ----------
if pgrep -f "fisco2/nodes/127.0.0.1" >/dev/null; then
    bash "$FISCO_DIR/stop_all.sh"
    info "FISCO BCOS 2.x nodes stopped"
else
    info "FISCO BCOS 2.x is not running, skipping"
fi

# ---------- 6. MySQL (optional) ----------
if [ "$1" == "--all" ]; then
    info "Stopping MySQL (the sudo password may be required)..."
    sudo systemctl stop mysql
    info "MySQL stopped"
else
    warn "MySQL is left running (to stop it: bash ~/bacme-stop.sh --all)"
fi

echo "================================================"
info "BACME project stopped"
