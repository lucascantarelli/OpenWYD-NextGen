# w2pp-OpenWYD
servidor aberto de wyd

[![CI](https://github.com/Jean1dev/w2pp-OpenWYD/actions/workflows/ci.yml/badge.svg)](https://github.com/Jean1dev/w2pp-OpenWYD/actions/workflows/ci.yml)

Reescrita em **Go** (big-bang) do servidor do WYD (With Your Destiny), mirando o **client `WYD.exe`
7662 sem modificação** — este build envia `ClientVersion` **12000** no login (medido em 2026-11-04;
`Basedef.h` documenta 7640). Os fontes legados em C++ ficam em `Source/`, os binários
legados + conteúdo do jogo em `Release/`, e os serviços Go novos em `tmserver/`, `dbserver/`,
`binserver/`, `webserver/`. A reescrita é guiada pela engenharia reversa documentada em
`docs/migration/`. Agradecimentos ao pessoal que me forneceu os fontes.

> Para detalhes de arquitetura, comandos e convenções, veja `CLAUDE.md`.

## Serviços

Só a borda client↔tmServer fala o protocolo legado; os links internos são gRPC (+mTLS):

- **tmServer** (`:8281` jogo + `:80` status) — servidor do jogo; dono de todo o estado do mundo em
  memória, num único goroutine sem locks (espelha o reactor single-thread original).
- **dbServer** (`:7514`) — persistência (PostgreSQL/pgx v5) via gRPC `api/db/v1`.
- **binServer** (`:3000`) — gate de billing via gRPC `api/bin/v1`.
- **webServer** (`:7600`) — web-api (gRPC `api/web/v1`) que a plataforma web (Next.js BFF) chama
  server-side; criar conta, login web e, no plano, cash/ranking/loja-web.
- **portal** (`:3000`) — UI de administração em Next.js, BFF puro sobre o web-api; gestão de
  contas, NPCs, drops, itens, mapa de colisão e atlas. Nunca escreve estado de personagem vivo.

Pacotes compartilhados (store, migrations, domain, secret/argon2id) ficam na raiz `internal/`.

## Como rodar

```bash
make run            # só o tmserver, persistência no-op — bring-up rápido do protocolo
make run-local      # stack completa via docker compose + semeia a conta test/test123
make test           # go test -race -cover ./...
```

`make run-local` imprime o IP:porta para apontar um client Windows real. A conta começa sem
personagens — crie-os no client.

O portal sobe junto com a stack em **http://localhost:3000** (login com conta de papel
moderator/admin). Para desenvolver com hot reload, sem container: `make portal-install` e depois
`make portal-dev`. Planejamento e issues ficam em `roadmap/index.md`; detalhes do portal em
`portal/README.md`.

### Variáveis de ambiente

A configuração de processo vem de `./.env`, carregado pelo `docker compose` — o arquivo **não** é
versionado, e `.env.example` documenta cada chave com o seu default:

```bash
cp .env.example .env                                   # primeira vez
W2PP_RATES_EDITING=true docker compose up -d tmserver   # o shell vence o arquivo
```

Nenhuma chave é obrigatória: as flags `-x` dos binários usam `W2PP_X` como default e o argumento de
CLI sempre vence a variável. Números de gameplay (rate, drop, preço, EXP) **não** vêm por env —
ficam em `Release/` e nos overlays do portal. Catálogo completo: `docs/migration/config-ops.md` §0.

## Cliente do jogo

O servidor Go mira o **`WYD.exe` build 7662 sem modificação**, que usa o protocolo/
`ClientVersion` **12000** (valor medido com o cliente real em 2026-11-04; o `Basedef.h` do legado traz
7640). O repositório não distribui um instalador oficial do cliente. Os links antigos
do servidor de teste e do cliente 7.59 foram removidos porque estão offline; veja o histórico nas
issues [#85](https://github.com/Jean1dev/w2pp-OpenWYD/issues/85) e
[#235](https://github.com/Jean1dev/w2pp-OpenWYD/issues/235).

O download do cliente está disponível em **[wyd-ten.vercel.app](https://wyd-ten.vercel.app/)**.

Use uma cópia do cliente obtida de uma fonte confiável e confira arquivos executáveis com seu
antivírus e, se possível, com um serviço como o VirusTotal. Clientes customizados de outros
servidores podem ter launcher, IP ou protocolo alterados e não são garantidos como compatíveis.

### Conectar o cliente ao servidor local

1. Inicie a stack com `make run-local` e anote o IP e a porta exibidos (por padrão, porta `8281`).
2. Abra `serverlist bin/serverlist editor.exe` e gere um `serverlist.bin` apontando para esse IP e
   porta. Para jogar na mesma máquina use o IP informado pelo script; para outra máquina na LAN,
   use um IP do servidor acessível por ela.
3. Faça backup do `serverlist.bin` existente na pasta do cliente e substitua-o pelo arquivo gerado.
   Os arquivos prontos dentro de `serverlist bin/` contêm IPs privados de exemplo e normalmente
   não servem para o seu ambiente.
4. Execute `WYD.exe`, entre com a conta local `test` / `test123` e crie um personagem.

O cliente consulta o status do servidor na porta HTTP `80` e conecta ao jogo na porta TCP `8281`;
libere ambas no firewall quando o cliente estiver em outra máquina. Se o cliente não listar o
servidor, gere novamente o `serverlist.bin` com o IP impresso por `make run-local` e confirme que as
duas portas estão acessíveis.
