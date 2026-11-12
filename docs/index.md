# WYD2 — Central de documentação

> Fonte única de navegação da pasta `docs/`. Este índice é a porta de entrada do projeto MkDocs (`mkdocs.yml` na raiz). Leia por perfil, não por ordem alfabética.

## Comece por aqui

| Quero… | Leia |
|---|---|
| Jogar no servidor local / usar o launcher | [Manual do usuário](manual-usuario/README.md) · [Depuração do cliente](cliente/depuracao.md) |
| Administrar pelo portal | Portal BFF (`wyd-portal/README.md`, fora do site) · [Monitoramento e runbooks](rust/06-observabilidade-runbooks.md) |
| Entender a arquitetura Hot/Cold | [Visão de pastas](arquitetura/visao-pastas.md) · [Estrutura e padrões (ADR-0002)](arquitetura/estrutura-e-padroes.md) |
| Implementar protocolo/rede | [Protocolo CPSock](arquitetura/protocolo.md) · [Contrato de transporte](arquitetura/contrato-transporte-cpsock.md) · [Pacotes do cliente](cliente/pacotes.md) |
| Mexer no banco | [Modelo de banco](arquitetura/modelo-banco.md) · [Catálogo gerado](arquitetura/banco-catalogo.md) · [Stack Docker](rust/05-stack-docker.md) |
| Balancear o jogo | [Matemática (SIM)](arquitetura/matematica-sim.md) · [Checklist funcional](rust/08-checklist-funcionalidades.md) |
| Operar / auditar | [Progresso vivo](rust/03-progresso.md) · [índice de evidências](rust/evidencias/README.md) · Planejamento vigente (`roadmap-rust/index.md`, fora do site) · [ADRs](arquitetura/adr/README.md) |

## Regras da casa (não pule)

1. **Estrutura antes de código** — árvore normativa em [estrutura-e-padroes](arquitetura/estrutura-e-padroes.md) §3 e §9–§15. Arquivo novo em `crates/` fora da árvore exige ADR ([rito PROC-21](arquitetura/adr/README.md)).
2. **Servidor é autoridade** — cliente nunca dita posição/dano/inventário ([autoridade-servidor](arquitetura/autoridade-servidor.md)).
3. **UNVERIFIED não vira regra** — comportamento não capturado: tolerar, logar, seguir pelo caminho conservador. Nunca inventar layout de pacote.
4. **Ciclo por issue** — uma issue por vez, gate `make check`, docs atualizadas na mesma entrega ([execução](rust/02-execucao.md)).
5. **Classificação de conhecimento** — `CONFIRMADO / VALIDADO / PROVÁVEL / HIPÓTESE / DIVERGENTE / DESCONHECIDO` (padrão `wyd2.md` §3.2). Hipótese nunca vira tipo, gate ou regra.

## Estado do projeto (2026-11)

- **Planejamento vigente:** `roadmap-rust/index.md` (fora do site, 15 fases) — fila de entregas rápidas por tamanho/desbloqueio.
- **Painel vivo:** [03-progresso](rust/03-progresso.md) — status + evidência por issue.
- **O que o jogo precisa ter:** [08-checklist-funcionalidades](rust/08-checklist-funcionalidades.md) — mapa ✅🚧⬜⛔ por sistema.
- **Dossiê do cliente 7662:** [cliente/](cliente/README.md) — inventário, RE, pacotes, depuração.

## Convenções de escrita

- Português, links relativos, tabelas com fonte e confiança por linha.
- Toda afirmação técnica não confirmada leva **hipótese** ou **UNVERIFIED**.
- Doc que descreve comportamento muda **na entrega que muda o comportamento**.
- Duplicidade é defeito: ao encontrar dois docs dizendo o mesmo, fundir e redirecionar (ver issue `PROC-01`, em `roadmap-rust/issues/`).

## Servir localmente

```bash
pip install mkdocs mkdocs-material
mkdocs serve   # http://127.0.0.1:8000
mkdocs build --strict   # falha em link quebrado
```
