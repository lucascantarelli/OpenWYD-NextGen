# REVIEW-02 — Código Go: padrões, docstrings em português e refatorações

**Fase:** revisão · **Depende de:** 58 (concluída) · **Esforço:** L · **Estado:** ⏳ proposta

## Objetivo

Deixar o código Go legível por quem chega hoje: comentário e docstring em **português**, sem
referência a número de issue/roadmap/legado, cobertura de `go doc` nos símbolos exportados, e as
refatorações que o dossiê listou como dívida (wiring dos `main.go`, validação duplicada Go/TS).

## Contexto/evidência (medido em 05/11/2026)

| Achado | Número | Como foi medido |
|---|---:|---|
| Arquivos Go | 569 | `find . -name '*.go' \| wc -l` |
| Arquivos `_test.go` | 262 | idem |
| Comentários com padrão inglês | ~365 | regex `// (The\|This\|Returns\|Sets\|Creates\|Handles\|Loads\|Parses\|Builds\|Reads…) ` |
| Referências a `issue N` / `roadmap/` / `docs/` dentro de `.go` | 123 | `grep -rn "issue [0-9]\|roadmap/\|docs/migration" --include=*.go` |

Exemplos reais que violam a convenção do projeto:

- `internal/droprate/droprate.go:10` — "The value is a DIVISOR: the bigger it is, the rarer…"
- `internal/itemeffects/itemeffects.go:5,12` — comentário em inglês **e** referência a
  `Source/Code/ItemEffect.h` (legado) no doc do pacote.
- `dbserver/internal/grpcsrv/server.go:99` — "The web client logs in with a one-time code…"
- `dbserver/internal/grpcsrv/*_test.go` — comentários citando `migration 0027`, `issue #401`.
- `api/db/v1/content.proto` e os `.pb.go` gerados citam `issue 58, fase 2` e nomes de arquivo de
  conteúdo que não existem mais (`ItemList.csv`). Gerado ou não, é o que o `go doc` mostra.

Já existe `.golangci.yml`, mas o CI roda com `only-new-issues: true` — a dívida antiga nunca é
cobrada. `cmd/*/main.go` concentra wiring (dívida D5 do dossiê); validação de login/motivo/item é
duplicada entre Go e TypeScript (D6).

# Analisar tudo
- Nosso projeto está ficando grande, veja se precisa de organização das pastas, arquivos, está ficando grandes quantidades de arquivos nos roots, isso pode ser organizado ?
- Pesquisar na internet sobre boas praticas em GO, organização de código, design patterns, arquitetura de software e metodologias ágeis, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Aplicar padrões de projeto modernos, boas práticas de código, design patterns, arquitetura de software e metodologias ágeis no projeto, para que possamos ter um código mais limpo, organizado e de fácil manutenção.
- Migrar todo o código legado em C++ para novas tecnologias e frameworks modernos, mantendo a compatibilidade com o cliente 7.64, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Migrar toda a persistência de dados de arquivos de texto, csv, json, xml, etc. para um banco de dados relacional ou não relacional, removendo assim grandes quantidades de arquivos e códigos legados do servidor, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Implementar um portal administrativo para facilitar a gestão do sistema, incluindo recursos de configuração avançada para drops, mobs, quests, eventos, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Aprimorar a interface do usuário para torná-la mais intuitiva e responsiva, incluindo melhorias na interface do cliente 7.64, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Implementar containers de monitoramento de log para todos os serviços, permitindo uma análise detalhada do desempenho e identificação de possíveis problemas em tempo real, para que possamos otimizar o desempenho e garantir maior segurança na plataforma.
- Aplicar SOLID, Design Patterns, Clean Code, Clean Architecture, DDD, TDD, BDD, CI/CD, DevOps e outras boas práticas de desenvolvimento de software no projeto, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Pesquise boas praticas na internet sobre padrões de projeto modernos, boas práticas de código, design patterns, arquitetura de software e metodologias ágeis, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analisar documentação e limpar informações/arquivos que já estão estruturados em uma documentação mais atualizada, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analisar arquivos de texto, csv, json, xml, etc. que já estão estruturados em um banco de dados relacional ou não relacional, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analisar codigos legados que já foram migrados para novas tecnologias e frameworks modernos, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- criar um projeto do mkdocs com toda a nova documentação, incluir dentro da arquitetura do projeto atual, com seu container, e organizar toda a documentação de forma coerente e acessível, incluindo descrições de funcionalidades, implementações, features a serem implementadas, mapeamentos, regras de negócio, etc. para que possamos ter um conhecimento mais profundo sobre o funcionamento do projeto e suas funcionalidades.
- mova todos os documentos para dentro de uma pasta em roadmap, chamada doc_legacy, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Analisar toda a arquitetura atual do projeto, incluindo a estrutura de diretórios, organização de arquivos, dependências, integrações e fluxos de dados, para que possamos ter um conhecimento mais profundo sobre o funcionamento do projeto e suas funcionalidades.
- Propor melhorias na arquitetura do projeto, incluindo a adoção de padrões de projeto modernos, boas práticas de código, design patterns, arquitetura de software e metodologias ágeis, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Implementar as melhorias propostas na arquitetura do projeto, incluindo a reorganização da estrutura de diretórios, otimização de arquivos, atualização de dependências, integração com novas tecnologias e frameworks modernos, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Documentar todas as alterações realizadas na arquitetura do projeto, incluindo a descrição das mudanças, justificativas, impactos e benefícios, para que possamos ter um conhecimento mais profundo sobre o funcionamento do projeto e suas funcionalidades.
- Aplicar boas práticas de arquitetura de software, como modularização, separação de responsabilidades, injeção de dependências, testes automatizados, monitoramento e logging, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analisar se estamos acertando nas escolhars de arquitetura, linguagens, frameworks, tecnologias, metodologias, padrões de projeto, boas práticas de código, design patterns, arquitetura de software e metodologias ágeis, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.


