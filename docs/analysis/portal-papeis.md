# Papéis e rotas do portal

Este é o contrato de autorização do portal, derivado do código — não do que a interface mostra.
Fonte: `portal/src/server/session.ts`, `portal/src/server/admin.ts`, `portal/src/middleware.ts`.

## Papéis

| Papel | Constante | Abre a UI de administração | Executa ação administrativa sensível |
|---|---|---|---|
| `player` | `Role` | não | não |
| `moderator` | `Role` | sim | conforme a RPC |
| `admin` | `Role` | sim | sim |

`isStaff` é verdadeiro para `moderator` e `admin`. **Não existe** `support`, `owner` ou `gm` neste
código: um quarto papel sem revisão do web-api herdaria as permissões de `player` e passaria a ser um
bug de segurança silencioso (a tela abriria, a ação seria negada).

## Como uma ação é autorizada

```
page/action → adminCall (session.admin.ts) → RPC gRPC do web-api → banco
```

1. **Sessão**: sem cookie de sessão válido, `adminCall` responde **401** e não chama o web-api.
2. **Papel**: com sessão mas sem `isStaff`, responde **403** e a tentativa entra na auditoria.
3. **Identidade**: `moderator_id` é lido **da sessão**; qualquer identificação no corpo da requisição
   é ignorada. Ação atribuída a outra pessoa é impossível por construção.
4. **Deadline**: a RPC tem prazo (10 s); crédito de tempo esgotado vira erro de gateway, não erro 500
   genérico.
5. **Resposta**: `toResponse` devolve só `{ error }`, sempre com `cache-control: no-store`. DSN,
   certificado, stack e SQL não atravessam a fronteira.

## Superfície pública

Estas rotas existem **sem** sessão, de propósito:

| Rota | Por quê |
|---|---|
| `POST /api/auth/login` | é o login |
| `POST /api/auth/logout` | encerrar sessão expirada precisa funcionar |
| `GET /api/health` | probe de container |
| `GET /guias/**` | documentação pública (conteúdo de `docs/`) |

Qualquer outra rota de API exige staff. Rota nova que não seja declarada pública **precisa** entrar no
`matcher` do middleware: o teste `portal/src/middleware.test.ts` compara a lista de rotas do disco com
o `matcher` e reprova o build quando as duas divergem.

## Rotas administrativas

O `matcher` do middleware cobre as telas do shell administrativo (grupos Visão, Conteúdo, Mundo,
Operação). A lista exata fica em `portal/src/middleware.ts` e é conferida por teste — por isso este
documento não repete a lista: documento que repete lista de código envelhece no primeiro commit.

Rotas marcadas `soon: true` na navegação (`/refino`, `/skills`, `/operacao`) aparecem como
indisponíveis e **não** têm tela: elas estão protegidas mesmo assim, para o dia em que a tela existir.

## Auditoria

Toda ação administrativa grava: quem (id da sessão), o quê (nome da RPC), alvo e resultado. Tentativa
negada também é registrada — negativa não auditada é negativa invisível. A leitura dessa trilha fica
nas telas de operação do portal, não em log de container.
