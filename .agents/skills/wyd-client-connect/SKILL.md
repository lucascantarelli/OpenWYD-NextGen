---
name: wyd-client-connect
description: Configura o cliente Windows do WYD (pasta `client/`) para conectar no servidor local — gera/instala o serverlist.bin, alinha a versão de protocolo (7640 vs 12000), lida com launcher/updater e proteções, libera as portas 8281/TCP e 80/HTTP e diagnostica login que trava, cai ou mostra "versão incorreta". Use sempre que o cliente não listar o servidor, não passar da tela de login, ou quando for preciso validar o handshake ponta a ponta.
---

# Conectar o cliente modificado ao servidor local

O cliente é uma pasta de binários fechados (`client/`) derivada do WYD Global/RaidHut. Ele
**não é recompilável**: tudo que se ajusta é (a) `serverlist.bin`, (b) arquivos de dados, (c)
patches em runtime. Trate a pasta como artefato: faça backup antes de escrever nela.

## 1. Como o cliente descobre o servidor

Ordem real do handshake:

```
1. WYD.exe lê serverlist.bin  → lista de canais (grupo, índice, IP, porta)
2. HTTP GET http://<IP>:80/serv00.htm  → status do canal (o servidor Go serve em :80)
3. TCP connect <IP>:8281              → CPSock (header de 12 bytes)
4. MSG_AccountLogin (0x020D) com ClientVersion → servidor compara com -client-version
```

Consequências práticas:

- **As duas portas precisam estar acessíveis a partir da máquina do cliente**: `8281/tcp` (jogo) e
  `80/tcp` (status). Firewall do Windows/host, NAT ou VPN quebram uma sem quebrar a outra.
- O status é servido pelo tmServer lendo `Release/Common/serv00.htm`. Se esse arquivo faltar, o
  servidor responde um corpo padrão — o cliente normalmente aceita, mas o canal aparece estranho.
- `serverlist.bin` é **binário ofuscado** pela tabela estática do CPSock. Não edite com editor de
  texto: use a ferramenta oficial.

## 2. Gerar e instalar o `serverlist.bin`

Ferramenta: `serverlist bin/serverlist editor.exe` (GUI Windows). Já existem bins prontos
por IP em `serverlist bin/<IP>/serverlist.bin` — servem de referência de tamanho/estrutura
(~7 KB), mas normalmente apontam para IPs privados que não são os seus.

Procedimento:

1. Descubra o IP do host do servidor que o cliente vai discar (`ipconfig` no host, ou o IP impresso
   por `make run-local`; use o IP da LAN, não `127.0.0.1`, se o cliente estiver em outra máquina).
2. Rode o editor, informe `IP:8281`, gere um canal (grupo 0, índice 0 é o caso comum) e exporte
   `serverlist.bin`.
3. **Backup** do atual: `cp "client/serverlist.bin" "client/serverlist.bin.bak"`.
4. Copie o novo por cima.
5. Abra `WYD.exe` **direto** (não pelo `Updater.exe`/launcher).

Por que não pelo launcher: `config.json` aponta para `https://d16tndoz98nwcc.cloudfront.net/info.json`
e o updater baixa um `serverlist.bin` remoto por cima do seu. Prova disso já está na pasta:
`.remote_serverlist.bin` (1245 bytes) é **uma página HTML de erro**, não um serverlist — resíduo de
uma tentativa de update. Se o launcher rodar, ele desfaz sua configuração.

## 3. Alinhar a versão de protocolo (o erro mais comum)

O login compara `MSG_AccountLogin.ClientVersion` com `-client-version` do tmServer. Divergência →
resposta `0x102` e o cliente volta/fica "conectando". O tmServer loga:

```
recv packet conn=… type=0x020d len=…
account login rejected … got=XXXX want=YYYY
```

Valores conhecidos:

