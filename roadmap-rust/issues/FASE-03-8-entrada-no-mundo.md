# Issue 03-8 — Entrada no mundo: spawn do personagem e estado vivo persistido

**Fase:** 3 · **Depende de:** 03-7 ✅, 04-3 ✅ · **Bloqueia:** 03-9 · **Esforço:** M
**Status:** ✅ entregue em 2026-11-09

## Objetivo

Transformar `_MSG_CharacterLogin` em **entidade viva no mundo**: ler o personagem do Postgres,
criar a entidade no `ComponentStore` na posição salva e devolver ao banco posição/HP/MP/EXP no
logout, queda de conexão e save periódico.

## Contexto / evidência

Hoje a seleção de personagem **decide no banco e responde**, mas a sessão não tem mundo:
`checks` do checklist 1.11 (“sem cena”) e 3.5 (“spawn no fio ⬜”). O canal autentica, seleciona
e libera a sessão; o personagem nunca vira entidade.

O `wyd-db` já tem o que é preciso: `repo/characters.rs` (leitura + soft delete),
`transfer/save.rs` (fila com coalescência e dreno) e `Grants::claim_gold` como exemplo de
escrita atômica no Cold Path. O `wyd-core` já tem `ComponentStore::spawn`/`SpawnSpec` e
`Stats` validados.

## Passos

1. `world.rs` (03-7): `entrar(character_id) -> Entity` a partir de `CharacterSheet` lido do
   banco **fora do tick**; spawn com `EntityKind::Player`, posição e `Stats` do registro.
2. Recusa tipada e atômica: personagem inexistente, apagado (`deleted_at`), já vivo em outro
   canal (a cerca de sessão já cobre a conta; aqui a checagem é por personagem) ou stats
   inválidos → nenhuma entidade criada.
3. Saída: `sair(entity)` publica `DomainEvent::PlayerLeft`, remove a entidade e enfileira o
   save (posição, HP/MP, EXP, ouro). Save periódico a cada N segundos pelo `transfer/save.rs`.
4. `cpsock.rs`: o laço de seleção, ao receber `CharacterLogin` aceito, chama `entrar` e mantém
   a conexão; queda de TCP dispara `sair` (sem perder o save).
5. Testes de integração com banco real (pulam sem DSN): entrar → posição no mundo == banco;
   sair → posição atualizada no banco; matar a conexão → save executado; personagem apagado →
   recusa sem entidade.

## Arquivos

- Criados: `crates/wyd-server/tests/entrada_mundo.rs`.
- Alterados: `crates/wyd-server/src/world.rs`, `cpsock.rs`, `characters.rs`,
  `crates/wyd-db/src/repo/characters.rs` (save de estado vivo, se faltar coluna),
  `docs/rust/08-checklist-funcionalidades.md` (1.11/3.5), `docs/rust/03-progresso.md`.
- Proibido tocar: fórmulas de dano/EXP (FASE-10), inventário no fio (issue própria),
  `ESCENE_FIELD` (03-9).

## Testes / verificação

```bash
WYD_TEST_DATABASE_URL=... cargo test -p wyd-server --test entrada_mundo
WYD_TEST_DATABASE_URL=... cargo test -p wyd-db --test pg
cargo test --workspace          # sem DSN: os testes de banco pulam com aviso
```

Saída esperada: personagem entra e sai com estado preservado; nenhuma linha duplicada em
`character_session`; save visível em `characters.updated_at`.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core — ECS, spawn/despawn, evento de domínio.
- .agents/skills/wyd-rust-db — save/load fora do tick, transação curta.
- .agents/skills/wyd-rust-testing — integração com banco real, sem cliente gráfico.

## Documentação de referencia

- `docs/wyd/canais-e-cidadania.md` §5.2 (trânsito entre canais é relogar sem perder nada).
- `docs/arquitetura/modelo-banco.md` (personagem/inventário) e `contrato-persistencia.md`.
- `AGENTS.md` §3 invariante 3 (servidor é autoridade).

## Concluído (2026-11-09)

- `world.rs`: `Engine::enter/leave/snapshots`, `LiveState`, `EnterWorldError` e `EngineHandle`
  (fila `std::sync::mpsc` com resposta síncrona de até 2 s). Nenhum mutex protege o `Engine`:
  só a thread `wyd-tick` toca o mundo.
- `characters.rs`: `SelectionStore::with_world(handle, saves)`; `enter_world` valida e encarna,
  `leave_world` desenfileira o estado para o save. Sem mundo (testes/legado), `enter_world`
  devolve `Ok(None)` e nada muda.
- `cpsock.rs`: `CharacterLogin` aceito agora chama `enter_world` **antes** de responder;
  recusa do motor vira `CharacterLoginFail`. `Stage::InGame(Option<Entity>)` carrega a
  entidade e a saída (logout, ociosidade ou queda de TCP) chama `leave_world`.
- `main.rs`: `SaveQueue`/`SaveDrain` (política local: 5 s, 256/lote), dreno de 1 s com flush
  final no shutdown, save periódico de 5 s e sequência final snapshot → dreno → tick.
- `tests/entrada_mundo.rs` (banco real): entra na posição 640,480, recusa duplicado, sai e
  relê `pos_x/pos_y/hp/mp` gravados pelo `SaveDrain`.

**Decisão registrada:** o save do mundo **não** escreve `gold`. Quem escreve ouro é o
`claim_gold` da FASE-07-4; um snapshot do motor sobrescreveria uma concessão concorrente.
EXP/posição/HP/MP convergem pelo save (coalescido por personagem).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] `CharacterLogin` cria entidade viva com dados do banco (não do cliente).
- [x] Saída/queda salva posição, HP/MP e EXP sem duplicar escrita (ouro é do claim de grant).
- [x] Recusa de entrada não cria entidade nem altera estado.
- [x] Teste de integração verde com banco; `cargo test --workspace` verde sem DSN.

## Riscos / rollback

- Risco: save no caminho da queda atrasar o fechamento do socket → mitigação: enfileirar,
  nunca escrever síncrono na thread de rede.
- Risco: personagem “preso” como vivo após crash → mitigação: `released_at`/`renewed_at` já
  existentes (0030) + save de recuperação.
- Rollback: `git checkout -- crates/wyd-server crates/wyd-db`; seleção volta a só responder.
