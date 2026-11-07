# Issue 08-3 — Distribuição do launcher/patch (updater substituto)

**Fase:** 8 · **Depende de:** 08-7, 08-4 · **Bloqueia:** FASE-07 (distribuição) · **Esforço:** M · **Status:** ✅

## Objetivo

Entregar o canal de **distribuição**: o updater substituto que busca manifesto remoto e
pacotes de patch, verifica assinatura e os entrega ao `patch::apply` da FASE-08-7.

> Reescopo (2026-11-07): verificação de hash, patch atômico e bootstrap foram entregues
> na FASE-08-7 (`manifest/patch/bootstrap/sha256.rs` + CLI `check`/`launch`). Esta issue
> não os reimplementa — constrói a entrega (download, resume, assinatura) sobre eles.

## Contexto / evidência

`pre-release-rust.md` FASE 8 §8.2 previa analisar o updater legado; o endpoint dele
(`settings.txt`) não é mapeado e foi declarado não suportado (ADR-0003). Base do
manifesto: `docs/cliente/inventario.md` §1–§4. Contrato de patch: `docs/cliente/customizacao.md` §3–§4.

## Passos

1. Documentar o protocolo de distribuição (manifesto remoto, download com resume,
   fallback sem rede).
2. Implementar busca + verificação de assinatura do pacote, acoplando o conteúdo a
   `patch::apply` (nada de segundo caminho de escrita).
3. Subcomando `update` na CLI (check → update → verify → launch).
4. Teste: corrupção em trânsito detectada; resume retoma; pacote sem assinatura válida
   é recusado antes de tocar o disco.

## Arquivos (autorizados por esta issue — lista fechada)

- Criar: `crates/wyd-launcher/src/distribuicao.rs` + `mod tests` no mesmo arquivo (padrão do crate).
- Alterar: `crates/wyd-launcher/src/lib.rs` (1 linha: `pub mod distribuicao;`), `crates/wyd-launcher/src/main.rs` (subcomando `update` + uso), `docs/cliente/customizacao.md` (§4, só se o contrato de entrega mudar).
- Proibido: qualquer outro `.rs` novo ou alterado; `client/` sem backup; `.deprecated-go/` (leitura apenas); arquivo novo na raiz do repo.
- Nota: uma primeira versão de `distribuicao.rs` + `update` já existe no working tree (passada interrompida, sem gate). Ela só vale se passar na revisão contra esta lista e no gate cheio; se divergir, reescrever, não remendar.

## Testes / verificação

```bash
cargo test -p wyd-launcher
./target/debug/wyd-launcher update <dir> <manifesto-remoto> --dry-run
```

Saída esperada: `test result: ok` incluindo pacote adulterado recusado e resume;
nenhuma escrita em `client/` fora de tmp com backup.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - build/distribuição do binário.
- .agents/skills/wyd-client-connect - serverlist e diagnóstico.
- .agents/skills/rust-code-quality - testes.

## Documentação de referencia

- docs/cliente/customizacao.md - contrato de patch que a entrega deve respeitar.
- docs/arquitetura/adr/0003-estrategia-patch-cliente.md - aplicação atômica, fail-closed.
- pre-release-rust.md (FASE 8 §8.2).
- .deprecated-go/docs/migration/config-ops.md - distribuição legada.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue (código da passada interrompida aceito após revisão: conforme a lista fechada, sem arquivo extra)
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Pacote íntegro e assinado aplicado via `patch::apply`; adulterado recusado antes do disco.
- [x] Resume retoma download interrompido; `update --dry-run` não escreve nada.
- [x] Manifesto remoto versionado; fallback sem rede documentado (fonte local; HTTPS fora do escopo, declarado em `distribuicao.rs` //!).

## Resultados (2026-11-07)

- `crates/wyd-launcher/src/distribuicao.rs` + `update`/`--dry-run` na CLI; 27/27 testes do crate (7 de distribuição, incluindo vetor RFC 4231 conferido contra OpenSSL e vetor NIST de 2 blocos).
- Gate: `cargo fmt --all` + `cargo clippy --workspace --all-targets -- -D warnings` + `cargo test --workspace` verdes (protocolo 28/28, launcher 27/27).
- Ponta a ponta em pastas temporárias (nunca `client/`): dry-run lista 1 divergência, update aplica + backup, `check` fica verde, payload adulterado recusado com exit 1.
- Correção honesta: o vetor HMAC esperado no teste estava errado de memória; o código estava certo (duas implementações independentes concordaram) — expectativa corrigida, não o código.

## Riscos / rollback

- Risco: update parcial corrompendo cliente → mitigação: reutilizar `patch::apply`
  (atômico + rollback) em vez de novo caminho de escrita.
- Risco: endpoint legado voltar a ser necessário → mitigação: protocolo próprio, sem
  depender de URL de terceiro.
- Rollback: `git checkout -- crates/wyd-launcher/`.
