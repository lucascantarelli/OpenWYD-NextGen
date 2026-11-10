# Issue 03-9 — Cena autoritativa (`ESCENE_FIELD` 30000): o primeiro pixel de mundo

**Fase:** 3 · **Depende de:** 03-8 ⬜, 08-2 ✅ · **Bloqueia:** 4.8/10.4 do checklist · **Esforço:** L

## Objetivo

Emitir o pacote autoritativo de cena com `HEADER.ID = ESCENE_FIELD` (30000): entidades
visíveis, posição, HP/EXP e remoção — o que faz o cliente desenhar o mundo e atualizar as
barras.

## Contexto / evidência

`ESCENE_FIELD = 30_000` existe em `crates/wyd-protocol/src/lib.rs:43` com o teste
`is_authoritative_scene`, mas **nenhum pacote é emitido** — `grep` no workspace encontra
apenas a constante, o reexport em `wyd-core` e comentários. O checklist 4.8/10.4 está ⬜ e o
invariante 4 do `AGENTS.md` (“pacote autoritativo de cena vai com `HEADER.ID = ESCENE_FIELD`,
não com o id da conexão”) ainda não tem implementação.

Consequência: mesmo com tick e spawn (03-7/03-8), o cliente não vê mundo; a barra de EXP/HP
não atualiza — sintoma clássico de ID errado no pacote, exatamente o que o invariante previne.

## Passos

1. Definir o layout de cena em `crates/wyd-protocol/src/payload/scene.rs` com o que está
   **provado**: `HEADER` 12 B + lista de entidades com campos presentes em `STRUCT_MOB`
   (já decodificado em `personagens.rs`). Campo sem prova fica de fora — nunca inventar
   offset (invariante 5).
2. Marcar o layout como `UNVERIFIED` no doc e no código, com **golden byte-exato** montado a
   partir do codec real (`tests/scene.rs`), não de reimplementação.
3. `world.rs`: a cada tick de visibilidade (não a cada tick de simulação), montar delta de
   cena pela `spatial::grid` — entrar/sair/mover — e enfileirar por sessão; `HEADER.ID =
   ESCENE_FIELD` sempre, inclusive para o próprio jogador.
4. Enviar pelo caminho de escrita já existente (buffer + flush no tick, como o legado).
5. Rate-limit de saída: teto de pacotes de cena por segundo por sessão (a grade é O(1), mas
   o cliente não pode ser inundado).
6. Testes: delta de entrada contém o jogador e vizinhos; sair do raio emite remoção; nenhum
   pacote de cena usa o id da conexão; pacote de 100 entidades ≤ `MAX_MESSAGE_SIZE` (8192).
7. Prova com o cliente 7662 quando houver captura (08-6): registrar o que apareceu e o que
   não. Sem isso, a issue fecha pelo lado servidor com `UNVERIFIED` declarado.

## Arquivos

- Criados: `crates/wyd-protocol/src/payload/scene.rs`, `crates/wyd-protocol/tests/scene.rs`,
  `crates/wyd-server/tests/cena.rs` (ou `wyd-core/tests/visibility.rs`).
- Alterados: `crates/wyd-protocol/src/payload/mod.rs`, `crates/wyd-core/src/systems/visibility.rs`
  (hoje só planejado), `crates/wyd-server/src/world.rs`, `docs/arquitetura/protocolo.md`,
  `docs/arquitetura/estrutura-e-padroes.md` §3/§10, `docs/rust/08-checklist-funcionalidades.md`.
- Proibido tocar: `ESCENE_FIELD` como id de conexão, layout de `MSG_Trade`/`MSG_SetHpMp`
  (UNVERIFIED), `client/`.

## Testes / verificação

```bash
cargo test -p wyd-protocol --test scene
cargo test -p wyd-core --test visibility
cargo test -p wyd-server --test cena
cargo test -p wyd-protocol -p wyd-net     # bytes no fio
```

Saída esperada: golden de cena byte-idêntico ao codec; delta correto em entrada/saída de raio;
`HEADER.ID == 30000` em 100% dos pacotes de cena.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net — framing, payload, sem `panic!` em entrada de rede.
- .agents/skills/wyd-rust-core — spatial grid, delta de visibilidade.
- .agents/skills/wyd-client-reverso — confirmar layout com captura quando possível.

## Documentação de referencia

- `AGENTS.md` §3 invariante 4 (`ESCENE_FIELD` 30000).
- `docs/arquitetura/protocolo.md` §4–§5 e `docs/arquitetura/contrato-transporte-cpsock.md` §8.
- `docs/cliente/superficie-conhecida.md` (pipeline CPSock do 7662).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Pacote de cena emitido com `HEADER.ID = ESCENE_FIELD` e golden byte-exato.
- [ ] Entrada/saída de visibilidade refletida em delta (teste com grid real).
- [ ] Nenhum campo de layout inventado: o que não está provado fica fora e documentado.
- [ ] Cliente desenha o jogador e atualiza barra — **ou** evidência do que faltou, com
  `UNVERIFIED` e a pergunta que fecha (nunca “deve funcionar”).

## Riscos / rollback

- Risco: layout de cena errado derrubar o cliente → mitigação: emissão atrás de flag
  (`WYD_SCENE_PACKETS=0`) e teste de tamanho antes de ligar por default.
- Risco: flood de cena em guerra → mitigação: delta por raio + teto por segundo.
- Rollback: desligar a flag; tick/spawn continuam válidos.
