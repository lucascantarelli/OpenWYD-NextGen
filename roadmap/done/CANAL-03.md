# CANAL-03 — ESC "Servidor" precisa pedir a troca ao servidor

**Status:** aberto (cliente) · **Bloqueia:** troca de canal pela janela do ESC
**Relacionado:** CHANNEL-01/02 (protocolo), `docs/migration/canais.md` §8.1.1-8.1.2

## Sintoma medido (06/11/2026)

No cliente real, `ESC → Servidor → NextGen-2` deixa o personagem **sem mundo**: sem skill, sem
teleporte, sem abrir NPC. Fechar e reabrir o cliente resolve.

## O que o servidor vê

Nada. O dispatcher registra **todo** frame recebido, inclusive os que não têm rota
(`tmserver/internal/handler/dispatch.go`, linha do `recv packet ... routed=false`). Durante a
tentativa pelo ESC não aparece linha nenhuma em `tmserver`/`tmserver2`. Ou seja: o menu não manda
pedido — não há o que o servidor responder, e por isso nenhum ajuste de protocolo conserta esse
caminho.

O caminho do comando (`/srv <n>`) é diferente e está correto: emite o token, fecha a sessão e o
cliente reconecta (CANAL-02 + `TestE2EChannelChange`).

## O que precisa ser feito (cliente)

O `ClientPatch_v7662` (`Source/Code/ClientPatch_v7662/`) é a DLL que já hooka o cliente. Falta um
hook na confirmação da janela de canais do ESC que dispare o **mesmo** pedido do comando:

1. ler o índice do canal escolhido na lista do ESC (a tabela de nomes é `0x277bb10`, indexada por
   `grupo * 90 + célula * 9`, `canais.md` §8.1.1);
2. enviar `MSG_MessageWhisper` com `MobName = "srv"` e `String = "<canal 1-based>"` — é o mesmo
   pacote que o comando gera, então o servidor não muda nada;
3. se o índice não estiver disponível naquele ponto, o fallback aceitável é **bloquear** a troca
   nessa janela (mostrar "use /srv <numero>") em vez de derrubar o mundo do jogador.

Pré-requisito de engenharia: identificar o endereço da rotina que trata a confirmação da janela.
Esse endereço **não** está mapeado ainda; sem ele o hook seria chute. Enquanto isso vale o
comportamento atual (só `/srv`).

## Build e entrega do binário (ação do usuário, PowerShell/Developer Prompt)

```powershell
# 1. compilar a DLL (Win32, Release)
msbuild Source\Code\ClientPatch_v7662\ClientPatch_v7662.vcxproj /p:Configuration=Release /p:Platform=Win32

# 2. backup do binário em uso e cópia da DLL nova para a pasta do cliente
Copy-Item client\ClientPatch.dll client\ClientPatch.dll.bak -Force
Copy-Item Source\Code\ClientPatch_v7662\Release\ClientPatch.dll client\ClientPatch.dll -Force
```

`client/` não é versionado (`AGENTS.md` §3.9): a DLL compilada **não** entra no git e a entrega é
manual, com o cliente fechado.

## Critério de pronto

- No cliente real, `ESC → Servidor → NextGen-2` sai do mapa, reconecta sozinho e volta com skill,
  teleporte e NPC funcionando — sem reiniciar o cliente.
- `docker compose logs tmserver` mostra `recv packet ... type=...` do pedido e, em seguida,
  `server change: token issued`.
