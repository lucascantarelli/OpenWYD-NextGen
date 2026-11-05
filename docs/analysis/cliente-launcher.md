# Cliente 7.64 / ClientVersion 12000 — o que dá para modernizar e como fazer o launcher

Este documento separa **fato verificado** de **hipótese** e fecha com o desenho do launcher próprio
integrado ao portal. Aviso honesto: as fontes públicas sobre o build exato 7.64/12000 são escassas;
quase tudo abaixo vem de builds vizinhas (7.54, 7.59, 7662/7640) e está marcado.

## 1. O que é o cliente (fatos)

| Item | Fato | Fonte |
|------|------|-------|
| Linguagem/engine | C++ nativo, **DirectX 9** com pipeline de função fixa (D3DX), 32-bit | port WASM do OpenWyd e port web 7.59 descrevem a camada de compat "Win32, DirectInput, Direct3D 9, D3DX" — https://github.com/alanpetry/OpenWyd · https://github.com/iguinhozinho/wyd-web-client |
| Resolução legada | **800×600**, simulação a **60 Hz** desacoplada da apresentação | idem |
| Texturas | `.wys` = **DDS com 1 byte de cabeçalho** (DXT1/DXT3); atlas WYT, registros de 260 B | `TextureManager.cpp` citado no port 7.59 |
| Terreno | `.trn`: nome + setor + **4.096 registros de 12 B** (altura ×0,1, textura +10, cor, UV) | idem |
| Descoberta de servidor | `serverlist.bin` gerado por `serverlist editor.exe`; cliente consulta status em **HTTP 80** e conecta em **TCP 8281** | README do repo + `serverlist bin/` |
| Patch | `ClientPatch_v7662.dll` desliga checksum e ajusta `SkillDelay/4`; 4 pontos de extensão (ReadMessage/SendChat/UI/SendPack) | `docs/migration/client-patch-hooks.md` |
| Proteção | ofuscação por tabela estática + checksum **não-rejeitante**; sem anti-cheat real | `docs/migration/protocol-spec.md` |
| Versão do protocolo | repo/documentação dizem **7640**; `AGENTS.md` fala **12000** aceito pelo servidor | conflito a resolver na issue 49 |

> **Não verificado**: existe patch de widescreen/UI-scaling pronto para 7.64? Os fóruns onde isso
> circula (RaGEZONE, webcheats) devolveram 403 na pesquisa. Trate como "provavelmente sim, não achamos".

## 2. O que dá para mudar sem tocar no executável

| Mudança | Viável? | Como |
|---------|:-------:|------|
| Texturas, sprites, ícones, HUD (arte) | ✅ | editar o DDS dentro do `.wys` (repor o byte de prefixo) e o atlas WYT |
| Nomes, fontes do jogo, textos de skill/item | ✅ | é conteúdo de servidor (`ItemList.csv`, `SkillData.csv`) + atlas de fonte |
| Mapa, NPC, loja, drop, quest, evento | ✅ | tudo servidor — o cliente só desenha |
| Logo/título da janela, checksum off, `SkillDelay` | ✅ | hooks da `ClientPatch` por endereço |
| Posição/tamanho dos painéis (layout) | ⚠️ | hardcoded no `.exe`; exige patch binário ou fonte |
| Resolução/widescreen | ⚠️ | patch binário por build; sem fonte é caso a caso |
| FPS acima de 60 | ⚠️ | a simulação é 60 Hz por design; "desbloquear" só a apresentação |
| Anti-cheat | ❌ | o cliente é não confiável por design; a autoridade é o servidor |

## 3. Três caminhos de modernização gráfica

| Caminho | Custo | Risco | Entrega |
|---------|-------|-------|---------|
| **A. Asset pack** — trocar `.wys`/`.wyt`/`.trn` por arte nova, mesmo engine | baixo | baixo (só visual) | Melhora imediata de percepção, zero mudança de gameplay |
| **B. Recompilar o cliente C++** (fonte disponível) — D3D9 → D3D11/Vulkan, UI escalável, widescreen nativo | alto | alto (paridade byte a byte, 60 Hz, física) | Modernização real; base é o projeto OpenWyd, que já tem camada de render substituível |
| **C. Cliente web/WASM** (port existente) — jogar no navegador | médio | médio (rede, input, assets 500 MB) | Canal alternativo, útil para testes e marketing; não substitui o cliente desktop no curto prazo |

