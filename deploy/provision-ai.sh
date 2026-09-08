#!/usr/bin/env bash
# =============================================================================
# ShiftMaster local AI provisioning
# =============================================================================
# Installs everything the assistant needs to run a language model ON THIS
# MACHINE: the llama.cpp inference server, a model file sized to the hardware
# it actually found, and the configuration lines to paste into .env.
#
# Run it once per server, before or after deploy.sh:
#
#     sudo ./deploy/provision-ai.sh
#
# It is the ONLY step that touches the network for AI purposes. After it
# finishes, the assistant answers with no internet connection at all: no
# OpenAI, no Anthropic, no Gemini, no hosted inference of any kind, no API key
# to buy or rotate. If this machine is offline, see "AIR-GAPPED" at the bottom.
#
# Nothing here is destructive: an existing model file is kept, an existing .env
# is never edited (the settings are printed for you to paste), and the running
# API is not restarted.
# =============================================================================

set -euo pipefail

MODEL_DIR="${AI_MODEL_DIR:-/opt/shiftmaster/models}"
BIN_DIR="${AI_BIN_DIR:-/opt/shiftmaster/bin}"
SERVICE_USER="${SERVICE_USER:-techsupport}"
LLAMA_VERSION="${LLAMA_VERSION:-b7062}"

say()  { printf '%s\n' "$*"; }
info() { printf '[INFO] %s\n' "$*"; }
warn() { printf '[WARN] %s\n' "$*" >&2; }
die()  { printf '[FATAL] %s\n' "$*" >&2; exit 1; }

# ── 1. Inspect the machine ───────────────────────────────────────────────────
# Model choice follows the hardware, not a preference. Getting this wrong in
# either direction is expensive: too large and the server swaps itself to a
# standstill under load, too small and the assistant misunderstands people.

say "============================================="
say "   ShiftMaster local AI — provisioning       "
say "============================================="
say ""
info "Inspecting this machine..."

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Linux)
        CPU_CORES="$(nproc)"
        RAM_MB="$(awk '/MemTotal/ {printf "%d", $2/1024}' /proc/meminfo)"
        CPU_MODEL="$(awk -F: '/model name/ {print $2; exit}' /proc/cpuinfo | sed 's/^ *//')"
        DISK_FREE_GB="$(df -BG --output=avail "$(dirname "$MODEL_DIR")" 2>/dev/null | tail -1 | tr -dc '0-9' || echo 0)"
        ;;
    Darwin)
        CPU_CORES="$(sysctl -n hw.ncpu)"
        RAM_MB="$(( $(sysctl -n hw.memsize) / 1024 / 1024 ))"
        CPU_MODEL="$(sysctl -n machdep.cpu.brand_string)"
        DISK_FREE_GB="$(df -g "$(dirname "$MODEL_DIR")" 2>/dev/null | tail -1 | awk '{print $4}' || echo 0)"
        ;;
    *)
        die "unsupported OS: $OS"
        ;;
esac

# GPU: an NVIDIA card changes both the model tier and the layer offload.
GPU_NAME=""
VRAM_MB=0
if command -v nvidia-smi >/dev/null 2>&1; then
    GPU_NAME="$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 || true)"
    VRAM_MB="$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || echo 0)"
fi
# Apple silicon shares system memory with the GPU.
if [ "$OS" = "Darwin" ] && [ "$ARCH" = "arm64" ]; then
    GPU_NAME="Apple silicon (Metal, unified memory)"
    VRAM_MB="$RAM_MB"
fi

say ""
say "  OS ............ $OS $ARCH"
say "  CPU ........... ${CPU_MODEL:-unknown} (${CPU_CORES} cores)"
say "  RAM ........... ${RAM_MB} MB"
say "  GPU ........... ${GPU_NAME:-none detected}"
say "  VRAM .......... ${VRAM_MB} MB"
say "  Disk free ..... ${DISK_FREE_GB} GB at $(dirname "$MODEL_DIR")"
say ""

