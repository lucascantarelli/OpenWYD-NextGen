# 19 — Missão extra: por que o cliente “parecia mais bonito”?

> **Pergunta do usuário:** o cliente do WYD2 da Ongame (versão anterior) parecia muito mais bem
> acabado — texturas melhores, partículas melhores — e o cliente atual parece mais antigo e mais
> simples. A Ongame fez downgrade? Outra equipe fez o cliente antigo? É possível investigar,
> comparar e trazer um cliente oficial melhor?
>
> **Método:** inventário medido de `client/` (2026-11-13), `docs/cliente/*`, medição do cliente
> 7.69 do Global (`docs/cliente/referencia-wyd-global.md`), guia oficial da Ongame arquivado
> (Wayback) e pesquisa externa de 2026-11-13 (`13-external-references.md`).
> **Conclusão antecipada:** a premissa está **parcialmente errada** e a percepção tem causas
> **testáveis** — nenhuma delas é “o cliente virou 2D”.

## 1. Resposta curta

| Pergunta | Resposta |
|---|---|
| O cliente atual é uma versão 2D mais simples? | **Não.** É Direct3D9 com malha *skinned*, ossos, 776 animações e 18 shaders. |
| A Ongame fez downgrade? | **Não há evidência.** O build em `client/` é de **2015-07-02**, ou seja, *posterior* à era Ongame clássica (2005–2016). |
| Então por que parece pior? | **(a)** resolução 800×600 esticada; **(b)** falha de *render target* (log local) que degrada efeitos; **(c)** ~1/3 do volume de assets do cliente 7.69; **(d)** repack sem `Graphics.ini` (sem os ajustes de textura/efeito que o guia da Ongame listava); **(e)** possível memória de **outra** geração/servidor. |
| Dá para melhorar? | **Sim**, sem trocar de cliente: modo de vídeo, driver/ambiente, pipeline de asset e UI. |
| Vale trazer um cliente oficial “mais bonito”? | O único candidato oficial é o **7.69+ do WYD Global** — mas é **outro dialeto** de protocolo/structs: seria re-RE completo, não upgrade. |

## 2. O que a premissa acerta e o que ela erra

| Afirmação | Veredito | Evidência |
|---|---|---|
| “O cliente anterior era mais bonito” | **Não verificável** (memória/ambiente) | Nenhum asset da era Ongame foi medido; só o guia de opções gráficas |
| “O atual é uma versão mais antiga” | **Divergente** | `TimeDateStamp` **2015-07-02**; o guia Ongame é de 2010–2012 |
| “Texturas mais simples” | **Fortemente indicado como percepção** | UI raster fixa + 800×600 esticado em monitor moderno; `Config.bin` modo 7 |
| “Efeitos de partículas mais simples” | **Fortemente indicado** | `WYD.log`: `Error in Init Render Target Texture` → pós-processamento indisponível |
| “Menos recursos visuais” | **Parcialmente confirmado** | 468 efeitos/4,6 MB aqui × 582/18,3 MB no 7.69; sem `ItemSet/MountData/CollectionData` |
| “Outra equipe fez o cliente antigo” | **Sem evidência** | Linhagem única JoyImpact/HanbitSoft; builds 6.13→7.69 |
| “Houve downgrade pela Ongame” | **Sem evidência** | O build atual é mais novo que a era Ongame; e é um **repack de comunidade**, não retail |

## 3. Linhagem de versões (o que se sabe)

| Época | Build/versão | Evidência | Render |
|---|---|---|---|
| 2003 | lançamento (KR) | Wikipedia | D3D |
| 2005–2016 | **Ongame Brasil** (6.x–7.x “Episode I”) | guias arquivados; encerrado 25/07/2016 | D3D9, opções de textura/efeito/brilho |
| 2015-07 | **este build 7662** | `TimeDateStamp`, hash, `ClientPatch_v7662` | D3D9 + SM1.1 |
| 2017+ | 7.69+ (Global/KR) | pesquisa (repo `TMProject2GlobalClient`) | D3D9, mais assets |
| 2019 | RaidHut assume o Global | `raidhut.biz` | — |
| 2026 | “renderer DX11” | **servidor privado** WYD International | DX11 (não oficial) |

**Não confirmado:** qual build exato a Ongame usava em 2016; existência de “7663” (zero evidência).

## 4. Matriz de recursos visuais por geração

> `✅` presente · `❌` ausente · `?` não verificado · `—` não se aplica.
> “Ongame” = era 2005–2016 (evidência documental, sem medição de asset).

