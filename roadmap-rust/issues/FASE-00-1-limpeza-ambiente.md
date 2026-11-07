# Issue 00-1 — Limpeza do ambiente Windows (Go → Rust)

**Fase:** 0 · **Depende de:** — · **Bloqueia:** FASE-01 a FASE-15 · **Esforço:** S

## Objetivo

Deixar a máquina sem nenhum resíduo Go (toolchain, caches, extensões, arquivos de
usuário) antes de qualquer execução Rust.

## Contexto / evidência

Ordem do `start-rust.md`: limpeza completa do ambiente é a primeira issue após o
mapeamento. Estado atual: Go 1.27.3 ainda instalado (`go version` responde);
`go.mod` arquivado em `.deprecated-go/go.mod`; `GOPATH`/caches intactos.

## Passos

1. Remover caches: `go clean -cache -modcache -testcache` (último uso do Go).
2. Desinstalar Go pelo Painel de Controle/winget (ação elevada do dono).
3. Remover extensões Go do VS Code e variáveis `GOPATH`/`GOROOT` do ambiente.
4. Confirmar: `where go` não resolve; `cargo --version` responde 1.99.1+.

## Arquivos

- Criados: nenhum (ação de ambiente).
- Alterados: configuração do VS Code (ver 00-2), variáveis de ambiente do usuário.
- Proibido tocar: `.deprecated-go/` (arquivo preservado), `client/`, qualquer código.

## Testes / verificação

```bash
where go || echo GO-REMOVIDO
cargo --version
```

Saída esperada: `GO-REMOVIDO` e `cargo 1.99.1` (ou superior).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - toolchain Rust como referência pós-limpeza.

## Documentação de referencia

- start-rust.md - ordem de limpeza antes da reconfiguração.
- docs/rust/00-visao-estrutura.md - toolchain vigente.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Nenhum `go` no PATH; caches removidos; extensões Go desinstaladas.
- [ ] Evidência (saída dos comandos) anexada.

## Riscos / rollback

- Risco: remover algo usado por outra ferramenta → mitigação: listar antes
  (`go env GOPATH GOROOT`, lista de extensões) e só então remover.
- Rollback: reinstalar Go via winget (não planejado; arquivo está preservado).
