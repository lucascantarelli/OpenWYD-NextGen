# CANAL-01 — Dois canais simultâneos na lista do cliente, canal novato fixo

**Fase:** 7 · **Depende de:** 04, 05 · **Esforço:** M · **Estado:** ✅ concluída (06/11/2026)

## Objetivo

Os dois servidores de jogo (`tmserver` e `tmserver2`) aparecem **ao mesmo tempo** na lista do cliente,
com nome próprio, e o canal novato é **fixo por configuração** (sem rotação por dia).

## Resultado entregue

| Peça | Onde | Evidência |
|---|---|---|
| Canal novato fixo (`-novice-channel` / `W2PP_NOVICE_CHANNEL`) | `tmserver/internal/handler/newbie_channel.go`, `docker-compose.yaml` | boot: `tmserver` → `novice channel role role=novice server_index=0`; `tmserver2` → `role=regular server_index=1` |
| Rotação por dia removida do Go | `newbie_channel.go` + `newbie_channel_test.go` | `go test -run Novice` PASS |
| Lista com dois grupos e nomes distintos | `client-tools/serverlist/`, `client-tools/servernames/`, `client/sn.bin` (não versionado) | login efetuado nos dois servidores pelo usuário (05/11) |
| Regras do binário documentadas | `docs/migration/canais.md` §8.1–§8.3 | offsets medidos no `WYD.exe` |
| Auditoria do papel de canal | `world_event_audit` ids 7/8 | consulta SQL registrada na sessão de 05/11 |

## O que foi medido (e agora é conhecimento do projeto)

1. **Uma linha por contador não-zero** do `sn.bin`; o grupo de cada linha é `contadores[n-1-linha] - 1`
   (`0x4b241b`/`0x4b2421`/`0x4b242b`), e a ordem na tela é **inversa** à dos contadores.
2. **Os canais de uma entrada são as células do grupo**, numeradas por `dia % células` — descoberto no
   teste com 10 células preenchidas (`NextGen-1` … `NextGen-10`). Célula vazia vira `Inexistente`.
3. **A lista de canais só é desenhada com 2+ contadores não-zero**: com `[2]` a tela mostrou apenas
   `NextGen`, sem canais.
4. Nome de servidor no `sn.bin` tem limite de **8 caracteres** (por isso `NextGen`/`NextGen2`).
5. A tabela `0x277bb10` (janela do ESC) tem **990 bytes** = 10 grupos × 11 células × 9 bytes, é
   **zerada no init** (`0x549406`) e preenchida por índice — buscar o literal falhava por isso.

## Configuração canônica em produção local

```
sn.bin      contadores [3 2 0 …]; nomes [Reserved, NextGen, NextGen2 ×9]
serverlist  grupo 0: célula 0 = URL de status (contiguidade exigida pelo cliente)
            grupo 1: célula 1 = .3 (novato), células 2..10 = .4 (PvP)
            grupo 2: células 1..10 = .4
```

## Limitações registradas

- A janela do ESC lista os canais mas **não envia pacote ao servidor** (medido: nenhum `recv packet`
  durante a tentativa), então ela não troca de servidor sozinha. A troca funciona por `/srv` — ver
  [`CANAL-02`](../test/CANAL-02.md).
- A segunda entrada (`NextGen2`) usa o número do dia no rótulo (`NextGen-5` em 05/11), comportamento do
  caminho degenerado do binário; não é configurável pelo `sn.bin`.

## Critérios de aceite

- [x] Dois canais na lista, cada um conectando no servidor correspondente (login real nos dois).
- [x] Canal novato fixo e independente da data.
- [x] Regra do binário documentada com offsets.
- [x] Teste automatizado do papel de canal (`newbie_channel_test.go`).
- [ ] Rótulo único `NextGen-2` para o segundo servidor — **inalcançável** neste build (o caminho
      `%s-%s` depende de tabela que nenhum módulo preenche).
