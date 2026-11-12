# Manual do usuário

> **FASE-11-1** (2026-11-07). Público: jogador e operador do dia a dia — quem joga, quem
> sobe a stack local e quem usa o portal. Sem jargão de implementação: para isso existe
> `docs/manual-técnico/`.

## 1. Jogar no servidor local

1. Instale o cliente WYD 7662 em `client/` (binário proprietário, **não** vem no repo).
2. Aponte o `serverlist.bin` para a máquina do servidor (`IP:8281`); o cliente busca o
   status em `http://IP/serv00.htm`.
3. Suba a stack: `make stack-up` (banco + portal). O servidor de jogo entra com as
   entregas da FASE-07 — enquanto isso, o login não completa e o portal responde
   `502` em `/api/health` (comportamento esperado, não é defeito).
4. Abra o jogo; o launcher valida e aplica patch antes de iniciar.

Diagnóstico de "não conecta": ver `docs/cliente/depuracao.md`. Versão de protocolo
enviada pelo cliente 7662 é **12000** — divergência derruba o login com falha visível.

## 2. Launcher (patch e integridade)

```bash
cargo run -p wyd-launcher -- check <dir-do-jogo> <manifesto>
cargo run -p wyd-launcher -- launch <dir-do-jogo> <manifesto> <registro-de-canais> [--dry-run] [-- args]
cargo run -p wyd-launcher -- serverlist <registro-de-canais> <dir-do-jogo> [--dry-run]
cargo run -p wyd-launcher -- update <dir-do-jogo> <pacote> [--dry-run]
```

- `check` compara hash de cada arquivo com o manifesto (nada é alterado).
- `launch` verifica o manifesto, gera e instala o `serverlist.bin` (atômico, com backup) e só
  então abre o jogo. Manifesto inacessível = não escreve nada e não abre o jogo (`exit 2`).
- `serverlist` faz só a parte da lista de servidores/canais — use quando só os canais mudaram.
  A fonte é `tools/client/serverlist-servers.txt` (um canal por linha).
- `update` aplica patch assinado de forma atômica e guarda backup do que substituiu.
- `--dry-run` mostra a decisão sem tocar no disco — use antes de atualizar em máquina
  de jogador.
- Regras de customização sustentável: `docs/cliente/customizacao.md` (ADR-0003).

## 3. Portal de administração

Endereço local: `http://127.0.0.1:3000` (login em `/login`). Telas (rotas reais do
`wyd-portal/src/app/(admin)/`):

| Tela | Rota | Para quê |
|---|---|---|
| Contas | `/contas` | papéis, banir/desbanir, trilha de alteração de papel |
| Personagens | `/contas/[id]` | ficha, itens, gold, nível |
| Guildas | `/guildas` | nome, fama, membros |
| Itens | `/itens`, `/itens/[index]` | catálogo e ajuste de preço/atributos |
| Drops | `/drops`, `/drops/editar`, `/drops/carry` | tabela de drops e capacidade |
| Mobs / NPCs | `/mobs`, `/mobs/[template]`, `/npcs/novo` | templates e criação |
| Mapa / Atlas | `/mapa`, `/atlas`, `/castelo` | mapa, atlas de teleporte, castelo |
| Eventos / Diária | `/eventos`, `/diaria` | agenda e recompensa diária |
| Loja / Faturamento | `/loja`, `/faturamento` | itens de loja e receita |
| Config | `/config` | parâmetros de jogo (com auditoria) |
| Monitoramento | `/monitoramento` | saúde da stack, alertas, presença por canal e volume de log; linha a linha fica no Grafana (`wyd-logs`) |

**O que o portal não faz:** não altera personagem em tempo real. Presente, item ou gold
entram por uma fila auditada que o servidor aplica no próximo login — se o jogador estiver
online, o efeito pode não aparecer na hora.

Sessão expira em 8h; papel promovido só vale no próximo login. Toda ação fica registrada
com autor, alvo e horário.

## 4. Perguntas frequentes

- **"Ganhei item mas não apareceu."** Estava online quando o grant entrou: relogar aplica.
- **"Não consigo logar no portal."** Sessão expirada ou conta sem `role` de staff.
- **"O servidor caiu ao subir."** Ver `make stack-logs`; banco ausente deixa o mundo sem
  NPCs (conteúdo vem do banco, não do código).
- **"Posso editar arquivos do cliente?"** Só pelo manifesto/launcher; editar à mão quebra
  o hash e o patch seguinte falha.
