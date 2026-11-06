# Portal e papéis

O portal é a interface de administração. Ele **não** conversa com o banco nem com o servidor de jogo
diretamente: toda leitura e toda ação passam pelo web-api (gRPC), e o navegador nunca vê credencial,
DSN ou stack.

## Papéis

| Papel | O que pode |
|---|---|
| `player` | nada no painel administrativo; é o papel padrão de conta criada |
| `moderator` | abre a UI de administração e executa ações de moderação |
| `admin` | tudo do `moderator` mais as ações sensíveis (contas, papéis, faturamento) |

O papel guardado no cookie de sessão é **dica de interface**. Quem autoriza de verdade é o web-api,
que relê `account.role` no banco em toda RPC administrativa. Uma tela que esconder um botão não protege
nada — a proteção é a checagem no servidor.

!!! warning "Não existe papel `support`"
    O código conhece exatamente três papéis (`player`, `moderator`, `admin`). Pedido de "acesso de
    suporte" é `moderator` com escopo de telas, não um papel novo — criar um papel sem revisar o
    web-api daria a ele as mesmas permissões de `player`.

## O caminho de toda ação

1. A tela chama uma ação do servidor (`adminCall`).
2. `adminCall` exige sessão de staff — sem sessão devolve **401**, com papel insuficiente **403** — e
   registra a tentativa negada na auditoria.
3. O `moderator_id` vem **da sessão**, nunca do corpo da requisição; ação atribuída a outra pessoa é
   impossível por construção.
4. A RPC tem deadline de 10 s e o resultado vira status HTTP.
5. A resposta sai com `cache-control: no-store` — dado de conta não fica em cache de navegador nem de
   proxy.

## O que o painel não faz

O painel **não escreve estado de personagem vivo**. Entrega de item, moeda ou EXP ao personagem passa
por uma fila (`delivery_queue`) que o servidor de jogo drena. Isso evita a segunda verdade: se o painel
escrevesse direto no save, o próximo save do jogador apagaria a entrega.

## Rotas e proteção

O portal separa superfície pública (login, e agora os guias) de superfície administrativa. As rotas
administrativas estão listadas em [Papéis e rotas](../../analysis/portal-papeis.md) e cobertas por teste:
rota nova fora da lista **reprova o build**.
