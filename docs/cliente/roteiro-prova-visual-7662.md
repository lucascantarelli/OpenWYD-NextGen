# Roteiro consolidado de prova visual — cliente 7662

> **Issues cobertas:** `BUG-01`, `BUG-02`, `GAME-17`, `GAME-18`.
> **Data do roteiro:** 2026-11-12.
> **Natureza:** plano de execução. A tentativa de 2026-11-12 chegou ao server select, mas foi
> bloqueada por entrada sintética; nenhum critério visual foi marcado. Ver §9.
> **Regra:** cada item só vira ✅ com **print + log do cliente/canal + estado do servidor**.
> Log de servidor sozinho prova o fio, não a tela; print sozinho não prova o frame.

## 1. Resultado esperado

Ao final, o operador deve conseguir responder, com evidência:

| Issue | Pergunta que a prova precisa responder |
|---|---|
| `BUG-01` | As recusas de senha/PIN aparecem no 7662? O admin de teste entra, vê a cena e reloga? |
| `BUG-02` | Conta sem PIN entra sem “Falha ao conectar”? PIN correto entra, PIN errado recusa e PIN tardio não abre segunda sessão? |
| `GAME-17` | O personagem entra com `ClientID < 1000`, `UpdateScore` antes da cena e `CreateType=2` só no self? |
| `GAME-18` | O Campo mostra os 11 NPCs, permite as quatro etapas, portões, buff, boss e relogin? |

O roteiro **não fecha** nenhuma issue. O fechamento é uma segunda etapa, depois de anexar a
evidência e confrontar cada critério de aceite.

## 2. Pré-condições

### 2.1 Cliente e rede

- [ ] `client/WYD.exe` é o build **7662**, SHA-256
  `04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8`.
  Hash diferente: **parar**, não é o build do roteiro.
- [ ] Backup de `client/` feito fora do repositório; `client/` não é versionado.
- [ ] VM Windows descartável (ou máquina de teste) com D3D9 funcional; o cliente não deve
  rodar na máquina de trabalho com a conta do dono.
- [ ] `serverlist.bin` aponta para o host do canal de teste na porta **8281**; o cliente fixa
  essa porta. Use o launcher/ferramenta para gerar o `.bin`, nunca edite à mão.
- [ ] O host do canal responde na `8281/tcp` e no status HTTP `:80/serv00.htm`.
- [ ] Protocolo do cliente é **12000**; divergência derruba o login e não é objeto deste
  roteiro.
- [ ] `WYD.log` do cliente está acessível em `client/WYD.log`.

Comandos de conferência (na máquina do cliente ou no host, conforme o caso):

```bash
sha256sum client/WYD.exe
node tools/client/serverlist.mjs --print client/serverlist.bin | head -5
curl -fsS http://<host-do-canal>/serv00.htm
```

### 2.2 Stack e contas

- [ ] Stack de teste no ar: `make stack-up` ou
  `docker compose up -d --build --wait`.
- [ ] Canal 1 saudável e acessível pelo IP configurado no `serverlist.bin`.
- [ ] Gateway/portal saudáveis, se forem usados para preparar conta/personagem:
  `curl -fsS http://127.0.0.1:7600/health` e
  `curl -fsS http://127.0.0.1:3000/api/health`.
- [ ] Conta **sem PIN** para a entrada padrão (o provisionamento de teste cria `qavivo01`;
  confira o slot 0 e a senha sem imprimi-la no terminal).
- [ ] Conta **com PIN** para o caso de PIN correto/errado; se não existir, crie uma conta de
  teste dedicada, nunca a conta do dono.
- [ ] Conta **admin de teste** para o critério de `BUG-01` (“admin entra no mundo”); papel
  admin, personagem em slot conhecido, sem dados pessoais.
- [ ] Conta **descartável** para o bloqueio de PIN (5 falhas = 10 min) e para seleção
  inválida. Não use `qavivo01` para esses casos.
- [ ] Personagem de teste para `GAME-18` com nível **1..35**, posição inicial conhecida e
  dano suficiente para abater os guardiões. O nível 1 do snapshot bate **1** de dano em
  `GremlinDecoy` (598 HP); se só houver esse personagem, registre o bloqueio e **não** reduza
  HP no cliente/servidor para simular morte.