# ── 2. Choose the model ──────────────────────────────────────────────────────
# Budget: the model file must fit in memory alongside the KV cache, the API,
# PostgreSQL and the OS. The rule below reserves roughly 6 GB for everything
# that is not the model, and never spends more than ~55% of RAM on weights.
#
# The candidates are all permissively licensed, all strong at instruction
# following and tool calling, and all genuinely multilingual — which for this
# deployment means they read Iraqi Arabic and code-switched Arabic/English,
# not merely that they have seen Modern Standard Arabic.

if [ "$VRAM_MB" -ge 20000 ]; then
    BUDGET_MB=$(( VRAM_MB * 60 / 100 ))
elif [ "$VRAM_MB" -ge 8000 ] && [ "$OS" != "Darwin" ]; then
    BUDGET_MB=$(( VRAM_MB * 75 / 100 ))
else
    BUDGET_MB=$(( (RAM_MB - 6000) * 55 / 100 ))
fi

# The tiers, largest first; the first one that fits the budget wins.
#
# The 9B row is the one the evaluation suite in internal/assistant/eval_test.go
# was measured against — Iraqi Arabic, code-switching, misspellings, overnight
# leave, adversarial prompts and an injected document. Treat it as the known
# quantity: the tiers above it are stronger and the tier below it is measurably
# weaker at choosing between many tools in Arabic, which is the hardest thing
# this assistant asks of a model.
#
# name | repo | file | approx MB | per-slot context | note
TIERS=$(cat <<'TIERS'
Qwen3.5-14B-Q4_K_M|unsloth/Qwen3.5-14B-GGUF|Qwen3.5-14B-Q4_K_M.gguf|9000|16384|strongest tier; needs a GPU or a large server
Qwen3.5-9B-Q4_K_M|unsloth/Qwen3.5-9B-GGUF|Qwen3.5-9B-Q4_K_M.gguf|5700|16384|recommended default: best quality that runs comfortably on a normal server
Qwen3.5-4B-Q4_K_M|unsloth/Qwen3.5-4B-GGUF|Qwen3.5-4B-Q4_K_M.gguf|2800|12288|constrained hardware; noticeably weaker at multi-step tool use
Qwen3.5-4B-Q4_K_M-small|unsloth/Qwen3.5-4B-GGUF|Qwen3.5-4B-Q4_K_M.gguf|2800|8192|minimum viable; expect shorter conversations
TIERS
)

CHOSEN=""
while IFS='|' read -r name repo file size ctx note; do
    [ -z "$name" ] && continue
    if [ "$size" -le "$BUDGET_MB" ]; then
        CHOSEN="$name|$repo|$file|$size|$ctx|$note"
        break
    fi
done <<< "$TIERS"

if [ -z "$CHOSEN" ]; then
    say ""
    warn "This machine has ${RAM_MB} MB of RAM. After leaving room for PostgreSQL,"
    warn "the API and the OS, there is not enough left for even the smallest model"
    warn "the assistant can use responsibly."
    warn ""
    warn "ShiftMaster itself is unaffected — every other feature works. The"
    warn "assistant will report itself unavailable until this machine has more"
    warn "memory, or until AI_BASE_URL points at another machine on the private"
    warn "network that does have it."
    exit 1
fi

IFS='|' read -r MODEL_NAME MODEL_REPO MODEL_FILE MODEL_MB MODEL_CTX MODEL_NOTE <<< "$CHOSEN"

# Slots: how many people can be mid-answer at once. Each costs context memory
# and competes for the same cores, so this follows the machine too.
if [ "$VRAM_MB" -ge 16000 ]; then
    PARALLEL=4
elif [ "$CPU_CORES" -ge 8 ] && [ "$RAM_MB" -ge 24000 ]; then
    PARALLEL=3
elif [ "$RAM_MB" -ge 12000 ]; then
    PARALLEL=2
else
    PARALLEL=1
fi

# Layer offload: everything the accelerator can hold, or pure CPU without one.
if [ -n "$GPU_NAME" ]; then GPU_LAYERS=-1; else GPU_LAYERS=0; fi

