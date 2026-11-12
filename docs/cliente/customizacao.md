# Customização do cliente e manifesto do launcher

> Estratégia decidida em ADR-0003. Este documento é a **especificação** que a
> implementação (`crates/wyd-launcher`, FASE-08-7) deve seguir — ainda sem código.
> Build de referência: 7662 (`04c9d2a1…27ddf8`).

## 1. Estratégia (ADR-0003, resumo)

Camada própria de patch por tabela de offsets presa a hash de build, com manifesto de
integridade versionado e aplicação atômica. Sem herdar código do `ClientPatch_v7662`
(GPL-3.0 — usá-lo como *leitura* é permitido, distribuir obriga GPL). Os três sites de
checksum já vêm desativados no disco, então o patch inicial é idempotente e o servidor
segue com checksum não rejeitante.

## 2. Manifesto de integridade

Formato (UTF-8, LF, `#` = comentário, caminhos relativos com `/`, uma entrada por linha,
ordenadas):

```text
# wyd-launcher-manifest v1
build 7662
protocol 12000
<sha256hex> <size-bytes> <caminho-relativo>
```Regras: parse estrito com número da linha no erro; `build` deve bater com o hash do `WYD.exe`; tamanho divergente reprova antes de hashear; desconhecido = reprovar, nunca ignorar; caminhos com espaço não são suportados (nenhum arquivo do cliente os usa). `serverlist.bin` e `sn.bin` ficam **fora** do manifesto de build — são por
ambiente e o launcher os gerencia separadamente.

Exemplo (build 7662, medido em 2026-11-07):

```text
# wyd-launcher-manifest v1
build 7662
protocol 12000
04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8 2347008 WYD.exe
e10c82df050a090f0bc2c351e520e14a4e4ad8e5b5dbefaedbbebb099d67cb3b 2674752 WYDLauncher.exe
96bbd83ec43f439cf2bf8505030e289097cf2dbe91ecc6e933a3f295a921d802 41984 ClientPatch.dll
6668cc00eaa365c6efad16802f149407dce8ad497a8f71416a30bd4bcfb66654 45056 JProtectDll.dll
665440b190aa06ce1acd0a8796c010419769afdefd0e71f57a52a64840b2d90a 57344 JPGLib.dll
fb674ce106e3ee9d50b01c17bda8a884e9ad9eda40c5d4760da8e173d737ff72 280064 XPva03.dll
```

## 3. Fluxo de patch (ordem obrigatória)

```
check(manifesto, dir) → divergências? aborta com relatório por arquivo
backup(dir)           → cópia para <dir>.bak-<build>-<data>
para cada arquivo:    → escreve tmp no MESMO diretório → verifica hash → rename atômico
verify(dir)           → revalida o manifesto inteiro
falha em qualquer ponto → rollback(restaura backup) e erro tipado
```

Nunca escrever direto no destino, nunca `tmp` em outro volume (rename deixaria de ser
atômico), nunca aplicar patch parcial como sucesso.

## 4. Bootstrap

O launcher inicia `WYD.exe` com *workdir* = pasta do jogo, sem passar pelo updater legado
(`settings.txt` → endpoint não suportado).Ordem: `check` verde → `patch` se necessário → `verify` → `spawn`. Falha em qualquer etapa = jogo não abre, com motivo no relatório. A entrega do pacote de patch (download) pertence à distribuição (FASE-07/08-3); sem pacote, o `launch` recusa diretório divergente em vez de adivinhar (fail-closed).

## 5. Licença

`ClientPatch_v7662` é GPL-3.0: serve de referência de endereços, nunca de código-fonte.
Qualquer contribuição que copie lógica dele exige reavaliação desta ADR antes de mergear.

## 6. Contrato da CLI futura (para a implementação)

```
wyd-launcher check <dir-do-jogo> <manifesto>   # exit 0 = íntegro; relatório por arquivo senão
wyd-launcher config <dir> <base> --key-file <arq>  # busca, verifica e instala o launcher.conf assinado
wyd-launcher update <dir-do-jogo> <pacote> [--dry-run]  # confere Ed25519 do pacote e aplica com rollback
wyd-launcher launch <dir-do-jogo> <manifesto>  # check → patch → verify → spawn
wyd-launcher serverlist <registro> <dir>       # serverlist.bin + .txt + sn.bin (atômico, com .bak)
wyd-launcher branding <dir>                    # Launcher.bmp + Launcher.hat da identidade única
wyd-launcher gui <dir>                         # LauncherGUI.html (contrato visual da GUI Tauri)
```

`branding`, `gui`, `serverlist`, `launch` e `retheme` resolvem a identidade efetiva por
`configuracao::resolver`: com `--config <launcher.conf> --key-file <arq>` o caminho é estrito (erro,
não fallback); sem `--config`, leem `<dir>/launcher.conf` e conferem com a chave de `--key-file` ou
`WYD_LAUNCHER_CONFIG_KEY`. Arquivo ausente, chave ausente, assinatura inválida ou valor fora do
contrato caem para a identidade embutida **com o motivo visível na janela** — nunca `panic` nem
tela sem marca. `config` é o comando que baixa, verifica e instala o arquivo.

O `update` é o único comando que não recebe chave: ele confere o pacote com a chave
**pública Ed25519 embutida** (`distribuicao::CHAVE_PUBLICA_HEX`) e o pacote é assinado por
`tools/launcher/sign-package.mjs`, com a privada no CI. A config remota continua HMAC com
`--key-file`/`WYD_LAUNCHER_CONFIG_KEY` — são domínios e chaves diferentes.

