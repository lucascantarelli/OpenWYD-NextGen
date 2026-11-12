# Evidência — WEB-02: painel do jogador vivo

> **Issue:** `WEB-02` — Painel do usuário (conta, recuperação, rankings, personagens), concluída.
> **Registro canônico:** [`roadmap-rust/changelog.md`](../../../roadmap-rust/changelog.md).
> **Data da coleta:** 2026-11-12.
> **Objetivo:** provar os quatro fluxos no portal e no gateway vivos, com dados reais do banco, `no-store` nas respostas pessoais e nenhum segredo no HTML.

## Stack usada

`docker compose ps` mostrou `wyd-db-1`, `wyd-gateway-1`, `wyd-portal-1` e os dois canais como `healthy`. O portal respondeu em `127.0.0.1:3000` e o gateway em `127.0.0.1:7600`.

```text
GET  /api/health (portal)  -> 200 {"ok":true,"gateway":"up","items":3220}
GET  /login (portal)       -> 200
POST /admin/v1/... (gateway) -> result=1 nos quatro endpoints
```

A sessão usada foi a conta `admin` (id `1`), que tem o personagem real `Cantarelli` no slot `0`. O teste não criou nem alterou contas, personagens ou configurações.

## Prova pelo portal

Executado com Node dentro do container `portal`, usando a credencial de staff já injetada no ambiente (sem imprimir a senha). O cookie de sessão foi mantido em memória.

```text
login: 200 role=admin cookie_httponly=yes
conta: 200 cache=private, no-cache, no-store, max-age=0, must-revalidate conta=ok personagens=ok ranking=ok sem_segredo=ok
recuperacao: 200 cache=no-store anti-enumeracao=ok
ranking: 200 top1=ok
```

Asserções que falhariam se o fluxo estivesse quebrado:

- `/conta` sem cookie ou com cookie inválido não entrega a página pessoal; com o cookie assinado entregou `Olá, admin.` e `sessão #1`.
- O bloco de personagens continha `class="character__slot">Slot 0` e o nome real `Cantarelli`.
- O bloco de ranking e a página pública `/ranking` continham a posição `#1` e personagens reais do topo (`K032900`, `K071900` ou `K106700`).
- A recuperação respondeu `200` com a mensagem uniforme `Se a conta existir...`, sem revelar se o login existe.
- O HTML não continha `$argon2` nem `password_hash`; o cookie de sessão veio com `HttpOnly`.
- `/conta` respondeu com `cache-control: private, no-cache, no-store, max-age=0, must-revalidate`; a recuperação respondeu `cache-control: no-store`.

## Prova direta no gateway

O mesmo processo do portal chamou o gateway com o token de serviço e os endpoints do painel:

```text
gateway: self=admin characters=1 ranking=10 recovery=ok
```

Cada chamada retornou HTTP `200` e `result: 1`:

- `panel/self` → conta `admin`;
- `panel/characters` → uma personagem, `Cantarelli`;
- `panel/rankings` → 10 posições, com `rank: 1` na primeira;
- `accounts/recovery` → `requested: true`.

## Auditoria da recuperação

A solicitação feita pelo portal ficou registrada no `audit_log` do banco, sem gravar o login no alvo:

```text
7551|11:14:03|0:admin|accounts.recovery|login:*|solicitação de recuperação
7547|11:13:54|0:admin|accounts.recovery|login:*|solicitação de recuperação
7541|11:13:34|0:admin|accounts.recovery|login:*|solicitação de recuperação
```

## Conclusão

Os quatro fluxos — conta, recuperação, rankings e personagens — funcionaram no portal vivo e no gateway vivo, com dados reais do banco. As respostas pessoais usam `no-store`, o cookie é `HttpOnly` e nenhum hash/segredo apareceu no HTML. O critério de aceite está cumprido; este arquivo preserva a prova fora de `tmp/` e a entrada canônica está no changelog.