info "Selected model: $MODEL_NAME"
say "        reason: $MODEL_NOTE"
say "        memory budget for weights: ${BUDGET_MB} MB, model needs ~${MODEL_MB} MB"
say "        context: ${MODEL_CTX} tokens per slot × ${PARALLEL} slot(s)"
say "        gpu layers: ${GPU_LAYERS} (0 = CPU only, -1 = as many as fit)"
say ""

REQUIRED_GB=$(( MODEL_MB / 1024 + 2 ))
if [ "${DISK_FREE_GB:-0}" -lt "$REQUIRED_GB" ]; then
    die "need about ${REQUIRED_GB} GB free at $(dirname "$MODEL_DIR"), found ${DISK_FREE_GB} GB"
fi

# ── 3. Install the inference server ──────────────────────────────────────────

install_llama() {
    if command -v llama-server >/dev/null 2>&1; then
        info "llama-server already installed: $(command -v llama-server)"
        return
    fi
    if [ -x "$BIN_DIR/llama-server" ]; then
        info "llama-server already installed: $BIN_DIR/llama-server"
        return
    fi

    info "Installing llama.cpp..."
    case "$OS" in
        Darwin)
            command -v brew >/dev/null 2>&1 || die "Homebrew is required on macOS: https://brew.sh"
            brew install llama.cpp
            ;;
        Linux)
            # Prefer the distribution package when one exists; otherwise take
            # the project's own prebuilt release, which needs no toolchain.
            if command -v apt-get >/dev/null 2>&1 && apt-cache show llama.cpp >/dev/null 2>&1; then
                apt-get update -qq && apt-get install -y llama.cpp
            else
                command -v curl >/dev/null 2>&1 || die "curl is required"
                command -v unzip >/dev/null 2>&1 || apt-get install -y unzip || die "unzip is required"
                local asset
                case "$ARCH" in
                    x86_64) asset="llama-${LLAMA_VERSION}-bin-ubuntu-x64.zip" ;;
                    aarch64|arm64) asset="llama-${LLAMA_VERSION}-bin-ubuntu-arm64.zip" ;;
                    *) die "no prebuilt llama.cpp for $ARCH — build from source: https://github.com/ggml-org/llama.cpp" ;;
                esac
                mkdir -p "$BIN_DIR"
                local tmp
                tmp="$(mktemp -d)"
                info "Downloading llama.cpp ${LLAMA_VERSION} (${asset})..."
                curl -fL --retry 3 -o "$tmp/llama.zip" \
                    "https://github.com/ggml-org/llama.cpp/releases/download/${LLAMA_VERSION}/${asset}" \
                    || die "download failed — check the release name at https://github.com/ggml-org/llama.cpp/releases"
                unzip -q -o "$tmp/llama.zip" -d "$tmp/x"
                find "$tmp/x" -type f \( -name 'llama-server' -o -name '*.so*' \) -exec cp {} "$BIN_DIR/" \;
                chmod 0755 "$BIN_DIR/llama-server"
                rm -rf "$tmp"
                info "Installed to $BIN_DIR/llama-server"
            fi
            ;;
    esac
}
install_llama

LLAMA_BIN="$(command -v llama-server || echo "$BIN_DIR/llama-server")"
[ -x "$LLAMA_BIN" ] || die "llama-server was not installed"

# ── 4. Fetch the model ───────────────────────────────────────────────────────
# One download, once. The application never fetches a model at runtime and
# never reaches the network to answer a question.

mkdir -p "$MODEL_DIR"
TARGET="$MODEL_DIR/$MODEL_FILE"

if [ -f "$TARGET" ] && [ "$(stat -c%s "$TARGET" 2>/dev/null || stat -f%z "$TARGET")" -gt 100000000 ]; then
    info "Model already present: $TARGET"
