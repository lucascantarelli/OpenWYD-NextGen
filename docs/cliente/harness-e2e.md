# Harness de captura e replay do cliente 7662

> Por que os testes passavam e o cliente continuava quebrado, e qual estrutura passa a valer.

## 1. O buraco: o oráculo era o próprio código

Até 2026-11-13, quase toda prova era **de fio**: o teste monta o pacote com o mesmo encoder do
servidor e verifica o que o servidor escreveu. Isso é tautológico — prova que o servidor é
consistente consigo mesmo, não que o cliente aceita, desenha ou responde.

Exemplos medidos:

| “Provado” | O que o teste realmente dizia | O que o cliente faz |
|---|---|---|
| `0x0117 NewAccountFail` nas recusas | o servidor escreve o opcode | o roteador do 7662 **não trata** `0x0117`; senha errada é silêncio |
| `sessao_unica` | lia 12 B de resposta | a recusa tem `Size=13`; o teste quebrava no decode |
| `campo_vivo` | cena completa no login | cena é por proximidade (raio 24); NPC a 79 tiles não vem |
| `GAME-19` equações | a fórmula reproduz 250 | no canal, personagem com arma causa **1 de dano** |
| `GAME-08` entrada no mundo | entidade/save no Postgres | cliente mostra personagem sem atributos, CP errado, nome vermelho |

Três causas estruturais:

1. **Oráculo circular.** Teste e implementação nascem da mesma interpretação do protocolo.
   Nenhum artefato externo (bytes do cliente real) entra na conta.
2. **Sem captura no repositório.** Toda prova visual foi bloqueada (`CLIENT-06`), e a issue era
   fechada com “resíduo declarado” — o resíduo era justamente o que falhava.
3. **Skip silencioso.** Sem `WYD_TEST_DSN`/`WYD_LIVE_CPSOCK`, o teste “passa” retornando cedo.
   O gate verde não dizia que o caminho real nunca rodou.

## 2. Estrutura em camadas (o que cada uma prova)

| Camada | Artefato | Prova | Status |
|---|---|---|---|
| **L0 — captura** | `tmp/cliente/*.jsonl` (frames C→S e S→C com `Size` e hex) | o que o **cliente real** manda e o que o servidor respondeu | ferramenta pronta; falta a primeira sessão real |
| **L1 — replay** | `tools/cliente-harness/replay.mjs` | o servidor atual responde à sequência real, com o mesmo framing | pronto |
| **L2 — golden de bytes** | capturas versionadas em `tests/golden/cliente/` (pequenas, sem dado sensível) | contrato externo: `Size`, ordem e bytes por opcode | a criar após L0 |
| **L3 — cliente no loop** | VM Windows + automação de input + screenshot diff | render, clique, atributos, CP, nome | bloqueado por input sintético (ver §5) |
| **L4 — regra de aceite** | checklist da issue | só fecha com L0+L1 verdes e print do cliente | a adotar |

## 3. Como capturar (L0)

1. Suba a stack (`make stack-up`).
2. Rode o proxy apontando para o canal:

   ```bash
   node tools/cliente-harness/proxy.mjs \
     --listen 127.0.0.1:8299 --target 127.0.0.1:8281 \
     --out tmp/cliente/sessao-login.jsonl
   ```

3. Gere o `serverlist.bin` do cliente apontando para `127.0.0.1:8299` e abra o 7662.
   Faça **um** fluxo por arquivo: login, criação de personagem, entrada no mundo, clique em NPC.
4. Encerre o proxy (Ctrl+C). O JSONL fica com `init` (INITCODE), frames `C` e `S`.

## 4. Como reproduzir (L1)

```bash
node tools/cliente-harness/replay.mjs \
  --session tmp/cliente/sessao-login.jsonl --target 127.0.0.1:8281
```

Exit `0` = a sequência de respostas (tamanho de cada frame) bate com a da captura.
`--exact` compara bytes — use quando a captura for determinística (o padrão ignora checksum,
keyword e ids que variam).

O que o replay **já** pega: frame faltando/sobrando, `Size` errado, ordem trocada, servidor que
fecha ou muda a cardinalidade. O que ainda **não** pega: opcode e conteúdo, porque o `message_type`
é ofuscado. Próximo passo: decodificar a captura com o `wyd-protocol` (CLI Rust) e comparar
`opcode` + campos, sem reimplementar a `pKeyWord` em JS.

## 5. Cliente no loop (L3): por que ainda não está pronto

- O 7662 é fechado e protegido (`JProtectDll`); `Stop-Process`/`taskkill` sem elevação dá acesso
  negado, e ele não abre conexão CPSock em 25 s de execução (2026-11-13).
- `SendInput`, `mouse_event`, `WM_LBUTTONDOWN` e `SendKeys` elevados já falharam (2026-11-12):
  o cliente usa DirectInput e descarta input sintético sem foco real.
- Caminhos viáveis, em ordem de custo: (a) VM com console/controle real e automação de
  scan code; (b) proxy que injeta pacotes na conexão do cliente (não exige clique, mas exige
  hook); (c) screenshot diff com um humano no loop para os quadros-chave.
- Nenhum desses pode ser declarado pronto sem **print + captura**; enquanto não houver, a prova
  visual continua sendo `QA-12`/`QA-13`/`CLIENT-12`, executada pelo dono.

## 6. Regra de aceite (L4)

Uma issue de cliente só volta a ✅ quando tiver, no mesmo commit/evidência:

1. captura L0 do fluxo (`tmp/cliente/*.jsonl` ou golden versionado);
2. `replay` exit 0 contra o servidor da versão;
3. print do cliente no estado esperado (ou diff de screenshot), com log do canal no mesmo instante.

Sem os três, o status máximo é 🟡 com o resíduo escrito — nunca ✅.

## 7. Verificação da própria ferramenta

```bash
make harness-cliente   # node --test --test-force-exit tools/cliente-harness/
```

Cobre: delimitação de frames (INITCODE + frame partido), recusa de `Size` fora da faixa, gravação
das duas direções pelo proxy e comparação do replay (positiva e negativa).