- [ ] Nenhuma conta do dono, nenhuma credencial real e nenhum PIN em print, log ou arquivo
  versionado.

Provisionamento idempotente da conta de teste (quando o banco de teste estiver configurado):

```bash
WYD_TEST_DSN="$WYD_TEST_DSN" \
  cargo test -p wyd-server --test sistema_conta_de_teste -- --nocapture
```

O teste imprime `CONTA=`, `SENHA=` e `SLOT=0`; exporte a senha em
`WYD_LIVE_PASSWORD` sem colá-la no documento.

### 2.3 Captura e higiene

- [ ] Pasta de trabalho `tmp/prova-visual-7662/` (ignorada pelo git).
- [ ] Nome dos prints: `A1-senha-errada.png`, `B2-primeiro-treinador.png`,
  `C3-gremlin-drop.png`, etc.; sempre com hora e coordenada quando for cena.
- [ ] Log do canal coletado no mesmo intervalo:
  `docker compose logs --since 15m server-canal-1 | tee tmp/prova-visual-7662/canal.log`.
- [ ] `client/WYD.log` copiado para a pasta da prova.
- [ ] Se for necessário capturar frame, usar Wireshark com filtro `tcp.port == 8281` ou os
  breakpoints de `docs/cliente/depuracao.md` §2. Não versionar `.pcap`, dump ou log bruto.
- [ ] Conta/senha/PIN mascarados antes de qualquer anexo versionado; servidor privado não
  entra em print.

## 3. Sessão A — login, recusas e entrada no mundo

**Objetivo:** cobrir `BUG-01` A1–A6, `BUG-02` e `GAME-17` numa única sessão, sem repetir
login. Faça primeiro os casos sem estado destrutivo; deixe bloqueio de PIN e seleção inválida
para o fim.

### A0. Prova viva antes da tela (opcional, mas recomendada)

Com a conta sem PIN e o canal real:

```bash
WYD_LIVE_CPSOCK=<host-do-canal>:8281 \
WYD_LIVE_ACCOUNT=qavivo01 \
WYD_LIVE_PASSWORD="$WYD_LIVE_PASSWORD" \
WYD_LIVE_SKIP_PIN=1 \
WYD_LIVE_CHARACTER_SLOT=0 \
  cargo test -p wyd-server --test sistema_cena_mobs -- --nocapture
```

Esperado no log: `pin result=skip reason=not_registered` ou
`pin result=deferred reason=selection_first`, seguido de
`character ok action=select`, `UpdateScore`, `CreateMob` e resposta de diálogo de NPC.
Isso prova o fio; **não** substitui o print da entrada.

- [ ] Teste vivo passou com a conta sem PIN
- [ ] Tail do log salvo na pasta da prova

### A1. Senha errada

1. Abrir o 7662 e tentar login com a conta de teste e senha errada.
2. Observar a tela por pelo menos 5 s; não fechar imediatamente.
3. Capturar o print e o log do canal.

**Esperado:** o cliente mostra a recusa (texto atual `Senha ou conta incorreta`) e o log
registra `NewAccountFail` + `MessagePanel` (0x0101, corpo de 140 B), motivo `Senha`.

**Se não aparecer:** registrar o comportamento exato e o frame que saiu. `0x0117`
(`NewAccountFail`) não é desenhado pelo 7662; o texto visível depende de `0x0101`. Não
alterar `reply_recusa` nesta sessão; a hipótese `Size=12` puro continua **UNVERIFIED** até a
captura.

- [ ] Print da recusa (ou da ausência)
- [ ] `Size`/opcode/motivo do frame no log
- [ ] Veredito: exibe / não exibe / bloqueado por cliente

### A2. PIN errado

1. Entrar com a conta **com PIN**, informar PIN errado uma vez.
2. Observar a mensagem e se a conexão cai antes do desenho.

**Esperado:** recusa visível e log `AccountSecureFail` com motivo `PinIncorreto`; a conexão
não deve cair antes da mensagem.