Recomendação: **A agora**, **B como projeto separado com orçamento próprio**, **C apenas como
laboratório**. Nenhum dos três é pré-requisito para o servidor funcionar.

## 4. Launcher próprio — desenho

Objetivo: controlar distribuição e atualização sem depender de link de fórum, e integrar login/cash
com o portal. Regra de ouro: **o launcher não distribui binário de terceiro**; ele atualiza a cópia
que o jogador já possui.

```
Launcher (Go + tray, Windows)            Portal/Next.js (BFF)              Servidor
  ├─ manifest.json (versão, hash) ◄────── /api/launcher/manifest  ◄──────── release DB
  ├─ baixa só o que mudou (diff por arquivo) ── /api/launcher/file/<sha>
  ├─ verifica assinatura (ed25519) e aplica patch atômico (tmp → rename)
  ├─ abre o jogo com serverlist.bin gerado na hora (IP/porta do servidor)
  └─ login (OIDC/sessão do portal) ─────► play code de uso único (já existe: IssuePlayCode)
```

Entregas por fase:

1. **Fase 1 — launcher mínimo** (issue 51): manifesto versionado, patch por hash, geração local do
   `serverlist.bin`, botão "jogar", log de erro. Sem login.
2. **Fase 2 — identidade**: login pelo portal, `play code` de uso único, "lembrar conta", bloqueio de
   conta refletido na hora (consulta `AccountAdminService`).
3. **Fase 3 — operação**: canal de notícias, status do servidor (usa `/status` :80 e health do portal),
   auto-reparo de arquivo corrompido, telemetria opt-in (crash id, versão, tempo de patch).
4. **Fase 4 — distribuição**: assinatura de manifesto, mirror/CDN, rollback de versão, beta channel.

Requisitos técnicos: patch **atômico e retomável**; nunca sobrescrever arquivo em uso; verificar hash
antes de aplicar; manifesto assinado; **nada de segredo no launcher**; `serverlist.bin` gerado com o
IP/porta que o servidor informa (hoje `scripts/run-local.sh` imprime isso).

## 5. `serverlist bin/` — o que é e o que fazer

- Contém `serverlist editor.exe` + **4 pares** de `serverlist.bin` (raiz e `client/`) gerados para
  IPs privados (`192.168.18.12`, `192.168.2.103/.105/.107`).
- É ferramenta + **artefato**, não código. Hoje está **versionado por engano** e viola `AGENTS.md`
  (nunca versionar `serverlist.bin` gerado com IP privado).
- Migração para o portal: gerar o bin no BFF a partir do IP/porta configurados (o formato é pequeno e
  conhecido) e oferecer download autenticado; o editor `.exe` sai do repositório.
- Ação imediata (issue 56): `git rm --cached` nos `.bin`, `.gitignore` para `serverlist bin/**/*.bin`,
  manter só o editor (ou removê-lo também, já que o portal passará a gerar).

## 6. Riscos

| Risco | Mitigação |
|-------|-----------|
| Falso positivo de antivírus em `WYD.exe` reempacotado | nunca reempacotar o exe; distribuir patch separado, assinar o launcher, publicar hashes |
| Cliente de origem desconhecida com launcher/IP alterados | launcher valida hash do exe e do patch; documentar a origem |
| Offsets do patch mudam entre builds (7640 vs 12000) | medir com captura antes de qualquer patch; manter mapa por build |
| Assets de terceiros / direitos autorais | não redistribuir assets em massa; patch só do que é nosso (UI própria, ícones autorais) |
| Widescreen quebrar hitbox/UI | testar em 4:3 e 16:9 com scripts de regressão visual |
