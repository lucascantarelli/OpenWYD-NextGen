# 17 — Prioridades e roadmap

> Derivado da matriz (`05`), dos bugs (`15`) e das lacunas (`18`). **Não substitui**
> `roadmap-rust/index.md`: ele organiza *trabalho* em 27 issues; aqui a ordem é por **paridade**.
> Nada desta página foi executado.

## 1. Critérios

| Critério | Peso |
|---|---|
| Bloqueia jogar de ponta a ponta | máximo |
| Destrava outras frentes (dependência) | alto |
| Integridade de dados (duplicação/economia) | alto |
| Esforço (S ≤ 1 dia · M ≤ 1 semana · L ≤ 1 mês · XL > 1 mês) | médio |
| Necessidade de captura/VM | penaliza (não é “ganho rápido”) |
| Necessidade de patch no cliente | penaliza |

## 2. Fase 0 — destravar o conhecimento (não é código)

| # | Entrega | Dep | Cx | Risco | Aceite | Teste |
|---:|---|---|---|---|---|---|
| 0.1 | Captura S→C do 7662 (`CLIENT-06`) | VM + cliente | L | alto | ≥ 1 frame de `CNFCharacterLogin`, `SELCHAR`, `ESCENE_FIELD` documentado com hash | `captura.mjs --check` verde |
| 0.2 | Decidir viabilidade de Frida/XTrap | ProcExp/ProcMon | S | médio | relatório “carrega/não carrega” | evidência em `docs/cliente/` |
| 0.3 | Dissector CPSock no Wireshark | 0.1 | M | baixo | pacote legível na GUI | captura anexada |
| 0.4 | Corrigir direção de `CreateItem` no catálogo | — | S | baixo | tabela + golden atualizados | teste de tabela |

## 3. Fase 1 — loop jogável fiel (P1)

| # | Entrega | Dep | Cx | Risco | Aceite | Teste |
|---:|---|---|---|---|---|---|
| 1.1 | Erro de senha visível via `0x0101` | 0.1 | S | médio | senha errada mostra texto no cliente | sistema + prova viva |
| 1.2 | `STRUCT_SELCHAR` com offsets reais | 0.1 | M | alto | roster renderiza nome/nível/classe | golden de frame |
| 1.3 | `STRUCT_MOB` S→C confirmado | 0.1 | M | alto | personagem entra e aparece | sistema + visual |
| 1.4 | Colisão de jogador | `AttributeMap` | M | alto | não atravessa parede | integração + fuzz de posição |
| 1.5 | Movimento no fio | 0.1 | M | alto | outro cliente vê o movimento | sistema 2 clientes |
| 1.6 | Regen de HP/MP | — | S | baixo | HP sobe com o tempo | unitário + tick |
| 1.7 | Respawn de NPC | — | S | baixo | mob volta sem recarga | integração |
| 1.8 | Cadência de ataque provada | 0.1 | S | médio | ≥ 800 ms entre golpes | teste de tempo |

## 4. Fase 2 — itens, economia e comércio (P2)

| # | Entrega | Dep | Cx | Risco | Aceite | Teste |
|---:|---|---|---|---|---|---|
| 2.1 | Item no chão + pickup | 0.1 | M | alto | drop aparece e é coletado | sistema + captura |
| 2.2 | Mover/equipar no fio | 0.1 | M | alto | equipar muda status e persiste | sistema |
| 2.3 | Loja de NPC completa | 2.2 | M | médio | comprar/vender com saldo | sistema + E2E portal |
| 2.4 | Trade atômica | 0.1 | L | **muito alto** | troca não duplica item | teste de concorrência + caos |
| 2.5 | Baú/armazém | 2.2 | M | médio | depositar/retirar persiste | sistema |
| 2.6 | Skill no fio | 0.1 | M | alto | skill causa dano/custo | sistema |
| 2.7 | Mix/compose no fio | 0.1 | M | médio | receita consome e gera | sistema |
| 2.8 | Cadeia +12…+15 / Anct | 2.8→ | L | alto | taxas do legado aplicadas | golden de fórmulas |

## 5. Fase 3 — social e progressão (P3)

| # | Entrega | Dep | Cx | Risco | Aceite | Teste |
|---:|---|---|---|---|---|---|
| 3.1 | Chat + whisper + filtro | — | M | médio | mensagem chega a quem deve | sistema 2 clientes |
| 3.2 | Party + EXP compartilhada | 3.1 | L | alto | EXP dividida conforme legado | integração |
| 3.3 | Guilda (criar/convidar/cargo) | 3.1 | L | alto | roster e permissões | sistema |
| 3.4 | Distribuição de pontos (`ApplyBonus`) | — | S | médio | ponto aplicado e persistido | sistema |
| 3.5 | Teleporte/portal | 1.5 | M | médio | troca de mapa sincronizada | sistema |
| 3.6 | Quest genérica (motor) | 3.5 | L | alto | aceitar/progredir/entregar | integração |
| 3.7 | Eventos/calendário | 3.6 | L | médio | evento liga/desliga por horário | sistema com clock duplo |
| 3.8 | PK/chaos | 1.8 | L | alto | penalidade e cor de nome | integração |

## 6. Fase 4 — expansão (só depois de P1–P3)

| # | Entrega | Dep | Cx | Risco | Aceite |
|---:|---|---|---|---|---|
| 4.1 | Escritor `.wyt` + pipeline de UI | — | S | baixo | tema novo aplicado sem patch |
| 4.2 | Modo de vídeo/resolução correta | — | S | baixo | 1024×768 sem distorção |
| 4.3 | Conversor de malha/arma | 0.3 | XL | alto | arma nova renderiza |
| 4.4 | Editor de mapa (licença clara) | 4.3 | XL | alto | campo novo jogável |
| 4.5 | Cliente independente | 4.3 | XL | muito alto | decisão após paridade |

## 7. Separação exigida pelo prompt (§14.3)

| Categoria | Onde está |
|---|---|
| **Correção de bugs** | §3 (1.1–1.3), §4 (2.1), `15` |
| **Funcionalidade ausente** | §5 inteira |
| **Adequação de protocolo** | Fase 0 + 1.2/1.3/1.5/2.1/2.6/2.7 |
| **Melhoria arquitetural** | `14` §9 (não antes de P1) |
| **Otimização** | `QA-10/11` (carga), não é prioridade agora |
| **Customização do cliente** | Fase 4.1/4.2 |
| **Expansão de conteúdo** | Fase 4.3/4.4 |

**Regra:** não misturar paridade oficial com customização. Nenhum item da Fase 4 entra na fila
antes de P1 fechar.

## 8. O que **não** fazer agora

1. Guerra/castelo/reinos (bloco XL sem dependência pronta).
2. Arena/Battle Royale.
3. Mercado/leilão (não existe no 7662).
4. Refactor de fronteira `wyd-net` × `cpsock` antes da captura.
5. “Endurecer” checksum.
6. Trocar de cliente (7.69) para “ficar mais bonito” — é re-RE completo.

## 9. Ordem topológica sugerida

```
0.1 → 0.2 → 0.3 → (1.1, 1.2, 1.3) → 1.4 → 1.5 → 1.6/1.7 → 2.2 → 2.1 → 2.3 → 2.4
                                              ↘ 2.6 → 3.1 → 3.2/3.3
1.8 → 3.8
3.4 → (paralelo, independente)
4.1/4.2 → contínuo, sem bloquear
```

**Caminho crítico:** `0.1 → 1.3 → 1.5 → 2.2 → 2.4` (captura → entrada → movimento → inventário
no fio → trade). É onde o projeto “vira jogo”.