- [ ] Print da recusa
- [ ] Log `AccountSecureFail`/motivo
- [ ] Veredito: exibe / não exibe

### A3. PIN bloqueado (conta descartável)

1. Usar a conta descartável, nunca `qavivo01`.
2. Errar o PIN até o limite (5 falhas; bloqueio de 10 min).
3. Confirmar o aviso de bloqueio e uma nova tentativa recusada.

**Esperado:** aviso de bloqueio no cliente e log com `PinBloqueado`/contador. Se o cliente
não exibir aviso, registrar a limitação; não desbloquear a conta de teste principal.

- [ ] Print do aviso (ou ausência)
- [ ] Log de bloqueio
- [ ] Conta descartável identificada apenas por rótulo, sem senha/PIN

### A4. Segunda conexão com a mesma conta

1. Abrir duas instâncias do cliente/VM (ou uma VM clonada) com a mesma conta.
2. Entrar com a primeira; tentar a segunda.
3. Observar o que a segunda mostra e o que o canal registra.

**Esperado:** aviso de conta em uso/sessão viva, ou recusa visível; log
`event=login result=conflict` com `origin_channel`.

- [ ] Print da segunda instância
- [ ] Log dos dois canais
- [ ] Veredito: exibe / apenas desconecta / bloqueado por ambiente

### A5. Seleção inválida (conta descartável)

1. Em uma conta descartável, criar um personagem e abrir a seleção.
2. Remover esse personagem pelo fluxo normal/portal em outra sessão.
3. Tentar selecionar o slot antigo no cliente aberto.

**Esperado:** recusa visível sem entrar no mundo; log
`SelecaoInvalida`/`PersonagemIndisponivel`. Se o cliente atualizar a lista e não enviar a
seleção, registrar “não reproduzível neste cliente” — não inventar um pacote.

- [ ] Print da seleção/recusa
- [ ] Log do motivo
- [ ] Veredito: recusa / lista atualizada / não reproduzível

### A6. Canal/zona indisponível (condicional)

Só execute se houver um canal/zona descartável que possa ser colocado em manutenção sem
interromper o teste em andamento. Caso contrário, registre **N/A** e não reinicie serviço.

**Esperado:** recusa visível quando aplicável; log `ZonaIndisponivel`/manutenção.

- [ ] Print (se executado)
- [ ] Log do motivo
- [ ] N/A documentado, se não houver ambiente descartável

### A7. Entrada no mundo — conta sem PIN

1. Abrir o 7662, autenticar com a conta sem PIN e escolher o personagem do slot conhecido.
2. Confirmar que **não** aparece “Falha ao conectar”.
3. Capturar a cena com personagem, NPCs próximos e HUD (HP/MP/EXP).
4. Anotar coordenada e horário.
5. Sair pelo caminho normal e entrar de novo imediatamente, sem esperar timeout.

**Esperado no log:** `pin result=skip`/`deferred`, `character ok action=select`,
`CNFCharacterLogin`, `UpdateScore` e `ESCENE_FIELD`; `client_id` do jogador `< 1000`.

- [ ] Sem “Falha ao conectar”
- [ ] Personagem próprio visível
- [ ] NPCs e HUD renderizados
- [ ] `client_id` na faixa de jogador no log
- [ ] Relogin imediato sem timeout
- [ ] Print da cena + tail do log

### A8. Entrada com PIN e PIN tardio

1. Repetir A7 com a conta que tem PIN, informando o PIN correto.
2. Repetir com PIN errado e confirmar a recusa visível.
3. Confirmar no log que o PIN tardio não abre segunda sessão (`pin result=ok mode=late`).

- [ ] Conta com PIN correto entra normalmente
- [ ] PIN errado continua recusando com mensagem visível
- [ ] PIN tardio não abre segunda sessão
- [ ] Print + log `pin result=...`/`character ok`

### A9. Admin de teste entra no mundo

1. Repetir A7 com a conta **admin de teste** (nunca a conta do dono).
2. Confirmar entrada, cena, saída e relogin.
3. Registrar o `client_id` e o motivo de o personagem ser admin.

- [ ] Admin de teste entra e vê a cena
- [ ] Saída/relogin imediatos
- [ ] Print + log sem credenciais

