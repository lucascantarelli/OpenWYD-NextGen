# Launcher próprio — planejamento

> Complementa a [issue 51](../roadmap/issues/issue-51.md) e a análise em
> [`docs/analysis/cliente-launcher.md`](analysis/cliente-launcher.md). Aqui está **o que
> fazer primeiro** e por quê.

## 1. Por que precisamos de um launcher nosso

O `WYDLauncher.exe` que vem na pasta do cliente fala com o servidor de origem: ele baixa
`serverlist.bin`, `GuildInfo`, `Missionitems.bin` e os `.dat`/`.bin` de lá, e **sobrescreve** o
que temos. Já temos a prova disso no repositório: `.remote_serverlist.bin` de 1.245 bytes é uma
página HTML de erro — o updater do cliente original tentou atualizar e gravou lixo no lugar.

Enquanto o launcher for o deles:

- não conseguimos distribuir IP/porta dos **nossos** canais;
- qualquer arquivo que personalizarmos (assets, textos, guild, serverlist) volta ao original;
- não há versão de cliente, nem patch incremental, nem rollback;
- não há login único com o portal, nem telemetria de erro do cliente.

## 2. Escopo mínimo (fase 1) — o que resolve hoje

Um launcher é, no fundo, três coisas: **descobrir o servidor**, **manter os arquivos iguais ao
manifesto** e **abrir o jogo com as configurações certas**.

| Peça | O que faz | Onde mora |
|---|---|---|
| Manifesto | lista de arquivos com tamanho + SHA-256 + versão | portal (BFF) → `GET /api/launcher/manifest` |
| Patch | baixa só o que mudou, aplica atômico (`*.tmp` → rename), valida hash | launcher |
| serverlist | gera o `.bin` do canal escolhido (ver `client-tools/serverlist`) | launcher (Go) |
| Config | resolução, som, idioma, atalhos, "abrir mais de uma janela" | launcher (arquivo local) |
| Login | abre o cliente, ou entrega o token do portal (fase 3) | launcher + portal |

Decisão de stack: **Go + Wails** (ou Fyne). Motivos: já é a linguagem do servidor, gera um `.exe`
único sem runtime, acessa HTTP/disco nativamente e reaproveita o gerador de `serverlist.bin` que
agora vive em `client-tools/serverlist`.

## 3. Ordem de implementação

1. **Gerador de serverlist** — feito (`client-tools/serverlist`): lê
   `Release/Common/serverlist.txt` (grupo, canal, endereço) e escreve o `.bin` de 7.040 bytes com
   a mesma ofuscação do `Basedef.cpp`. É a base do "escolher canal" do launcher.
2. **Manifesto versionado** — `launcher-manifest.json` (schema: `version`, `files[]`), servido
   pelo portal, gerado por um script que varre o `Release/` limpo. O launcher compara hash e só
   baixa o diferencial.
3. **Updater seguro** — download para `tmp`, verificação SHA-256, aplicação por `rename`, e um
   `.bak` da versão anterior (rollback de uma linha). Nunca escrever direto no arquivo em uso.
4. **Configurações** — tela única (vídeo, áudio, idioma, "permitir múltiplas janelas"), gravada em
   `launcher.json` e aplicada ao abrir o `WYD.exe` (as opções que o cliente lê de arquivo/registro
   precisam ser levantadas com o cliente fechado — hoje só conhecemos o `serverlist.bin`).
5. **Escolha de canal** — lê o manifesto de canais (grupo/canal/endereço/status), mostra lotação e
   gera o `.bin` do canal escolhido; o `serv00.htm` no `:80` de cada canal diz se está no ar.
6. **Assinatura e integridade** — assinar o manifesto (Ed25519, chave pública embutida no launcher)
   antes de qualquer uso em produção. Sem isso, um manifesto adulterado vira execução de código.

## 4. Fase 2/3 — o que vem depois

- **Login pelo portal**: o launcher abre a tela de login do portal (SSO), recebe um `play code`
  de uso único (`W2PP_PLAY_CODE_SECRET` já existe no webserver) e injeta no cliente. Elimina
  senha em texto no cliente e casa com a auditoria do portal.
- **Diagnóstico**: upload opcional do último `ErrorLog`/crash, com consentimento, para uma tabela
  do portal. É o que transforma "o cliente caiu" em issue com contexto.
- **Notícias/eventos** e **loja de doações**: o launcher é o lugar natural para o banner e o
  botão, lendo do portal (nunca do banco direto).
- **Anti-cheat/versão**: o launcher é quem sabe qual build do cliente está rodando; ele pode
  recusar abrir com arquivos alterados (hash do `WYD.exe` e das DLLs de proteção).

## 5. Riscos e regras

1. **Nunca** commitar binários do cliente, `serverlist.bin` com IP privado, nem chave de
   assinatura privada (`client/` e `.freebuff/` já estão fora do git).
2. O launcher **não** fala gRPC nem toca o banco: ele é cliente HTTP do portal, como qualquer
   navegador. Segredos e DSN ficam do lado do servidor.
3. Patch por hash exige que o `Release/` versionado seja a fonte; arquivos gerados (mapas
   `dist/`, `world-atlas.json`) entram no manifesto por script, não à mão.
4. Antes de distribuir, testar em uma cópia do cliente com backup do `serverlist.bin` e do
   `WYD.exe` (ver skill `wyd-client-connect`).