else
    info "Downloading $MODEL_FILE (~${MODEL_MB} MB) — this is the only network step."
    curl -fL --retry 3 -C - -o "$TARGET.partial" \
        "https://huggingface.co/${MODEL_REPO}/resolve/main/${MODEL_FILE}" \
        || die "model download failed"
    mv "$TARGET.partial" "$TARGET"
    info "Downloaded to $TARGET"
fi

# The model is readable only by the account that serves it.
if id -u "$SERVICE_USER" >/dev/null 2>&1; then
    chown -R "$SERVICE_USER":"$SERVICE_USER" "$MODEL_DIR" 2>/dev/null || true
fi
chmod 0640 "$TARGET" 2>/dev/null || true

# ── 5. Verify it actually runs here ──────────────────────────────────────────
# A model that downloads but will not load is worse than no model: the
# assistant would sit in "starting" forever. Prove it loads before printing a
# configuration that claims it works.

info "Verifying the model loads on this machine..."
PORT="${AI_VERIFY_PORT:-8099}"
"$LLAMA_BIN" --model "$TARGET" --host 127.0.0.1 --port "$PORT" \
    --ctx-size "$MODEL_CTX" --parallel 1 --jinja --reasoning off --reasoning-budget 0 \
    --no-webui --gpu-layers "$GPU_LAYERS" >/tmp/shiftmaster-ai-verify.log 2>&1 &
VERIFY_PID=$!
trap 'kill "$VERIFY_PID" 2>/dev/null || true' EXIT

OK=0
for _ in $(seq 1 180); do
    if curl -sf "http://127.0.0.1:${PORT}/health" >/dev/null 2>&1; then OK=1; break; fi
    kill -0 "$VERIFY_PID" 2>/dev/null || break
    sleep 1
done

if [ "$OK" != "1" ]; then
    warn "The model did not become healthy. Last lines of the runtime log:"
    tail -20 /tmp/shiftmaster-ai-verify.log >&2 || true
    die "verification failed — the assistant would not start with this configuration"
fi

REPLY="$(curl -sf -X POST "http://127.0.0.1:${PORT}/v1/chat/completions" \
    -H 'Content-Type: application/json' \
    -d '{"messages":[{"role":"user","content":"Reply with the single word: ready"}],"max_tokens":8,"temperature":0}' \
    | sed -n 's/.*"content":"\([^"]*\)".*/\1/p' | head -1 || true)"
kill "$VERIFY_PID" 2>/dev/null || true
wait "$VERIFY_PID" 2>/dev/null || true
trap - EXIT

info "Model loaded and answered: ${REPLY:-<empty>}"

# ── 6. Print the configuration ───────────────────────────────────────────────
# Printed, never written: .env holds credentials and belongs to the operator.

say ""
say "============================================="
say "   Add these lines to /opt/shiftmaster/.env  "
say "============================================="
say ""
say "AI_ENABLED=true"
say "AI_MODEL_PATH=$TARGET"
say "AI_MODEL_NAME=$MODEL_NAME"
say "AI_SERVER_BIN=$LLAMA_BIN"
say "AI_CONTEXT_SIZE=$MODEL_CTX"
say "AI_PARALLEL=$PARALLEL"
say "AI_GPU_LAYERS=$GPU_LAYERS"
say "AI_BASE_URL=http://127.0.0.1:8081"
say "AI_RUNTIME_LOG=/var/log/shiftmaster/ai-runtime.log"
say ""
say "Then restart the API:   sudo systemctl restart shiftmaster"
say "Check it came up:       curl -s localhost:8080/api/assistant/status"
say ""
say "The assistant appears on the Home page for every signed-in user as soon"
say "as the model finishes loading; while it loads, the page shows that it is"
say "starting rather than hiding the feature."
say ""
say "---------------------------------------------"
say "AIR-GAPPED INSTALL"
say "---------------------------------------------"
say "On a machine with no internet, copy these two things from a machine that"
say "has them and skip this script:"
say "  1. the llama-server binary  → $BIN_DIR/"
say "  2. the .gguf model file     → $MODEL_DIR/"
say "then set the variables above by hand. Nothing else is needed: the"
say "assistant makes no network calls to answer a question."
say ""
