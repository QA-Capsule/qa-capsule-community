# QA Capsule — Script de démarrage
# Mets tes clés ici, puis lance : .\start.ps1

# ── Clé Gemini (obligatoire pour le self-healing AI) ────────────────────────
$env:GEMINI_API_KEY = "AIza_REMPLACE_PAR_TA_VRAIE_CLÉ"

# ── Autres clés optionnelles (décommente si tu les utilises) ─────────────────
# $env:OPENAI_API_KEY    = "sk-..."
# $env:ANTHROPIC_API_KEY = "sk-ant-..."
# $env:QACAPSULE_MCP_TOKEN = "mon-token-mcp-secret"

# ── Lancement ────────────────────────────────────────────────────────────────
Write-Host "▶ GEMINI_API_KEY = $($env:GEMINI_API_KEY.Substring(0, [Math]::Min(8, $env:GEMINI_API_KEY.Length)))..." -ForegroundColor Cyan
Write-Host "▶ Démarrage QA Capsule..." -ForegroundColor Cyan

go run ./cmd/qacapsule/main.go
