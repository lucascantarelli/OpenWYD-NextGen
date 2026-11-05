# Issue 52 — Cliente 7.64/12000: mapa de offsets, assets e avaliação de resolução

**Fase:** 9 · **Depende de:** 49 (versão medida) · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Produzir o **mapa técnico do build que usamos** (12000) — offsets da `ClientPatch`, formato do
`serverlist.bin`, limites de resolução/UI — e entregar o primeiro **asset pack** próprio. Decidir, com
dados, se vale recompilar o cliente.

## Contexto

- `docs/migration/client-patch-hooks.md` mapeia endereços do **7662**; o build do usuário é outro
  (7.64/12000). Aplicar patch de outra build corrompe o exe.
- Engine: DirectX 9, pipeline fixo, 800×600, simulação 60 Hz — ver
  [`docs/analysis/cliente-launcher.md`](../../docs/analysis/cliente-launcher.md) §1–§3.
- `.wys` = DDS + 1 byte; `.wyt`, `.trn` e `itemicon.bin` são documentados por ports públicos.

## Passos

1. **Inventário do build**: hash de `WYD.exe`/`ClientPatch.dll`, versão de protocolo, mapa de offsets
   (título, checksum, `SkillDelay`, força-gráficos) validado por teste de fumaça no cliente.
2. **Formato do `serverlist.bin`**: documentar campos/tamanho a partir dos arquivos existentes e do
   editor; escrever parser/gerador em Go (base do launcher).
3. **Asset pack v1**: substituir um conjunto pequeno e visível (logo, tela de login, ícones de UI) por
   arte própria, mantendo o byte de prefixo do `.wys`.
4. **Avaliação de resolução**: testar patch widescreen conhecido (se existir para esta build) e medir
   hitbox/UI; registrar resultado — viável/não viável com evidência.
5. **Decisão de engine**: comparar custo/risco de recompilar (OpenWyd como base) × manter D3D9 com
   assets novos; registrar ADR.

## Arquivos

- `docs/migration/client-patch-hooks.md` (seção do build 12000),
  `docs/analysis/cliente-launcher.md`, `client-tools/serverlist/` (parser/gerador Go),
  `docs/adr/0002-cliente-modernizacao.md` (novo), assets em `images/`/pacote de patch.

## Verificação

- Parser/gerador: `go test ./client-tools/serverlist/...` com os 4 `.bin` existentes (round-trip).
- Patch: cliente abre, loga e joga; checksum desligado; `SkillDelay` compatível com o servidor.
- Asset pack: cliente abre tela de login com a arte nova (captura anexada).

## Critérios de aceite

- [ ] Mapa de offsets **do nosso build** documentado e validado (não copiado do 7662)
- [ ] `serverlist.bin` gerado por código nosso, com round-trip nos arquivos atuais
- [ ] Asset pack v1 aplicado sem quebrar login/render
- [ ] Relatório de resolução: o que funciona, o que quebra, custo
- [ ] ADR de engine registrado (manter D3D9 × recompilar × WASM)

## Riscos

- **Offsets errados** → exe corrompido: sempre patch em cópia, hash antes/depois, rollback.
- **Direitos autorais**: só arte própria; nunca redistribuir assets originais.