## 4. Sessão B — Campo de Treinamento (`GAME-18`)

**Pré-condição:** A7/A9 passaram. Se a entrada no mundo falhar, **parar** e registrar
`GAME-18` como bloqueada por `GAME-17`; não tentar compensar com teleporte ou alteração de HP.

O Campo fica a aproximadamente 33 tiles do spawn. Andar até lá e confirmar a cena por
proximidade antes de interagir. Usar a conta/personagem de teste preparado, com dano
suficiente para os guardiões.

### B1. NPCs do Campo

Olhar em volta e capturar um print amplo.

**Esperado:** 11 NPCs do seed: 4 treinadores, Unicorn, Cabuncle, Sephira +9, 3 guardiões e
OrcWarrior. O log de cena deve mostrar os `CreateMob` correspondentes.

- [ ] Print amplo com coordenada
- [ ] Log de cena com os 11 NPCs
- [ ] Os quatro treinadores e o Chefe identificáveis

### B2. Primeiro Treinador

Falar com o Primeiro Treinador.

**Esperado:** o primeiro clique **aceita** a etapa 1 (`Campo de Treinamento: etapa 1/4
(aceita)`), sem “falta a prova 451”; `character_quest step=1 progress=0`.

- [ ] Print do diálogo
- [ ] Log/estado `step=1 progress=0`
- [ ] Nenhuma recusa por prova ausente

### B3. `GremlinDecoy` e chave 451

1. Abater `GremlinDecoy`.
2. Conferir o inventário/drop.
3. Se a mochila estiver cheia, confirmar que o drop fica no chão.

**Esperado:** chave 451 no inventário (ou no chão, se cheio), com log de loot/quest.

- [ ] Print do drop/inventário
- [ ] Log de loot/`quest`
- [ ] Chave 451 visível

### B4. Portão 1

Usar a chave 451 no portão.

**Esperado:** portão visível abrindo, chave consumida e fechamento após o tempo (30 s).

**Bloqueio conhecido:** o portão ainda não é entidade visível no snapshot; o cliente pode não
receber `_MSG_UpdateItem`/`EF_KEYID`. Se não aparecer, registrar o ponto exato e **não**
simular visual; a etapa visual de portão continua aberta.

- [ ] Print antes/depois
- [ ] Log do consumo da chave
- [ ] Veredito: visível / ausente com bloqueio registrado

### B5. Etapas 2 e 3

Repetir com `GiantCrill`/452 e `OrcSniper`/453; falar com o Segundo e o Terceiro Treinador.

**Esperado:** etapas aceitas, `Equip[6]` encantado, `Equip[1..7]` + sanc 3, portões 2 e 3
abrindo. Se o portão visual estiver ausente, registrar o bloqueio e não marcar a parte visual.

- [ ] Prints dos treinadores/equipamento
- [ ] Logs das etapas e chaves
- [ ] Portões 2/3: visíveis ou bloqueio registrado

### B6. `Cabuncle` — buff

Falar com o `Cabuncle` e observar HP/MP/ícone/duração; repetir para testar reaplicação.

**Esperado:** affect/duração visível e reaplicável. **Bloqueio conhecido:** o affect do buff
ainda não tem pacote de ícone/duração no fio; se não aparecer, registrar e não inventar.

- [ ] Print antes/depois
- [ ] Log do affect (se houver)
- [ ] Veredito: visível / ausente com bloqueio registrado

### B7. Boss e recompensa final

1. Abater `OrcWarrior`.
2. Conferir Emblema 524 no inventário.
3. Falar com o Chefe e conferir a recompensa sorteada (682/481/652), uma única vez.

- [ ] Print do Emblema/inventário
- [ ] Print da recompensa
- [ ] Log da entrega única
- [ ] Repetir a entrega é recusado

### B8. Relogin e persistência

Sair e relogar; conferir etapa/conclusão e inventário.

**Esperado:** estado persiste; repetir a entrega não duplica recompensa; sem espera de
timeout.

- [ ] Print após relogin
- [ ] Estado `character_quest`/inventário
- [ ] Reentrega recusada