## Passos

1. **Congelar o padrão antes de mexer**: escrever `docs/development-guidelines/Go-comments.md` com a
   regra (docstring em português, início pelo nome do símbolo, sem issue/roadmap/legado, exemplo de
   bom/mau) e apontar para ela no `AGENTS.md` §8.
2. **Varredura mecânica assistida**: por pacote, traduzir comentários e remover referências a
   issue/roadmap/legado; o conteúdo histórico que importa migra para `docs/` (ex.: a decisão do
   `droprate` como divisor vai para `docs/migration/game-rules.md`).
3. **Docstrings faltantes**: ativar no `.golangci.yml` os linters de documentação (`revive` com
   `exported`, `godot`) e corrigir por pacote, começando por `internal/*` e `api/*` (o que é
   consumido por mais de um binário).
4. **Refatorar wiring**: extrair de cada `cmd/*/main.go` uma função `newApp(...) (app, error)` por
   serviço, testável sem `os.Args`; `main` fica com flags → `newApp` → `Run`.
5. **Unificar validação** Go/TS: escolher o lado autoritativo (servidor), expor as regras em um pacote
   `internal/validation` e fazer o portal consumir mensagens do erro em vez de reimplementar limites.
6. Ligar o gate: `golangci-lint run` completo no CI em modo bloqueante **por diretório já limpo**
   (lista explícita), para a dívida não voltar.

## Arquivos

- `docs/development-guidelines/Go-comments.md` (novo)
- `.golangci.yml`
- `internal/**`, `api/**`, `dbserver/**`, `tmserver/**`, `webserver/**`, `binserver/**`
- `cmd/*/main.go` → `newApp` + testes
- `internal/validation/` (novo, se a unificação for por lá)

## Testes/verificação

```bash
go build ./... && go vet ./...
golangci-lint run          # sem only-new-issues no escopo já limpo
go doc ./internal/droprate # amostra: docstring em português, sem issue/legado
grep -rn "issue [0-9]" --include=*.go . | grep -v '\.pb\.go'   # esperado: vazio
```

## Critérios de aceite

- [ ] Zero comentário em inglês nos pacotes do servidor (varredura por regex documentada no PR).
- [ ] Zero `issue N` / `roadmap/` / `docs/` em `.go` escrito à mão (gerados: regenerar sem a citação).
- [ ] Todo símbolo exportado de `internal/` e `api/` com docstring no padrão.
- [ ] `cmd/*/main.go` sem lógica além de flags + chamada de `newApp`.
- [ ] `golangci-lint run` bloqueante nos diretórios limpos.

## Riscos/rollback

Risco: tradução em massa gerar comentário que mente sobre o código. Mitigação: por pacote, com teste
verde antes/depois; comentário que descreve comportamento errado é pior que ausente. Rollback: cada
pacote é um commit isolado.

## Issues que este review abre

- `REFACTOR-01` wiring dos `main.go` (`newApp` por serviço)
- `REFACTOR-02` validação única Go/TS
- `REFACTOR-03` tradução de comentários/docstrings por pacote
- `REFACTOR-04` regenerar `.pb.go` sem citação de issue/arquivo morto
