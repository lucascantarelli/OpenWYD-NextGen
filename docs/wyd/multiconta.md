# Multiconta e múltiplas janelas (multiboxing)

Pesquisa e decisão para o nosso servidor. **Confirmado por fonte** vem com link; o resto é
decisão nossa, marcada como tal.

## O que a fonte confirma

- No WYD Global (Raid Hut), o lançamento permitia **até três contas por computador ao mesmo
  tempo** (`wydglobal.raidhut.com.br/en/wydlaunch`: "each computer can access the game with up
  to three accounts at the same time").
- A segunda senha (numérica) é **por conta**, pedida na tela de personagem antes de escolher o
  personagem, e a troca é feita **no site da conta**, exigindo a senha numérica atual; o
  esquecimento passa por recuperação por e-mail/suporte (FAQ WYD Global, "Recuperação de
  Dados": <https://feedback.kersef.com/en/knowledge-bases/21/articles/64301-recuperacao-de-dados>).
- O cadastro acontece no **primeiro acesso**, na própria tela de personagem (FAQ WYD Global,
  "Guia para novatos": <https://feedback.kersef.com/en/knowledge-bases/21/articles/58185-guia-para-novatos>).
- Comunidade/cliente não oficial trata a senha numérica como parte do login
  (<https://github.com/kevinkouketsu/Wyd2Client>).

## O que NÃO tem fonte (decisão nossa, conservadora)

| Pergunta | Fonte | Decisão |
|---|---|---|
| Quantos dígitos no 7662 | não encontrada | 4 a 6 dígitos (o campo do fio tem 6 bytes) |
| Limite/bloqueio por tentativas | não encontrada | 5 falhas → bloqueio de 10 min, contado por conta |
| Reset de PIN pelo jogador | só via site/suporte | staff pelo portal (auditado), sem auto-reset no jogo |
| PIN por personagem × por conta | fontes dizem "conta" | por conta |

## Múltiplas janelas: o que o servidor precisa entender

1. **Uma sessão viva por conta** já é regra (`0030_sessao_unica`): a mesma conta não entra em
   dois canais. Isso é o que impede "conta duplicada" — e não impede 4–5 contas distintas.
2. **Um jogador, N contas**: o servidor não tem (nem deve ter) noção de "pessoa". Ele vê
   contas. O que liga as janelas é o IP/dispositivo e o comportamento (mesmo horário, mesma
   rota, trocas entre si).
3. **Riscos** de liberar sem controle: farm multi-conta para alimentar uma conta principal,
   comércio interno (RMT), uso de janelas extras como mula/buffer, e bots operando várias
   contas. Nenhum desses é bloqueado por negar multiboxing — negar só pune o jogador comum.
4. **Controles recomendados** (nesta ordem):
   - limite de **conexões simultâneas por IP** (ex.: 6) — barra abuso sem quebrar 4–5 janelas;
   - **sessão única por conta** (já existe) e por personagem (já existe na cerca);
   - **auditoria** de logins por IP/hora por conta, para detectar 20 contas do mesmo IP;
   - **PvP/economia**: restringir troca/entrega entre contas do mesmo IP apenas em eventos
     competitivos, não no jogo todo;
   - nunca usar "mesmo IP" como prova de fraude: família, lan house e CGNAT existem.

## Launcher

Não há trava de instância única no `wyd-launcher`: `launch` só faz `spawn` do `WYD.exe`
(`bootstrap::spawn`), então abrir várias janelas já funciona. O que falta é conveniência
(`launch --instances N`) e um aviso na UI; nada de mutex global.

## Implementado em 2026-11-09

- **Sessão única por conta** (`0030`) + **limite por IP** no `cpsock`: `WYD_MAX_CONNECTIONS_PER_IP`
  (padrão **6**, `0` desliga). A vaga é liberada no `Drop` da conexão, inclusive em pânico.
  Teste: `crates/wyd-server/tests/sistema_limite_ip.rs` (2ª conexão do mesmo IP é fechada; ao sair a
  vaga volta). 4–5 janelas continuam passando; 20 contas do mesmo IP, não.
- **Launcher**: `launch ... --instances N` (1–5) abre N janelas independentes e imprime os PIDs;
  não existe trava de instância única.

### Observabilidade da recusa (2026-11-09)

`/metrics` publica `wyd_ip_limit_refusals_total` (`metrics::ip_limit_refusals`), e cada 25
recusas sai uma linha agregada `result=ip_limit_summary total=… last_ip=…` no log do
`accept`, além do evento individual `result=refused reason=ip_limit ip=… total=…`.