## 5. Matriz de decisão

Preencher somente com evidência anexada. `B` = bloqueado; `N/A` = não aplicável.

| Issue | Critério | Evidência | Veredito |
|---|---|---|---|
| `BUG-01` | A1 senha errada visível ou limitação documentada | `13-connect-elevated-640.png`; o login não abriu | **B** |
| `BUG-01` | A2–A6 matriz mensagem × exibe/não exibe | não executado; depende do login | **B** |
| `BUG-01` | Admin de teste entra, vê cena, sai e reloga | não executado | **B** |
| `BUG-01` | `docs/cliente/pacotes.md` e checklist §1 atualizados | sem frame novo; nada a atualizar | **B** |
| `BUG-02` | Conta sem PIN entra sem “Falha ao conectar” | não executado | **B** |
| `BUG-02` | PIN correto entra; PIN errado recusa visivelmente | não executado | **B** |
| `BUG-02` | PIN tardio não abre segunda sessão | não executado | **B** |
| `GAME-17` | `ClientID < 1000`, `UpdateScore` antes da cena, `CreateType=2` no self | provado no fio em rodada anterior; sem print novo | **B** (visual) |
| `GAME-17` | Sem “Falha ao conectar”; personagem/NPCs/HUD visíveis | não executado | **B** |
| `GAME-17` | Relogin imediato | não executado | **B** |
| `GAME-18` | 11 NPCs visíveis | Sessão B, não executada | **N/A nesta sessão** |
| `GAME-18` | Quatro etapas, provas e recompensas | Sessão B, não executada | **N/A nesta sessão** |
| `GAME-18` | Portão visível, chave consumida, fecha | Sessão B, não executada | **N/A nesta sessão** |
| `GAME-18` | Buff do `Cabuncle` visível/reaplicável | Sessão B, não executada | **N/A nesta sessão** |
| `GAME-18` | Boss, Emblema e recompensa única | Sessão B, não executada | **N/A nesta sessão** |
| `GAME-18` | Relogin/persistência | Sessão B, não executada | **N/A nesta sessão** |

> **Tentativa de 2026-11-12:** `B` = bloqueado por entrada sintética no cliente 7662, mesmo
> depois de corrigir a geometria (`Config.bin` modo 7, cliente `800×600`). Nenhum critério
> visual foi marcado; o diagnóstico de vídeo e os arquivos estão no §9.5.

**Regra de fechamento:** só remover a issue e registrar no changelog quando todos os
critérios aplicáveis estiverem ✅. `GAME-18` não fecha se portão ou buff continuarem sem
captura; `BUG-01` não fecha só com o log do servidor; `GAME-17` não fecha sem print da
entrada; `BUG-02` não fecha sem a prova sem PIN e a recusa visível.

## 6. Empacotamento da evidência

```
tmp/prova-visual-7662/
  ambiente.txt          # hash, serverlist (sem IP privado no versionado), schema, TAG
  canal.log             # docker compose logs --since ...
  wyd-client.log        # cópia de client/WYD.log
  A1-senha-errada.png
  A2-pin-errado.png
  A7-entrada.png
  B1-campo.png
  B2-primeiro-treinador.png
  ...
  resumo.md             # matriz preenchida + veredito + bloqueios
```

No `resumo.md`, registrar para cada item: issue, passo, horário, print, trecho do log,
esperado, observado e veredito. Não anexar `.pcap`, dump, senha, PIN ou `serverlist.bin` com
IP privado.

## 7. Skills e normas de apoio

- `wyd-client-connect` — serverlist, porta 8281, versão 12000 e diagnóstico de conexão.
- `wyd-client-reverso` — opcodes, frames, breakpoints e captura sem inventar layout.
- `wyd-rust-testing` — teste vivo e separação entre prova de fio e prova visual.
- `wyd-rust-net` — framing, checksum e recusas do CPSock.
- `wyd-game-admin` — conta/personagem de teste e permissões, sem usar a conta do dono.
- `rust-test-quality` — evidência, nomenclatura e gate da prova.
- Normas: `docs/cliente/depuracao.md`, `docs/cliente/pacotes.md`,
  `docs/arquitetura/protocolo.md`, `docs/rust/02-execucao.md` e
  `docs/rust/08-checklist-funcionalidades.md`.

