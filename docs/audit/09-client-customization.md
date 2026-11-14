# 09 — Customização visual e funcional do cliente

> Base: ADR-0003 (estratégia de patch), `docs/cliente/customizacao.md`,
> `docs/cliente/formatos.md`, `crates/wyd-launcher` e verificação direta de `client/` (2026-11-13).

## 1. O que é possível hoje — por mecanismo

| Alvo | Mecanismo | Ferramenta | Risco | Precisa de patch no `.exe`? |
|---|---|---|---|---|
| Tela de servidores (`.wyt`) | trocar pixels | `tools/client/wyt.mjs --png/--bmp` | baixo (só arte) | não |
| HUD/janelas (`UI/`, `NUI/*.wyt`) | trocar raster | idem | baixo | não |
| Nomes de canal/servidor | `sn.bin` + `serverlist.bin` | `wyd-launcher serverlist` / `serverlist.mjs` | baixo | não |
| Endereço dos canais | célula de 64 B no `serverlist.bin` | idem | médio (IP/porta) | não |
| Resolução/cores | `Config.bin` (modo de vídeo) | edição de 1 byte | **médio** (modo errado = janela estranha) | não |
| Marca do launcher | `Launcher.bmp`, `Launcher.hat` | `wyd-launcher branding` | baixo | não |
| Texturas de malha (`.wys`) | substituir arquivo | precisa de conversor DDS (não validado aqui) | médio | não |
| Malhas (`.msa`/`.msh`) | substituir arquivo | modelagem + conversor inexistente provado | **alto** | não |
| Animações (`.ani`/`.bon`) | substituir arquivo | idem | alto | não |
| Mapa (terreno/objetos) | `Env/FieldXXXX.dat/.trn` | editores de comunidade (licença incerta) | alto | não |
| Layout de UI (posição de botão) | **não está em arquivo** — o `.exe` desenha | — | — | **sim** |
| Novo opcode / nova janela | hook em `ReadMessage`/`SendPacket` | patch de tabela de offsets | alto | **sim** |
| Novo comportamento de jogo | **servidor** (Rust) | — | — | **não** |

**Conclusão estrutural (Confirmado):** o `.wyt` é **raster puro** — 22 B de cabeçalho + pixels +
8 B finais, sem texto, sem controle, sem evento (256/259 amostras fecham a fórmula). Trocar
pixel muda o desenho; **não** move nem cria botão. Qualquer mudança de layout exige patch.

## 2. Camada de patch (ADR-0003)

```
manifesto (build + protocol + sha256 + size por arquivo)
  → check (fail-closed, erro por linha)
  → backup .bak-<build>-<data>
  → tmp no MESMO diretório → verifica hash → rename atômico
  → verify (revalida tudo)
  → launch só com verify verde
```

| Regra | Por quê |
|---|---|
| Patch preso ao hash do build | offset fora do build = crash silencioso |
| Nunca herdar código do `ClientPatch_v7662` | é **GPL-3.0**; distribuir obriga GPL |
| `serverlist.bin`/`sn.bin` fora do manifesto de build | são por ambiente |
| Ed25519 no pacote, SHA-256 por arquivo | integridade + autoria |
| Rollback em qualquer falha | nunca deixar instalação meio-patchada |

Estado: **implementado e testado** (`manifest.rs`, `patch.rs`, `distribuicao.rs`, 140 testes no
crate); spawn real no Windows ainda não exercitado nesta máquina.

## 3. O que dá para mudar **sem** tocar o executável

### 3.1 Identidade visual

| Item | Arquivo | Como |
|---|---|---|
| Logo do launcher | `Launcher.bmp` | gerado do `identidade.tokens` |
| Shell HTML da GUI | `Launcher.hat` | HTML puro gerado |
| Tela de seleção de servidor | `NUI/ServerList.wyt` | raster 287×280 32 bpp |
| Fundo de tela | `UI/PCK_Background.wyt` | 800×600 24 bpp |
| Ícones on/off | `NUI/iconon.wyt` / `iconoff.wyt` | iguais em tamanho hoje (393.246 B) |
| Nome dos canais | `sn.bin` | ≤ 8 caracteres + NUL |