| Build do cliente | ClientVersion enviado |
|------------------|----------------------|
| `WYD.exe` 7662 (Cavaleiros de Kersef, protocolo 7640) | `12000` |
| Protocolo documentado (`Basedef.h`) | `7640` |

O `docker-compose.yaml` já sobe o tmServer com `-client-version 12000`, porque o build usado neste
projeto manda 12000. Se o seu cliente mandar outro valor, **não chute**: leia o `got=` no log e
ajuste a flag (ou `W2PP_CLIENT_VERSION`) para exatamente aquele número. Nunca "resolva" o problema
desligando a checagem de versão no código — isso quebra a paridade e esconde incompatibilidade real
de layout.

## 4. Proteções, DLLs e o que não tocar

A pasta contém camadas de proteção/anti-cheat: `JProtectDll.dll`, `jlock101.sys`/`jlock102.sys`,
`JPGLib.dll`, `version.dll` (proxy DLL), `d3dx9_42.dll` (render). Consequências:

- **Não substitua nem edite DLLs sem pedido explícito.** São a camada que faz o cliente rodar; um
  DLL trocado errado transforma "não loga" em "não abre".
- O servidor Go roda com `-reject-checksum` **desligado** de propósito: o cliente tem a verificação
  de checksum desativada por patch (`Source/Code/ClientPatch_v7662`). Se o cliente original (sem
  patch) for usado, espere divergência de checksum.
- `WYD.log` na pasta é o diário do cliente. Ele mostra inicialização gráfica
  (`VertexShader HW Accel Enabled`, `Init Render Target Texture`) — erro de render target não é
  problema de rede, é GPU/driver/`d3dx9`.

## 5. Antes de culpar o cliente: teste headless

Sempre valide o servidor sem o cliente gráfico primeiro. Se o headless loga e o cliente não, o
problema é cliente/rede; se o headless falha, é servidor.

```bash
# (comandos rodam na raiz do repositório)
W2PP_E2E_ADDR=127.0.0.1:8281 W2PP_E2E_ACCOUNT=test W2PP_E2E_PASSWORD=test123 \
W2PP_E2E_VERSION=12000 go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v
```

Sem Go no host, rode dentro do container (ver skill `wyd-testing-bots`).

## 6. Diagnóstico rápido

| Sintoma | Verificar | Correção |
|---------|-----------|----------|
| Servidor não aparece na lista | `serverlist.bin` foi sobrescrito? `.remote_serverlist.bin` mudou de data? | regenerar e não abrir o launcher |
| Aparece, mas "servidor em manutenção"/sem status | `curl http://<IP>/serv00.htm` do host do cliente | liberar porta 80, conferir IP |
| Trava em "conectando" | `docker compose logs tmserver` mostra `recv packet type=0x020d`? | se não mostra, TCP 8281 bloqueado |
| Cai após o login, com `0x102` no log | `got=` vs `want=` | alinhar `-client-version` |
| Loga, escolhe personagem e o cliente fecha | falta template de classe / `CNFCharacterLogin` sem BaseMob | conferir `Release/DBsrv/run/BaseMob/{TK,FM,BM,HT}` e log `base mob templates loaded` |
| Personagem entra mas sem equip visual | BaseMob não carregado (fallback relacional) | mesmo caminho acima |
| Tela preta / erro de render | `WYD.log`, `d3dx9_42.dll`, aceleração | não é rede; ajustar `Graphics.ini` |
| Dois clientes não se veem | conta/personagem distintos e mesmo canal | usar `test` e `test2`, ambos na mesma porta |

## 7. Regras de higiene

- Nunca versione `serverlist.bin` gerado com IP privado, nem dumps/logs do cliente.
- Nunca copie para o repositório `WYD.exe`, DLLs ou `.sys`.
- Sempre faça backup antes de sobrescrever arquivo do cliente; a pasta não é reconstruível a partir
  do git.
- Se for necessário mexer em `sn.bin`/`sndata.bin` (nome dos canais), lembre que são ofuscados: só
  troque via ferramenta própria e mantenha o backup.