## 8. Limites

- A prova visual depende de VM/cliente 7662 e de um operador humano; não é executável apenas
  com `cargo test`.
- `GAME-18` tem bloqueios conhecidos de portão visível e buff do `Cabuncle`; o roteiro os
  registra, não os resolve.
- O nível 1 do snapshot não abate os guardiões com dano 1; sem personagem equipado, B3–B7
  ficam bloqueadas e isso deve ser registrado, não simulado.
- Nenhuma alteração de código deve ser feita durante a captura. Correções só depois de
  correlacionar print, frame e log.

## 9. Execução de 2026-11-12 — bloqueada por entrada sintética

**Resultado:** a Sessão A não passou do server select. O cliente abriu, desenhou `WYD`,
`WYD-1`, `WYD-2` e os botões `Conectar`/`Fechar`, mas nenhuma entrada sintética alterou a tela.
Nenhum critério de aceite visual foi marcado.

### 9.1 O que foi confirmado

- `client/WYD.exe` PID 13732, SHA-256
  `04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8`; `serverlist.bin` com
  canais `192.168.100.3:8281` e `192.168.100.4:8281`.
- Stack `docker compose ps` saudável, incluindo `server-canal-1`, `server-canal-2`, `db`,
  `gateway`, `portal`, `loki`, `prometheus`, `grafana`, `mkdocs` e `alloy`.
- `client/WYD.log` registra `VertexShader HW Accel Enabled` e
  `Error in Init Render Target Texture`; o erro é **não fatal** nesta execução porque o server
  select renderizou (`03-screen-640.png`).
- A janela está `Enabled=True`, `Visible=True`, foreground `WYD CDK`, classe
  `With Your Destiny`, retângulo `0,0,1030,788`, cliente `1024x759`, origem `3,26`.
- O canal 1 não registrou `event=login`, `event=handshake`, `event=session`, `event=pin` ou
  `event=character` no intervalo; nenhuma conexão TCP estabelecida em `8281`. O cliente não
  chegou a tentar autenticar.

### 9.2 O que falhou

| Tentativa | Onde | Resultado |
|---|---|---|
| `mouse_event` elevado | `WYD-1`, canal, `Conectar` | tela inalterada; login não abriu |
| `WM_LBUTTONDOWN`/`WM_LBUTTONUP` | `Fechar` (`cliente 509,565`) | cliente continuou aberto |
| `SendInput` absoluto (4 eventos) | `Fechar` | cliente continuou aberto |
| `WScript.Shell.SendKeys` elevado | `{TAB}`, `{ENTER}` | tela inalterada |
| driver em loop | fila de comandos | bloqueado pelo antivírus (`ScriptContainedMaliciousContent`); não contornado |

O fato de `Fechar` também não reagir separa “seleção ausente” de “input descartado”: o cliente
não processa mouse/teclado injetado. Não é um problema do canal, da conta ou do PIN.

### 9.3 Evidência local

Os arquivos ficam em `tmp/prova-visual-7662/` (ignorado pelo git): `00-launch.png`,
`03-screen-640.png`, `10-wyd1-elevated-640.png`, `11-connect-elevated-640.png`,
`13-connect-elevated-640.png`, `14-fechar-test-640.png`, `15-enter-test-640.png`,
`17-sendinput-fechar.png`, `canal.log`, `wyd-client.log`, `window-state.ps1` e `resumo.md`.
Nenhum print contém conta, senha ou PIN.

### 9.4 Próximo passo

Repetir A1–A9 com **operador humano** (mouse/teclado físico) usando o checklist do §10. Sem
essa prova, `BUG-01`, `BUG-02` e `GAME-17` permanecem abertas; a matriz acima fica com `B` e o
`GAME-18` como `N/A` nesta sessão.

### 9.5 Diagnóstico de vídeo (`Config.bin`)

O erro de render target **não** era apenas cosmético, mas também não era a causa exclusiva do
input inerte. O `client/Config.bin` tem 30 B = 15 palavras de 16 bits; a palavra no offset 2 é
um índice 1-based para a tabela de modos do binário:

