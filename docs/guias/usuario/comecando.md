# Primeiros passos do jogador

## 1. Apontar o cliente para o servidor

O cliente lê a lista de servidores do `serverlist.bin` (ao lado do `WYD.exe`). O arquivo é gerado pela
ferramenta do repositório e **não é versionado** porque carrega IP privado:

```bash
mingw32-make client-serverlist      # Windows sem make: use mingw32-make
```

Ele escreve a partir de `client-tools/serverlist/servers.txt`. Se você editar esse arquivo, gere de
novo — o `.bin` não se atualiza sozinho.

!!! warning "Versão do protocolo"
    O cliente desta build envia **12000**. Servidor com outra versão responde `0x102` (version
    mismatch) e a tela de login nem abre. Isso é configuração, não bug de conta.

## 2. Escolher o canal

A lista do cliente mostra uma linha por servidor configurado, e cada linha tem canais (as "células"
da entrada). Hoje o ambiente local sobe **dois** servidores de jogo:

| Canal | Papel | Para quem |
|---|---|---|
| `NextGen-1` | novato | conta nova; EXP com multiplicador de campo de treino |
| `NextGen-2` | regular | personagem já estabelecido |

O canal novato é **fixo por configuração** (`W2PP_NOVICE_CHANNEL`), não gira por dia.

## 3. Trocar de canal sem sair do jogo

Use o comando de troca com o número do canal desejado:

```
/srv 2
```

O personagem é levado ao outro servidor sem pedir a senha de novo: o servidor emite um token de uso
único (validade de 2 minutos) que o cliente devolve no login. O menu de canais do ESC **lista** os
canais, mas não envia a troca ao servidor — o caminho que funciona é o comando.

!!! note "Se a troca falhar"
    "Senha inválida" ao reconectar quase sempre é o token recusado: os dois servidores e o dbServer
    precisam compartilhar a mesma `W2PP_CROSSOVER_KEY`. O log do dbServer registra a recusa.

## 4. Ler o ganho de EXP

Ganho de EXP aparece **no painel de EXP** (a barra numérica da direita), não como fala do personagem.
Isso vale para kill, recompensa de quest e EXP administrativo: o servidor manda `MsgExpPanel`, não
mensagem de chat — foi um defeito antigo, corrigido em 06/11.

## 5. Onde pedir ajuda

- Bug no jogo: registre com personagem, mapa, horário e o que você fez antes — sem isso não dá para
  reproduzir.
- Conta bloqueada: só a administração desbloqueia; não existe comando de jogador para isso.
