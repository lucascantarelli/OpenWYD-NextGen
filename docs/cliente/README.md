# Dossiê do cliente — `client/`

> **Estado:** inventário e baseline publicados em 2026-11-07 (FASE-08-4).
> Plano de engenharia reversa: [`engenharia-reversa.md`](engenharia-reversa.md).
> O que já se sabe: [`superficie-conhecida.md`](superficie-conhecida.md).
> O cliente **não é versionado** (`.gitignore`): esta pasta documenta um artefato local,
> não o distribui.

## 1. Por que este dossiê existe

`client/WYD.exe` é o **usuário final do `wyd-launcher`** (`crates/wyd-launcher`): o launcher
confere hash, aplica patch e inicia este binário. Sem conhecer o cliente não há como:

- definir o manifesto de integridade do launcher (quais arquivos, quais hashes, qual versão);
- saber o que um patch pode tocar sem quebrar o jogo;
- evoluir UI, eventos e assets (`.wyt`, `.msa`, `.bin`) com previsibilidade;
- fechar a lacuna entre o que o servidor Rust emite e o que o binário 7662 realmente lê.

O cliente é **fechado** (HanbitSoft/JoyImpact — fonte nunca liberada). Todo conhecimento vem do
binário local `client/` + `ClientPatch_v7662` (DLL em `client/`), de capturas de pacote e de
desassemble (`tools/client/`). Este dossiê é a fonte canônica desse conhecimento no repositório.

## 2. Documentos

| Documento | Conteúdo |
|---|---|
| [`inventario.md`](inventario.md) | Binários, SHA-256, arquitetura PE, camadas de proteção, dados e assets |
| [`superficie-conhecida.md`](superficie-conhecida.md) | CPSock, hooks confirmados por endereço, pontos de extensão, formatos, lacunas |
| [`analise-estatica.md`](analise-estatica.md) | Resultado da 08-5: PE, strings, âncoras, patches, limites do método |
| [`formatos.md`](formatos.md) | Estrutura inicial de `.wyt`, `.msa`, `sn.bin`, `serverlist.bin` |
| [`mapa.md`](mapa.md) | Mapa do cliente: documentado×lacuna, relação com o launcher, roadmap e plano de desassemble |
| [`customizacao.md`](customizacao.md) | Estratégia de customização, manifesto do launcher e fluxo de patch |
| [`engenharia-reversa.md`](engenharia-reversa.md) | Plano de desassemble profundo: alvos, ferramentas, marcos, entregáveis, riscos |
| [`depuracao.md`](depuracao.md) | Roteiro de trace dinâmico: breakpoints, fases login→campo, freeze, fixtures |
| [`pacotes.md`](pacotes.md) | Catálogo de pacotes com estado GERADO/PENDENTE por opcode |
| [`../arquitetura/contrato-dispatch-opcode.md`](../arquitetura/contrato-dispatch-opcode.md) | Contrato de dispatch: 63 handlers do C++ × 61 rotas Go × 198 opcodes Rust, o que o cliente 7662 envia/omite e as divergências a decidir |
| [`referencia-wyd-global.md`](referencia-wyd-global.md) | Cliente real do WYD **Global** (terceiro, `wyd_launcher` 2.7.3): inventário medido, arquitetura Electron/Squirrel, domínios de conteúdo que o 7662 não separa e limites de uso — referência, **não** contrato |
| [`gerado/`](gerado/) | Artefatos determinísticos (`pe`, `strings`, `evidencia`, `formatos`, `ancoras`, `goldens`) |

Histórico consolidado neste dossiê: `docs/cliente/customizacao.md` (patches/hooks),
`docs/cliente/mapa.md` (paridade entre builds), `docs/cliente/depuracao.md` (freeze).

## 3. Regras

1. **Nunca versionar** binário, DLL, `.sys`, dump, log ou `serverlist.bin` com IP privado.
2. **Toda afirmação precisa de build + hash + endereço/arquivo.** Sem isso, é **UNVERIFIED**.
3. **UNVERIFIED não é gate** (`AGENTS.md` §3.5): o servidor segue pelo caminho conservador e loga.
4. **O servidor é autoridade.** Nada que o cliente envia é confiável; hook ou offset nunca vira
   regra de jogo no servidor.
5. **Backup antes de escrever** em `client/`; patch só por ferramenta, nunca à mão.
6. **Não redistribuir** binário/assets: o dossiê registra hash e endereço, nunca o conteúdo.
7. Mudança de offset ou de layout exige atualização deste dossiê **na mesma entrega**.

## 4. Estado resumido

| Item | Situação |
|---|---|
| Mapa client↔launcher + visão de pastas do repo | ✅ gate de arquitetura (`mapa.md`, `visao-pastas.md`) |
| Inventário de arquivos e hashes | ✅ publicado (`inventario.md`) |
| CPSock/hooks mapeados por endereço | ✅ documentado (`superficie-conhecida.md`) |
| Cabeçalho PE, imports, âncoras e patches | ✅ FASE-08-5 (`analise-estatica.md`) |
| Estrutura inicial de `.wyt`/`.msa`/`sn.bin` | ✅ parcial (`formatos.md`) — `serverlist.bin` opaco |
| Roteiro de trace + goldens gerados C→S | 🚧 FASE-08-6 (`depuracao.md`, `fixtures/cliente/`) |
| Captura ao vivo e structs S→C | ⬜ FASE-08-6 (exige servidor executável + VM) |
| Camada de customização + manifesto do launcher | ✅ FASE-08-7 (launcher implementado, 19/19 testes) |
