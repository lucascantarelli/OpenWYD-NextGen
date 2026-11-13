# 13 — Referências externas e hierarquia de confiança

> **Consulta:** 2026-11-13. Fontes classificadas pelo mesmo esquema de `docs/wyd/fontes.md`
> (A oficial · B servidor oficial · C privado com guia técnico · D comunidade · E dado do repo).
> **Regra:** mecânica é estável, número não é. Nenhum número de servidor privado vira regra.

## 1. Hierarquia aplicada

1. Comportamento comprovado no cliente `client/` e no legado (**E**).
2. Implementação oficial do WYD2 Ongame (**A**).
3. WYD Internacional (**⚠️ privado** — nível C).
4. WYD Global atual (**B**).
5. Documentação oficial e materiais técnicos (**B**).
6. Repositórios públicos independentes (**C/D**).
7. Fóruns e comunidades (**D**).
8. Servidores privados (**C**).
9. Hipóteses (**—**).

## 2. Referência 1 — WYD2 Ongame Brasil (histórica, nível A)

| Item | Valor | Fonte |
|---|---|---|
| Publicadora | Ongame (licença JoyImpact), lançado 2005 | EverybodyWiki / imprensa |
| Encerramento | **25/07/2016** (anúncio em 30/04/2016) | `gamevicio.com` (2016-04) |
| Guia básico/intermediário (arquivado) | grupo/XP, skills, guilda, cidadania, refino +1…+11, montaria, chat, C.C. | `web.archive.org/web/20101224192748/http://wyd.ongame.com.br/guia/?m=2` |
| Criação de guilda | `/criar`, 100kk, 3 sub-líderes, medalhas | Wayback, guia 2010 |
| Montaria | 8 espécies, 10 cavalos, ovo→cria→adulto | Wayback, guia 2010 |
| Interface | EXP em 3 lâmpadas | Wayback, guia 2010 |
| **Opções gráficas** | resolução, “Textura: comprimido/descomprimido”, “Efeito: Min/Max”, brilho | Wayback, guia 2011 |

**Uso no projeto:** mecânica da era 7.x é referência válida para o 7662. **Os números são de
época** e não publicam percentuais de refino.

## 3. Referência 2 — WYD Internacional (⚠️ servidor privado)

| Item | Valor |
|---|---|
| Natureza | **privado**; declara não ser afiliado à publicadora |
| Relevância | publica taxas por passo e descreve algoritmo de drop — nível **C** |
| Renderer 2026 | anuncia DX11, sombras dinâmicas, água com espuma, chuva/trovoada, céu melhorado |
| Limite | **nenhum** número dele entra como regra; serve de hipótese e de comparação visual |

## 4. Referência 3 — WYD Global (oficial licenciado, nível B)

| Sistema | Fonte | Trecho citável |
|---|---|---|
| Hub de guias | `wydglobal.raidhut.com/en/2411` | “guides… indispensable tool for WYD adventurers” |
| Base de conhecimento | `feedback.kersef.com/pt-br/knowledge-bases/21-faq-wyd-global` | mapas Armia/Nippleheim/Kefra, Chaos, Kingdom |
| Refino Bahamut | `wydglobal.raidhut.com/en/2747` | +1~+5 20% · +5~+9 15% · +9~+12 10% · +12~+15 5%; falha consome itens exceto Bahamut |
| Refino celestial +11→+15 | `wydglobal.raidhut.com/en/873` | NPC “Alchemist Odin” |
| Fadas | `feedback.kersef.com/.../58182` | 5 fadas; azul 3 dias, 80 NPs |
| Dragões | `wydglobal.raidhut.com/en/2733` | God2 nível 300+, 1.000.000 gold, Dragon Guardian |
| Guildas/guerra | `wydglobal.raidhut.com/en/2120` | Noatun War a cada 2 semanas, domingo 21h–22h (GMT-3) |
| Reino | `feedback.kersef.com/.../20122` | ingresso nível **220**, taxa em safiras |
| Chaos/PK | `feedback.kersef.com/.../20250` | “less than **−65** chaos points” (texto EN) × “**−75**” (PT) → **divergente** |
| Cerco | `feedback.kersef.com/.../20258` | Castle de Noatun em duas zonas |
| Montaria | `feedback.kersef.com/.../20121` | 8 espécies, 10 cavalos; evolução 0→179 |
| Quests iniciais | `wydglobal.raidhut.com/en/2403` | 4 estágios até “Orc Emblem”, 3.000 EXP |

**Publicadora atual:** Raid Hut (licenciada JoyImpact) desde fev/2019. **Desenvolvedora:**
`wyd2.co.kr` (JoyImpact), em coreano, sem URLs estáveis por sistema.

## 5. Referência 4 — servidores privados e repositórios (C/D)

