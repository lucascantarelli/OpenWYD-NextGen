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
wyd-launcher launch <dir-do-jogo> <manifesto>  # check → patch → verify → spawn
```

Erros tipados, sem `panic` em input (manifestos corrompidos são o caso normal, não exceção).
SHA-256 em `std` puro + vetores NIST (decisão offline, sem dependência externa).

## 7. Aceite pendente de implementação

- [x] manifesto parseado com erro por linha; `check` reprova arquivo ausente/trocado/corrompido (19 testes + `check` real em `client/`: 6/6 OK)
- [x] patch atômico com rollback comprovado por teste (falha no meio restaura tudo; fonte corrompida não toca em nada)
- [x] `launch` só inicia com `verify` verde; executável ausente = erro tipado, sem spawn (dry-run + erro verificados; spawn real pendente de ambiente com jogo)
- [x] nenhum binário do cliente versionado; licença respeitada
