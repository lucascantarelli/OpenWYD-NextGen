# Política de segurança

Servidor de jogo WYD2 em Rust, portal administrativo e launcher. O que está aqui vale para este
repositório; o cliente proprietário (`client/`) **não é versionado** e não é distribuído pelo repo.

## Como reportar

**Não abra issue, PR ou discussão pública para vulnerabilidade.**

Use o canal privado do GitHub: **Security → Advisories → Report a vulnerability** em
<https://github.com/lucascantarelli/wyd/security/advisories/new>. O relatório fica visível apenas
para os mantenedores e permite combinar correção e divulgação.

Um bom relatório traz: versão/commit afetado (tag `vX.Y.Z` ou sha), superfície (protocolo, portal,
imagem), passos para reproduzir e o impacto real (leitura de dados, execução, negação de serviço).

## Prazos que tentamos cumprir

| Etapa | Prazo alvo |
|---|---|
| Confirmação de recebimento | 3 dias úteis |
| Triagem e classificação (severidade) | 7 dias úteis |
| Correção de `CRITICAL` em versão suportada | 14 dias úteis |
| Divulgação coordenada (advisory público) | junto com a correção |

## Versões cobertas

Apenas a linha atual em `develop` e a última tag `v0.1.x` recebem correção de segurança. Não há
suporte a versões anteriores por enquanto (projeto em pré-release).

## Escopo

- Handshake CPSock, parsers de pacote e tratamento de entrada da rede (bytes malformados, DoS).
- Autenticação de conta/PIN, sessão única entre canais e escalonamento de privilégio.
- Portal: autenticação de staff, autorização por papel e vazamento de dados entre contas.
- Entrega de grants, auditoria e `delivery_queue` (idempotência e integridade).
- Segredos: `.env`, `SESSION_SECRET`, tokens e dumps de conta nunca devem aparecer em log ou imagem.

## Fora de escopo

- Engenharia social, phishing e ataques físicos.
- Bugs de balanceamento, economia ou gameplay sem impacto de segurança.
- Relatos gerados só por scanner, sem impacto demonstrado.
- Vulnerabilidade em dependência já coberta por advisory público **e** sem caminho de exploração no
  nosso uso (o `cargo audit`/`deny` e o Trivy do CI já cobrem a detecção).

## Como o repositório se protege

`cargo audit` + `cargo deny` (gate semanal), Trivy `CRITICAL` bloqueante nas quatro imagens,
dependabot semanal, `imagens` com labels de proveniência (versão + commit) e segredos fora do
repositório — detalhes em `docs/rust/10-devops-ci.md` §5 e §8.