| Fonte | Licença | Estado | Uso |
|---|---|---|---|
| `github.com/Jean1dev/w2pp-OpenWYD` | **GPL-2.0** | Go, 672 commits, ativo (2026-10-29) | **melhor doc técnica** de formato/regra, com `arquivo:linha` |
| `github.com/alanpetry/OpenWyd` | **sem licença declarada** | C++ do cliente + porte WASM/WebGL | inventário de 8.550 assets, loaders por formato |
| `github.com/open-wyd/ows-clientdata` | — | dump de cliente (~279 MB) | **referência**, nunca conteúdo do nosso seed |
| `github.com/felipeletsgo/wyd748` | GPL-3.0 | Go + C++ (7.69 adaptado a 7.48) | comparação de dialeto |
| `github.com/GreegTech/Source-759-WYD` | GPL-3.0 | só README/LICENSE no índice | **não verificado** |
| `github.com/Amatsukan/WYD-Server-CPP` | estudo | C++ decompilado | comparação de handler |
| `github.com/Rechdan/WYD-ItemList-Editor` | GPL-3.0 | C# | editor do `.bin` de **servidor** |
| `github.com/whyrenan/wyd-fieldscene-studio-pro` | **MIT** | HTML/JS | parsing de BIN de UI |
| `github.com/whyrenan/wyd-uitexturelistn-converter` | MIT | HTML/JS | `UITextureListN.bin` |
| `github.com/azraelexe94/WYT-CONVERT` | sem fonte | `.exe` | `.wyt`⇄`.tga` em massa |
| `github.com/iguinhozinho/wyd-web-client` | — | JS | loaders de `.trn`/`.wys` |
| `github.com/FoxdemoX/WydMapEditor` | **sem licença** | C#/OpenGL, 3 commits | editor de mapa — **risco jurídico** |
| `github.com/kevinkouketsu/WYD2Bot` | GPL-3.0 | C# | lê HeightMap/ItemList/SkillData |

**Armadilha:** `acgfbr/wyd-client` aparece em busca mas retorna 404 — **não existe/não é público**.

## 6. Clientes oficiais disponíveis hoje

| Cliente | Status | URL | Uso para nós |
|---|---|---|---|
| **WYD Global** (Raid Hut + JoyImpact) | oficial, ativo (launcher 2.7.3) | `wydglobal.raidhut.com/en/download` | **referência de escala/estrutura** (7.69+); não é contrato |
| **WYD Origem / Asgard** (Raid Hut BR) | oficial licenciado, interface legada | `wyd.raidhut.com` | referência de UI clássica |
| **WYD 2 Coreia** | oficial (JoyImpact) | `wyd2.co.kr` | sem documentação acessível |
| **WYD M** | oficial mobile/Windows (2022/2025) | Google Play | **não aplicável** |
| Ongame Brasil | **morto** (2016) | Wayback | histórico |
| WYD Internacional | ⚠️ privado | `wyd-international.com` | só hipótese/taxa |

**Nenhum cliente oficial é redistribuível.** Servem para medir e comparar, nunca para copiar
asset ou layout de struct.

## 7. Correções de premissa que valem registro

| Premissa comum | Correção | Nível |
|---|---|---|
| “Existiu um cliente 2D com sprites, depois virou 3D” | todos os clientes de PC documentados são D3D com câmera fixa; o 7662 tem skin/bones/anim | Confirmado |
| “Kefra, Armia e Nippleheim são reinos” | reinos são **Hekalotia** e **Akeronia**; Armia é cidade, Nippleheim é região, Kefra é dungeon/boss | Confirmado (fonte oficial) |
| “Familiares é sistema oficial” | não há sistema com esse nome na documentação oficial; há **montarias** e animais do BeastMaster | Fortemente indicado |
| “7663 existe” | zero evidência; 7662 é **build** com protocolo **7640** | Confirmado |
| “O cliente melhorou com o Global” | o Global tem cliente 7.69+ com mais conteúdo; o renderer moderno de 2026 é de **servidor privado** | Confirmado |
| “Taxa de refino X é do WYD2” | taxas variam por servidor e patch; a oficial publicada é do Global (Bahamut) | Confirmado |
| “Existe wiki oficial” | não existe; há hub de guias + KB (Kersef) | Confirmado |

## 8. Registro de pesquisa (formato exigido)

| Campo | Valor |
|---|---|
| Data da consulta | **2026-11-13** |
| Consultas | cliente/gerações visuais; formatos e ferramentas; sistemas oficiais |
| Fontes primárias | Wayback (Ongame), `wydglobal.raidhut.com`, `feedback.kersef.com`, GitHub API, `en.wikipedia.org` |
| Informação extraída | ver §2–§6 |
| Confiabilidade | A/B/C/D conforme marcado |
| Comparação com o código local | feita em `04`, `10`, `11`, `12`, `19` |
| Divergências | chaos −65 × −75; `ItemList.bin` 140 × 164 B; `SkillData` 96 × 104 B; `familiares` oficial × interno |
| O que não foi confirmado | versão do cliente Ongame em 2016; existência de 7663; semântica de `.msa` × `.msh` × `.wys`; refino clássico em % |
