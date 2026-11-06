# ADR 0003 — Portal como BFF sobre o web-api

**Data:** 2026-10 · **Estado:** aceita

## Contexto

O portal é uma UI de administração: catálogo, contas, preços, faturamento, logs. Ele convive com o
servidor de jogo, que é dono do estado vivo, e com um banco que guarda contas e conteúdo. Dar ao
browser acesso ao gRPC ou ao Postgres significaria espalhar credencial, certificado e autorização por
30 telas.

## Decisão

O portal é **BFF puro**: só lê e envia intenção pelo web-api gRPC. Toda ação admin passa por
`adminCall` (`portal/src/server/admin.ts`), que exige sessão de staff, injeta `moderator_id` **da
sessão**, aplica deadline, mapeia resultado para HTTP e audita. O browser nunca vê gRPC, DSN ou stack.

## Consequências

- Autorização tem um lugar só, e a auditoria é consequência do caminho, não de disciplina.
- A web nunca escreve estado de personagem vivo: grant vai por `delivery_queue` (ADR 0005).
- Uma tela nova custa um RPC tipado, não uma credencial nova.