| ID | Recurso | Ongame (era) | 7662 (atual) | 7.69 Global | Presente em ambos os clientes medidos? | Ausente em ambos? |
|---|---|---|---|---|---|---|
| VIS-01 | Render Direct3D9 | ✅ | ✅ | ✅ | **sim** | — |
| VIS-02 | Câmera fixa isométrica com zoom/rotação | ✅ | ✅ | ✅ | **sim** | — |
| VIS-03 | Malha com skin + ossos | ? | ✅ (903 `.msh`, 81 `.bon`) | ✅ | **sim** | — |
| VIS-04 | Animações por osso | ? | ✅ (776 `.ani`) | ✅ | **sim** | — |
| VIS-05 | UI raster própria (`.wyt`) | ✅ | ✅ (342 + 8) | ✅ (396 + 20) | **sim** | — |
| VIS-06 | Moldura/texturas de UI estáticas | ✅ | ✅ | ✅ | **sim** | — |
| VIS-07 | Partículas (`.msa`+`.wys` em `Effect/`) | ? | ✅ (468) | ✅ (582) | **sim** | — |
| VIS-08 | Shaders compilados (SM 1.1) | ? | ✅ (18) | ✅ (18) | **sim** | — |
| VIS-09 | Terreno com altura | ✅ | ✅ (por campo, `Env/`) | ✅ (heightmap 4096²) | **sim** (modelos diferentes) | — |
| VIS-10 | Céu/neblina | ? | ? | ✅ (módulos de cena) | provável | — |
| VIS-11 | Água (mar/rio) | ? | ? | ✅ (`TMSea`) | provável | — |
| VIS-12 | Chuva/folhas | ? | ? | ✅ (`TMRain`, `TMLeaf`) | provável | — |
| VIS-13 | Sombra projetada | ? | ? | ✅ (`TMShade`) | provável | — |
| VIS-14 | Iluminação por vértice | ? | ? | ✅ (`TMLight`) | provável | — |
| VIS-15 | Opção textura comprimida/descomprimida | **✅ (guia)** | **❌** (sem `Graphics.ini`) | ✅ (`config_settings.json`) | não | — |
| VIS-16 | Opção efeito mínimo/máximo | **✅ (guia)** | **❌** | ✅ | não | — |
| VIS-17 | Ajuste de brilho | **✅ (guia)** | **❌** | ✅ | não | — |
| VIS-18 | Sets/montarias/coleções como dado visual | ? | ❌ (sem `.bin` próprio) | ✅ (`ItemSet`, `MountData`, `CollectionData`) | não | — |
| VIS-19 | Novas telas (`NUI`, `ccmode`) | ❌ (provável) | ✅ | ✅ | — | — |
| VIS-20 | Configurador próprio (`WydConfig`) | ❌ | ❌ | ✅ (`WydConfig.exe/.swf`) | não | — |
| VIS-21 | Localização em `Lang/` | ❌ (PT fixo) | ❌ | ✅ (73 arquivos) | **sim (ausente nos dois)** | — |
| VIS-22 | Sombras dinâmicas por sol | ❌ | ❌ | ❌ (só no privado) | **não** | **sim** |
| VIS-23 | Água com reflexo/espuma | ❌ | ❌ | ❌ | **não** | **sim** |
| VIS-24 | Renderer DX11 | ❌ | ❌ | ❌ | **não** | **sim** |
| VIS-25 | PBR / HDR / AO | ❌ | ❌ | ❌ | **não** | **sim** |
| VIS-26 | UI em alta resolução / escala | ❌ | ❌ | ❌ | **não** | **sim** |
| VIS-27 | Cliente 64-bit / widescreen nativo | ❌ | ❌ | ❌ | **não** | **sim** |

**Respostas diretas:**

1. **O que só a Ongame tinha:** os **controles de qualidade** (VIS-15/16/17) e, provavelmente,
   uma resolução nativa que não precisava ser esticada. Não há evidência de “arte melhor”.
2. **O que só o atual tem:** telas novas (`NUI`, `ccmode`), shaders compilados no pacote e base
   de assets maior em **malha** (189 MB) — mas menor em **efeito/UI** que o 7.69.
3. **O que ambos têm:** o núcleo visual (D3D9, skin, partículas, UI raster, terreno).
4. **O que falta em ambos:** tudo que é moderno (DX11, PBR, sombra dinâmica, água com espuma,
   UI escalável, 64-bit).

## 5. Por que “parece pior” — cinco hipóteses ranqueadas

| # | Hipótese | Confiança | Como falsificar |
|---:|---|---|---|
| H1 | **800×600 esticado** em monitor moderno borra textura e UI (era nativo em 2006) | **Confirmado como mecânica** | trocar modo para 1024×768 e comparar |
| H2 | **Falha de *render target*** desliga pós-processamento/efeitos | **Confirmado o sintoma** | testar outra GPU/Windows; ver se o log muda |
| H3 | **Volume de assets** ~1/3 do 7.69 → menos variedade por cena | **Fortemente indicado** | contar efeitos usados por mapa nos dois |
| H4 | **Repack sem `Graphics.ini`** → textura/efeito no padrão mínimo | **Fortemente indicado** | inspecionar bytes do `Config.bin` e comparar |
| H5 | A memória é de **outra geração** (7.5x com assets maiores, ou servidor privado DX11 de 2026) | **Hipótese** | comparar screenshots datados do usuário |

### 5.1 Evidência local que sustenta H1/H2

