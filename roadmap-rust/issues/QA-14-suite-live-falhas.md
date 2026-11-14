# QA-14 — Suíte *live* do servidor: 3 falhas conhecidas com canal no ar

> **ID canônico:** `QA-14`
> **ID histórico:** `QA-14`
> **Estado 2026-11-13:** 🟡 · ver `roadmap-rust/index.md`.

**Tipo:** Teste · **Esforço:** S · **Depende de:** stack no ar + banco descartável · **Bloqueia:** declarar a suíte de sistema 100% verde em modo *live*

## Objetivo

Fechar as três falhas que aparecem quando os testes de sistema rodam **contra os canais reais**
(`WYD_LIVE_CPSOCK`), hoje mascaradas como SKIP no gate padrão.

## Resultado medido (2026-11-13)

Canal 1 em `127.0.0.1:8281`, canal 2 em `127.0.0.2:8281`, banco descartável com as 50 migrations:

| Binário | Resultado |
|---|---|
| `sistema_cena_armia` | ✅ 1 passed |
| `sistema_gold_inicial` | ✅ 1 passed |
| `sistema_loja_viva` | ✅ 1 passed |
| `sistema_selecao` | ⚠️ 3 passed, 1 failed (`sessao_viva_publica_entidade_no_gauge`) |
| `sistema_campo_vivo` | ❌ 1 failed (`personagem_novo_entra_e_aceita_a_quest_do_campo`) |
| `sistema_sessao_unica` | ❌ 1 failed (`mesma_conta_nao_entra_em_dois_canais_e_libera_ao_sair`) |

## Causa raiz de cada falha

1. **`campo_vivo` — teste anterior à cena por proximidade.**
   `campo_vivo.rs:93` exige uma instância do Primeiro Treinador na cena recebida. O personagem
   nasce em `(2096,2096)` e o treinador está em `(2077,2017)` — 79 tiles de distância, fora do
   raio de 24 (`world::scene_around`, `ENV_SCENE_VIEW_TILES`). O NPC existe no banco
   (`npc_definition` id 6301) e o canal loga `event=npc result=spawn map=0 spawned=6111`.
   **Correção:** andar até o raio antes de ler a cena (ou posicionar o personagem no teste), não
   afrouxar o assert.
2. **`selecao` — `expect` em variável opcional.**
   `selecao.rs:332` faz `std::env::var("WYD_LIVE_ACCOUNT").expect(...)`; sem a variável o teste
   falha em vez de pular como os demais. **Correção:** tratar ausência como SKIP documentado
   (ou exigir a variável no alvo `make` que roda o modo live).
3. **`sessao_unica` — leitura de quadro incompleta.**
   `sessao_unica.rs:94` lê só `HEADER_SIZE` (12 B) e desofusca; a recusa do servidor tem
   `Size=13` (1 B de motivo), então `decode_in_place` devolve
   `LengthMismatch { declared: 13, got: 12 }`. É o mesmo `Size=13` documentado em `BUG-01` e
   ligado a `DIV-01` (o 7662 pode ler só o header). **Correção:** ler `Size` do header antes de
   desofuscar; a decisão de protocolo continua em `BUG-01`, não aqui.

## Reprodução

```bash
export WYD_TEST_REQUIRED=1
export WYD_TEST_DSN="postgres://admin:<senha>@127.0.0.1:5432/wyd_validacao2"
export WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_CPSOCK_2=127.0.0.2:8281
cargo test -p wyd-server --test sistema_campo_vivo --test sistema_cena_armia \
  --test sistema_gold_inicial --test sistema_loja_viva --test sistema_selecao \
  --test sistema_sessao_unica -j 1 --no-fail-fast -- --nocapture --test-threads=1
```

## Critérios de aceite

- [ ] `campo_vivo` vê o treinador movendo o personagem (sem relaxar o assert).
- [ ] `selecao` pula de forma explícita sem `WYD_LIVE_ACCOUNT`/`WYD_LIVE_PASSWORD`.
- [ ] `sessao_unica` lê o quadro pelo `Size` declarado e o veredito de protocolo fica em `BUG-01`.
- [ ] Alvo documentado (`make` ou seção em `docs/rust/`) para rodar o modo live com as variáveis.
