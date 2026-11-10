#!/bin/bash
# =============================================================
# One-click start script for the BACME project
# Start order: FISCO BCOS 2.x -> MySQL -> Truora -> IPFS -> backend -> frontend
# Usage: bash ~/bacme-start.sh
# =============================================================

# ---------- Paths (must match the deployment; adjust if different) ----------
FISCO_DIR="$HOME/fisco2/nodes/127.0.0.1"          # FISCO BCOS 2.x node directory
FISCO3_DIR="$HOME/fisco/nodes/127.0.0.1"          # legacy 3.x chain (stopped if running, to avoid port conflicts)
TRUORA_DIR="$HOME/Truora-Service/dist"            # Truora-Service directory
PROJECT_DIR="$HOME/DPKIACME"                      # project directory
ACME_DIR="$PROJECT_DIR/acmeClient"                # backend directory
WEB_DIR="$PROJECT_DIR/webui/dist"                 # frontend build directory
WEB_PORT=5173                                     # frontend port
LOG_DIR="$HOME/bacme-logs"                        # log directory for IPFS/backend/frontend

export JAVA_HOME="/usr/lib/jvm/java-8-openjdk-amd64"
export PATH="$JAVA_HOME/bin:/usr/local/go/bin:$HOME/go/bin:/usr/local/bin:$PATH"

mkdir -p "$LOG_DIR"

# ---------- Helper functions ----------
GREEN="\033[32m"; RED="\033[31m"; YELLOW="\033[33m"; NC="\033[0m"
info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Whether a port is listening
port_up() { ss -lnt 2>/dev/null | awk '{print $4}' | grep -qE "[:.]$1\$"; }

# Wait for a port to listen: wait_port <port> <timeout in seconds> <service name>
wait_port() {
    local port=$1 timeout=$2 name=$3 i=0
    while ! port_up "$port"; do
        sleep 2; i=$((i + 2))
        if [ $i -ge "$timeout" ]; then
            error "$name did not start within ${timeout} seconds (port $port is not listening)"
            return 1
        fi
    done
    info "$name is ready (port $port)"
}

echo "================ Starting the BACME project ================"

# ---------- 1. FISCO BCOS 2.x ----------
if pgrep -f "fisco/nodes/127.0.0.1" >/dev/null; then
    warn "A legacy FISCO 3.x chain is running; stopping it to avoid port conflicts"
    bash "$FISCO3_DIR/stop_all.sh" >/dev/null 2>&1
    sleep 3
fi

NODE_NUM=$(pgrep -f "fisco2/nodes/127.0.0.1" | wc -l)
if [ "$NODE_NUM" -ge 4 ]; then
    info "FISCO BCOS 2.x is already running ($NODE_NUM nodes), skipping"
else
    info "Starting FISCO BCOS 2.x nodes..."
    bash "$FISCO_DIR/start_all.sh"
    wait_port 20200 60 "FISCO BCOS node" || exit 1
    sleep 5   # wait for the nodes to connect to each other
fi

# ---------- 2. MySQL ----------
if systemctl is-active --quiet mysql; then
    info "MySQL is already running, skipping"
else
    info "Starting MySQL (the sudo password may be required)..."
    sudo systemctl start mysql
    wait_port 3306 30 "MySQL" || exit 1
fi

# ---------- 3. Truora-Service ----------
if port_up 5021; then
    info "Truora-Service is already running, skipping"
else
    info "Starting Truora-Service (takes about 30-90 seconds)..."
    (cd "$TRUORA_DIR" && bash start.sh >/dev/null 2>&1)
    wait_port 5021 180 "Truora-Service" || {
        error "See the log: tail -n 50 $TRUORA_DIR/log/Oracle-Service-error.log"
        exit 1
    }
fi

# ---------- 4. IPFS ----------
if port_up 5001; then
    info "IPFS is already running, skipping"
else
    info "Starting IPFS..."
    nohup ipfs daemon > "$LOG_DIR/ipfs.log" 2>&1 &
    wait_port 5001 60 "IPFS" || { error "See the log: $LOG_DIR/ipfs.log"; exit 1; }
fi

# ---------- 5. Backend acme-server ----------
if port_up 8080; then
    info "Backend acme-server is already running (or port 8080 is in use), skipping"
else
    cd "$ACME_DIR" || { error "Directory $ACME_DIR not found"; exit 1; }
    if [ ! -x ./acme-server ]; then
        info "acme-server not found, building it..."
        go build -o acme-server cmd/main.go || { error "Build failed"; exit 1; }
        sudo setcap 'cap_net_bind_service=+ep' ./acme-server
    fi
    info "Starting backend acme-server..."
    nohup ./acme-server > "$LOG_DIR/acme-server.log" 2>&1 &
    wait_port 8080 60 "Backend API" || { error "See the log: $LOG_DIR/acme-server.log"; exit 1; }
    wait_port 80 10 "Backend validation file server" || warn "Port 80 is not listening; domain control validation will fail"
fi

# ---------- 6. Frontend ----------
if port_up "$WEB_PORT"; then
    info "Frontend is already running, skipping"
else
    if [ ! -f "$WEB_DIR/index.html" ]; then
        error "Frontend file $WEB_DIR/index.html not found"
        exit 1
    fi
    info "Starting frontend..."
    (cd "$WEB_DIR" && nohup python3 -m http.server "$WEB_PORT" > "$LOG_DIR/webui.log" 2>&1 &)
    wait_port "$WEB_PORT" 20 "Frontend" || exit 1
fi

echo "================================================"
info "All services started!"
echo "  Frontend:    http://localhost:$WEB_PORT  (open it in a browser inside the VM)"
echo "  Backend log: tail -f $LOG_DIR/acme-server.log"
echo "  Truora log:  tail -f $TRUORA_DIR/log/Oracle-Service.log"
echo "  IPFS log:    tail -f $LOG_DIR/ipfs.log"
echo "  Stop:        bash ~/bacme-stop.sh"