```
client/WYD.log:
  11/12 16:48:00 - VertexShader HW Accel Enabled
  11/12 16:48:00 - Error in Init Render Target Texture

client/Config.bin (30 B):  e1 1d | 07 00 | 02 00 14 00 14 00 ff ff 31 00 02 00 01 00 01 00 01 00 …
                                  ^ modo 7 = 800x600x32
```

- `Config.bin` está em **modo 7 (800×600)**; a documentação do projeto já registra a troca de
  modo 8 (1024×768) para 7 em uma sessão anterior.
- `client/` **não tem** `Graphics.ini` nem `config_settings.json` (o Global tem os dois).
- Os bytes após o modo (`02 00 14 00 14 00 ff ff 31 00 02 00 01 00 01 00 01 00`) são
  **candidatos** a textura/efeito/brilho — **Hipótese**, não verificado.

## 6. Protocolo de teste A/B (30–60 min, sem código)

> Faça **backup de `client/`** antes. Uma variável por vez; registre screenshot no mesmo mapa,
> mesma posição, mesmo NPC/hora.

| Passo | Ação | O que mede | Hipótese testada |
|---:|---|---|---|
| 1 | Baseline: abrir em modo atual e capturar tela (+ `WYD.log`) | referência | — |
| 2 | Trocar `Config.bin` byte 2 de `07` para `08` (1024×768) | nitidez/UI | H1 |
| 3 | Rodar em outra GPU/Windows (ou atualizar driver) e ver o log | render target | H2 |
| 4 | Comparar `Effect/` usados por um mesmo mapa/skill nos dois clientes | variedade de efeito | H3 |
| 5 | Variar **um** byte candidato (12–20) por vez, com backup | textura/efeito | H4 |
| 6 | Comparar com screenshot do Global (mesma cidade) — só referência | delta visual | H5 |

**Critério de decisão:** se H1+H2 explicarem a diferença, **não há downgrade** — há configuração
e ambiente. Se H3 se confirmar como dominante, o caminho é **pipeline de asset**, não troca de
cliente.

## 7. Clientes alternativos — o que dá para usar

| Cliente | Oficial? | Ganho visual | Compatível com nosso servidor? | Jurídico | Veredito |
|---|---|---|---|---|---|
| **WYD Global 7.69+** | ✅ (RaidHut/JoyImpact) | +assets (558 MB mesh), +UI, +efeitos, sets/mounts | **Não** — `WYD.exe` 6,49 MB, tabelas 164/104 B, structs diferentes | EULA; **não redistribuir** | **benchmark visual** e fonte de *estrutura*; trocar exigiria re-RE completo |
| **WYD Origem / Asgard** | ✅ (licenciado BR) | interface legada (possivelmente o “look” lembrado) | desconhecido | EULA | **investigar** como referência histórica |
| WYD 2 Coreia | ✅ | desconhecido (site coreano) | não | EULA | baixa prioridade |
| WYD M | ✅ mobile | moderno, outro jogo | não | EULA | **não aplicável** |
| WYD International | ❌ **privado** | DX11, sombras, água — o mais bonito | não | privado/ilegal segundo a publicadora | **só referência visual**; não integrar |
| `open-wyd/ows-clientdata` | — (dump) | — | não | duvidoso | corpus de teste de parser |
| Source 7.59/7.69 (`GreegTech`, `felipeletsgo`, `alanpetry`) | ❌ (GPL-3.0 declarada sobre código proprietário) | permitiria cliente próprio | não | **risco alto** | só estudo; não integrar |

**Achado relevante:** existe **código-fonte de cliente** (7.48/7.59/7.69) publicado com GPL-3.0.
Isso torna a “Abordagem D” (cliente próprio) tecnicamente viável — e juridicamente arriscada,
porque a GPL não pode ser imposta sobre código de terceiro sem autorização do titular.

## 8. Recomendação

1. **Não trocar de cliente.** O 7662 é o alvo do servidor; trocar significa refazer protocolo,
   structs e todo o conhecimento de `docs/cliente/`.
2. **Corrigir o ambiente**: modo de vídeo, driver/render target, e (se possível) recuperar os
   ajustes de textura/efeito que o build atual não expõe.
3. **Medir antes de opinar**: o protocolo do §6 responde “houve downgrade?” em uma tarde.
4. **Se o objetivo é “mais bonito”**, o caminho é **pipeline de asset** (`.wyt` writer, texturas
   de maior resolução, UI redesenhada) — e isso não quebra protocolo.
5. **7.69 Global** entra como **benchmark** (mesma cidade, mesma resolução) e como referência de
   arquitetura de conteúdo — nunca como base de código/asset.

## 9. O que não foi possível confirmar

- Se o cliente Ongame de 2016 era visualmente melhor — **não há medição**; só o guia de opções.
- Qual build exato a Ongame usava.
- Se `Config.bin` bytes 12–20 controlam textura/efeito.
- Se água/chuva/sombra existem no 7662 (módulos confirmados só na linhagem 7.69).
- Se o erro de render target é driver, D3D9 ou asset.
- Se “7663” existe.

Todos estão registrados como `UNK-33`…`UNK-37` e `UNK-20` em `18-unknown-features.md`.