`serverlist` e `branding` escrevem só **dados por ambiente** e **skin** — nunca o
`WYD.exe` —, sempre com backup `.bak` e `tmp` + rename no mesmo diretório. Os nomes de
servidor do `sn.bin` e as cores da skin saem de `docs/arquitetura/identidade.tokens`, a mesma
fonte que o portal referencia em `globals.css`; o launcher não tem paleta nem nome próprios e
`crates/wyd-launcher/tests/contrato/identidade.rs` + `wyd-portal/src/identidade.test.ts` quebram se um
lado divergir.

Erros tipados, sem `panic` em input (manifestos corrompidos são o caso normal, não exceção).
SHA-256 em `std` puro + vetores NIST (decisão offline, sem dependência externa).

## 7. Aceite pendente de implementação

- [x] manifesto parseado com erro por linha; `check` reprova arquivo ausente/trocado/corrompido (19 testes + `check` real em `client/`: 6/6 OK)
- [x] patch atômico com rollback comprovado por teste (falha no meio restaura tudo; fonte corrompida não toca em nada)
- [x] pacote assinado com Ed25519 (chave pública embutida, `verify_strict`), assinatura do Node verificada em teste golden e rollback coberto
- [x] `launch` só inicia com `verify` verde; executável ausente = erro tipado, sem spawn (dry-run + erro verificados; spawn real pendente de ambiente com jogo)
- [x] nenhum binário do cliente versionado; licença respeitada

## 8. Identidade única e superfície de entrada (2026-11-09)

A paleta/espaçamento/tipografia e os nomes de canal vivem **só** em
`docs/arquitetura/identidade.tokens` (`chave = valor`, parser estrito). O portal aponta para os
mesmos valores em `globals.css`, o launcher embute o arquivo (`include_str!`) e o gerador Node lê
`servidor.N`; não existe paleta paralela em Rust nem `NextGen` fixo no gerador. Três testes
seguram isso: `tests/contrato/identidade.rs` (tokens × CSS, `sn.bin`/`Launcher.bmp` instalados × gerados),
`identidade.rs` no portal (vitest) e `tests/golden/serverlist_paridade.rs` (bytes Rust × Node).

O que é customizável hoje, por arquivo:

| Arquivo | Origem | Decodificável |
|---|---|---|
| `Launcher.hat` | gerado (`branding::hat`) | sim — HTML puro |
| `Launcher.bmp` | gerado (`branding::bmp`) | sim — BMP 24 bpp 598×369 |
| `sn.bin` | gerado (`snbin::build`) | sim — leiaute medido em `formatos.md` §3 |
| `serverlist.bin` | gerado (`serverlist::generate`) | sim — `formatos.md` §4 |
| `NUI/*.wyt`, `UI/*.wyt` | externos (cliente 7662) | **não** — magic `WT10`, sem decoder |

Sem IP privado, sem segredo e sem binário do cliente no git: a skin e os `.bin` são artefatos de
ambiente, regerados por comando.

## 9. Densidade de pixel: por que a arte parece "ampliada" (medição de 2026-11-09)

A queixa de "imagem grande com zoom" tem causa medida, e **não** é o formato `.wyt`.

O `.wyt` é raster cru, BGRA/BGR sem compressão, com largura e altura em `u16` no cabeçalho
(`docs/cliente/formatos.md` §1). Nada no arquivo amarra o tamanho de exibição: quem decide o
retângulo é o `.exe`.

Medição desta passada, com o `ServerList.wyt` re-tematizado instalado (`client/NUI/ServerList.wyt`):

| Medida | Valor |
|---|---|
| Asset | 287×280 px, 32 bpp, **duas colunas de 140 px** (runs `140@3` e `140@144`) |
| Desenhado na tela | colunas de **183 px** e **180 px** (runs `183@332` e `180@516`, captura 1030×788) |
| Fator | **≈1,29× de ampliação horizontal** (140 → 181) |

Ou seja: cada pixel autorado cobre ~1,7 pixel de tela. O cliente **amplia** a textura para um
retângulo fixo do binário — é exatamente o "pegamos uma imagem e demos zoom".

### O que isso permite

1. **Podemos autorar com mais densidade.** O retângulo de desenho não vem do cabeçalho do `.wyt`
   (ele desenha 287 px de asset em ~364 px de tela), então uma textura 2× (574×560, cada coluna com
   280 px) continua sendo desenhada no mesmo lugar e o cliente a **reduz** — redução preserva
   detalhe, ampliação inventa pixel. É o caminho para arte moderna sem tocar no layout.
2. **Limites reais** (não bloqueiam): D3D9 aceita texturas até 8192 px e NPOT; o custo é memória —
   BGRA sem compressão, então 287×280 = 314 KB, 2× = 1,25 MB, 4× = 5 MB. Contra um teto de VRAM de
   qualquer GPU atual, isso é irrelevante.
3. **O que continua limitado**: o canvas do jogo é fixo em 1024×768 e o texto/HUD é desenhado pelo
   `.exe` com fonte própria — aumentar a textura não aumenta a fonte nem o HUD. Densidade maior
   melhora molduras, fundos, ícones e ilustrações; não corrige a tipografia.
4. **O que não fazer**: pegar o asset de 287 px e "ampliar" no gerador. Isso só interpola (é o que o
   `retheme` faz hoje, de propósito, ao trocar a paleta) e nunca ganha detalhe.

**Teste pendente (1 captura):** instalar um `ServerList.wyt` 2× e conferir que o painel continua no
mesmo retângulo. Hoje não foi possível: o `WYD.exe` em execução (PID 8552) é protegido — `taskkill`,
`Stop-Process` e `WM_CLOSE` são negados — e o cliente só recarrega textura ao iniciar. Quando ele
sair, o roteiro é: gerar 2×, capturar, medir as colunas (esperado: 181 px, mesma posição) e comparar
a nitidez com `tmp/cliente/depois.png`.
