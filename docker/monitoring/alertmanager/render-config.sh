#!/bin/sh
# Render da configuração do Alertmanager (DEVOPS-02) — FONTE ÚNICA.
#
# Por que existe: o Alertmanager não expande variável de ambiente no config, e o
# projeto não versiona segredo. Este script escolhe o destino real
# (`ALERTMANAGER_DESTINO`) e materializa os segredos vindos do ambiente em
# `$OUT` (fora do git e da imagem).
#
# Modos:
#   render-config.sh            → renderiza e `exec`uta o Alertmanager (boot do compose);
#   render-config.sh --check    → renderiza e valida com `amtool check-config`
#                                 (usado por `make monitoring-rules`, sem subir a stack).
#
# Bootstrap: sem `SLACK_API_URL`, a integração Slack aponta para o receptor local
# (`SLACK_BOOTSTRAP_URL`), para a stack funcionar e o payload de Slack ser verificável.
set -eu

ENCERROU_CONFIG=/etc/alertmanager/alertmanager.yml
OUT="${OUT:-/tmp/wyd-secrets}"
MODO="${1:-}"

# Escapa o que quebra o `sed` (&, |, \) e impede injeção no arquivo.
esc() { printf '%s' "${1:-}" | sed -e 's/[&|\\]/\\&/g'; }

mkdir -p "$OUT"

if [ -n "${SLACK_API_URL:-}" ]; then
  printf '%s' "$SLACK_API_URL" > "$OUT/slack_api_url"
  echo "alertmanager: destino=${ALERTMANAGER_DESTINO:-slack} slack=webhook-do-ambiente"
else
  printf '%s' "${SLACK_BOOTSTRAP_URL:-http://alert-sink:9099/alerts}" > "$OUT/slack_api_url"
  # No modo `--check` não há ambiente de produção: o aviso só faz sentido no boot.
  [ "$MODO" = "--check" ] ||
    echo "alertmanager: AVISO SLACK_API_URL ausente - slack em bootstrap para ${SLACK_BOOTSTRAP_URL:-http://alert-sink:9099/alerts}" >&2
fi

if [ -n "${SMTP_PASSWORD:-}" ]; then
  printf '%s' "$SMTP_PASSWORD" > "$OUT/smtp_password"
fi

sed -e "s|__DESTINO__|$(esc "${ALERTMANAGER_DESTINO:-slack}")|g" \
    -e "s|__SLACK_CHANNEL__|$(esc "${SLACK_CHANNEL:-#wyd-alertas}")|g" \
    -e "s|__SMTP_TO__|$(esc "${SMTP_TO:-alertas@localhost}")|g" \
    -e "s|__SMTP_FROM__|$(esc "${SMTP_FROM:-alertas@localhost}")|g" \
    -e "s|__SMTP_SMARTHOST__|$(esc "${SMTP_SMARTHOST:-127.0.0.1:587}")|g" \
    -e "s|__SMTP_USER__|$(esc "${SMTP_USER:-alertas@localhost}")|g" \
    "$ENCERROU_CONFIG" > "$OUT/alertmanager.yml"

if [ "$MODO" = "--check" ]; then
  echo "config renderizada: destino=${ALERTMANAGER_DESTINO:-slack} slack=$([ -n "${SLACK_API_URL:-}" ] && echo webhook-do-ambiente || echo bootstrap) smtp=$([ -n "${SMTP_PASSWORD:-}" ] && echo senha-do-ambiente || echo sem-senha)"
  exec amtool check-config "$OUT/alertmanager.yml"
fi

# `--web.external-url` é o que faz os links do Slack (Alertmanager/silenciar) apontarem para um
# endereço útil em vez do hostname interno do container (5fa8289e0b74:9093).
exec /bin/alertmanager \
  --config.file="$OUT/alertmanager.yml" \
  --storage.path=/alertmanager \
  --data.retention=120h \
  --web.external-url="${ALERTMANAGER_EXTERNAL_URL:-http://127.0.0.1:9093}"
