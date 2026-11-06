# CHANNEL-02 — `/srv` compatível com o `Enc` do cliente

**Fase:** canais · **Depende de:** CHANNEL-01 ✅ · **Esforço:** S · **Estado:** ✅ concluída (06/11/2026)

## Sintoma

No cliente gráfico, `/srv 1` respondia “você já está neste canal” (correto quando o personagem
está no canal 1) e `/srv 2` não produzia efeito visível. O servidor emitia o token e fechava a
sessão, e o e2e headless passava — a falha estava no **formato do frame** que o cliente lê.

## Causa

O legado monta `MSG_DBCNFServerChange.Enc` como `"*<NewServerID> <Enc[0..7]> <slot>"`
(`CFileDB.cpp:1996-2011`). O primeiro campo, começando por `*`, é o que o cliente usa para saber
o canal de destino. A implementação Go emitia apenas `~x1:<canal>:<slot>:<exp>:<hmac>`; o
dbServer aceitava o token, mas o cliente não conseguia extrair o destino e o comando parecia
um no-op.

## Correção

- `internal/crossover`: `Build` agora devolve `*<canal> ~x1:<canal>:<slot>:<exp>:<hmac>` (38 bytes
  para um dígito de canal, dentro do `Enc[52]`). `IsToken` reconhece a forma nova e a antiga
  durante o TTL; `Verify` exige que o canal do prefixo `*` seja igual ao alvo assinado, então um
  token do canal 2 não pode ser reapresentado como canal 1.
- `tmserver/internal/handler/login.go` e `dbserver/internal/grpcsrv/server.go` usam
  `crossover.IsToken` em vez de comparar o prefixo interno.
- `TestE2EChannelChangeBack` cobre o caminho inverso: personagem no canal 2, `/srv 1`, token com
  prefixo `*1 `, login e seleção no canal 1.

## Evidência (06/11/2026, stack reconstruída)

```text
--- PASS: TestE2EChannelChange (0.29s)
    channel 1: /srv 2 issued token (38 bytes)
    channel 2: token login OK, selected slot 0
--- PASS: TestE2EChannelChangeBack (0.15s)
    channel 2: /srv 1 issued token (38 bytes)
    channel 1: token login OK, selected slot 0
--- PASS: TestE2EServerChangeSameChannel (3.44s)
    same-channel /srv: no token (last frame type 0x18b)
--- PASS: TestE2ESmokeLogin (0.58s)
PASS
```

`go test ./internal/crossover ./tmserver/internal/handler -run ServerChange ./dbserver/internal/grpcsrv`
exit 0. `docker compose logs --since=10m db | grep -c 42P01` → **0**.

## O que continua sendo limite do cliente

A janela de canais do **ESC** não envia pacote ao servidor (medido em 05/11: nenhum frame em
`tmserver`/`tmserver2`). Não há correção server-side para ela; o caminho suportado é o comando
`/srv <canal>`, agora com o `Enc` no formato legado.

## Critérios de aceite

- [x] `Enc` começa com `*<canal> ` e continua assinado/validado.
- [x] `/srv 2` no canal 1 e `/srv 1` no canal 2 emitem token e fecham a sessão.
- [x] Login com token funciona nos dois destinos, sem redigitar senha.
- [x] Mesmo canal responde aviso, nunca token.
- [x] Teste unitário e e2e cobrem o prefixo de wire (regressão do “nada acontece”).