### 3.2 Conteúdo de jogo (servidor, não cliente)

Itens, drops, taxas, quests, NPCs e mapas **não** devem ser “customizados no cliente”: a verdade
é o Postgres. O cliente carrega `ItemList.bin`/`SkillData.bin` localmente, então **há duas
verdades** — e elas precisam casar (`DIV-07`). Ampliar conteúdo exige atualizar **as duas**.

## 4. O que **não** é viável (com a evidência atual)

| Pretensão | Por que não |
|---|---|
| Criar janela nova só com assets | layout/eventos vivem no `.exe` |
| Mudar texto de UI | strings estão no binário, não nos `.wyt` |
| Adicionar ícone de skill novo | `SkillData.bin` (248 registros) + `itemicon.bin` (6.500) têm tamanho fixo; crescer exige mudar o leitor |
| Novo slot de equipamento | `MAX_EQUIP` é constante de build (16 aqui × 18 no OSS v769) |
| Suporte a resolução maior sem `Graphics.ini` | o build atual **não tem** `Graphics.ini`; o Global tem |
| Efeito de partícula novo sem shader | `pseffect*/vseffect*` são bytecode SM 1.1 compilado |

## 5. Pipeline de asset (proposto, não provado ponta a ponta)

### 5.1 UI (`.wyt`)

```
arte (Photoshop/GIMP/Krita) → TGA 24/32 bpp → wyt.mjs (empacotar) → teste visual → manifesto
```

`wyt.mjs` hoje faz **leitura** (`--png`, `--bmp`, `--scan`). Escrever `.wyt` é trivial (cabeçalho
fixo + raster), mas **não** existe comando de escrita — é a primeira ferramenta a criar.

### 5.2 Malha/arma (`.msa`/`.msh`/`.wys`/`.ani`)

```
Blender → export próprio (FVF, palette, influences) → .msh/.msa + .wys (DDS) + .ani → teste no cliente
```

**Bloqueio:** não há conversor provado. O `.msa` tem magic `0x112`, `0x20`, `1` e nome de textura
em `@34`; geometria/esqueleto seguem opacos. Criar o conversor exige Ghidra/IDA no loader.

### 5.3 Mapa (terreno)

`Env/` tem `FieldXXXX.dat` + `.trn` por campo (522 arquivos), `AttributeMap.dat` e
`EnvTextureList*.bin`. Editores de comunidade existem, mas **licença incerta** — não integrar
sem decisão jurídica.

## 6. Limites jurídicos e operacionais

1. **Não redistribuir** binário ou assets do cliente (HanbitSoft/JoyImpact). O launcher distribui
   **patch**, não o jogo.
2. **GPL-3.0** do `ClientPatch_v7662`: leitura de endereços OK; código não.
3. Assets do **WYD Global** são de terceiro: servem de referência de escala/estrutura, **nunca**
   de conteúdo para o nosso seed.
4. Patch só por ferramenta, com backup e verificação; nada de edição manual em `client/`.
5. Toda mudança de offset/layout atualiza `docs/cliente/` na **mesma** entrega.

## 7. Prioridade de customização (não misturar com paridade)

| Ordem | Item | Esforço | Valor |
|---:|---|---|---|
| 1 | Escritor `.wyt` + pipeline de UI | S | alto (identidade visual) |
| 2 | Resolução/modo de vídeo correto (`Config.bin`) | S | **muito alto** (qualidade percebida) |
| 3 | Branding do launcher | feito | — |
| 4 | Conversor de malha/arma | L | alto (conteúdo novo) |
| 5 | Editor de mapa com licença clara | L | médio |
| 6 | Cliente independente | XL | só após paridade |

> Regra do prompt §14.3 respeitada: **primeiro paridade funcional, depois expansão.** Nada desta
> página deve entrar na fila antes de `PAR-CLI-01/02` e `PAR-MOV-01`.