| Índice | Modo | Índice | Modo |
|---:|---|---:|---|
| 1 | 640×480×16 | 6 | 640×480×32 |
| 2 | 800×600×16 | 7 | 800×600×32 |
| 3 | 1024×768×16 | 8 | 1024×768×32 |
| 4 | 1280×1024×16 | 9 | 1280×1024×32 |
| 5 | 1600×1200×16 | 10 | 1600×1200×32 |

- Original: índice **8** (`1024×768×32`). Na tela de 1366×768 a janela ficou `1030×788`, com
  cliente `1024×759`; o canvas de 1024×768 não cabia e o log registrava
  `Error in Init Render Target Texture`.
- Índice **7** (`800×600×32`): janela `806×629`, cliente **`800×600` exato**; o server select
  renderizou, mas o log **continuou** com `Error in Init Render Target Texture`.
- Índice **2** (`800×600×16`): o cliente caiu para `1386×788`/cliente `1380×759` e manteve o
  erro; 16 bits não é um modo utilizável nesta máquina.
- Preferência de GPU NVIDIA (`HKCU\Software\Microsoft\DirectX\UserGpuPreferences`) foi testada
  e removida: o erro e o input continuaram iguais.
- Em `800×600`, `Fechar` também não respondeu a clique elevado. Não é geometria, resolução,
  cor ou GPU: o cliente descarta mouse/teclado injetado. O `Config.bin` foi deixado no índice
  **7** para o operador humano; o backup original está em
  `tmp/prova-visual-7662/Config.bin.bak`.

### 9.6 Evidência do diagnóstico de vídeo

`19-config-mode7.png` (janela `806×629`, cliente `800×600`), `19-config-mode7-640.png`,
`23-mode2.png` (fallback de 16 bits), `24-gpu-nvidia.png` e `25-mode7-fechar.png`; logs em
`client/WYD.log`. Nenhum print contém conta, senha ou PIN.

## 10. Checklist do operador humano (input físico)

> Use o cliente já no modo 7 (`800×600`), contas de teste, nunca a conta do dono. Não registrar
> senha/PIN em print, chat ou documento. Para cada linha, copie `client/WYD.log` e
> `docker compose logs --since 20m server-canal-1` para `tmp/prova-visual-7662/` e preencha a
> matriz do §5; só marque ✅ com print **e** log.

| # | Ação no cliente | Conta de teste | O que observar | Print |
|---:|---|---|---|---|
| 1 | Abrir, selecionar `WYD-1`/canal e clicar `Conectar` | — | server select → tela de login, sem travar | `A0-login.png` |
| 2 | Digitar senha errada e `Enter`; esperar 5 s | `qacampo01` | texto de recusa visível ou ausência dele | `A1-senha-errada.png` |
| 3 | Reabrir, login correto, sem PIN, escolher `QACAMPO01` | `qacampo01` | **sem** “Falha ao conectar”; personagem, NPCs e HUD | `A7-entrada.png` |
| 4 | Reabrir com a conta com PIN, errar o PIN uma vez; depois acertar | `qavivo01` | recusa visível no PIN errado; entrada normal no correto | `A8-pin-errado.png`, `A8-pin.png` |
| 5 | Repetir a entrada e sair/relogar imediatamente | admin de teste | cena, saída e relogin sem timeout | `A9-admin.png`, `A7-relogin.png` |
| 6 | Conta descartável: 5 PINs errados; segunda instância; personagem removido | descartável | aviso de bloqueio, conflito de sessão e seleção inválida | `A3-bloqueio.png`, `A4-conflito.png`, `A5-selecao.png` |
| 7 | Repetir A1–A6 e anotar o opcode/`Size` do último frame | — | `NewAccountFail`/`AccountSecureFail`/motivo no log | `A1.log`…`A6.log` |

**Regra de parada:** se `Fechar` ou `Conectar` não reagirem ao mouse físico, o cliente está
bloqueado por proteção/ambiente; registre `B` e não continue tentando automação. Se a tela
mostrar texto, copie-o exatamente e anexe o print; não traduza nem invente mensagem.
