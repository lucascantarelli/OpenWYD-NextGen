# Documento de bugs encontrados

> Issues canônicas deste documento: `BUG-01` (mensagens, entrada no mundo, personagens,
> atributos, NPCs de loja/banco, CP e andar/correr) e `BUG-02` (PIN/segunda senha).
> O detalhe executável e os critérios de aceite ficam nos arquivos das issues.

| Issue | Estado | Evidência atual | O que falta |
|---|---|---|---|
| `BUG-01` | ⬜ | Recusas com motivo no log, testes de fio, `ClientID`/`UpdateScore` e cena | Matriz A1–A6 no 7662; cada recusa com print/log |
| `BUG-02` | 🟢 | PIN opcional, seleção antes do PIN e testes de fio | Prova viva sem PIN (`WYD_LIVE_SKIP_PIN=1`) e entrada no 7662 |

> **Tentativa 2026-11-12:** a Sessão A chegou ao server select, mas o cliente 7662 descartou
> mouse/teclado sintéticos; nenhum login saiu do cliente e nenhum critério visual foi marcado.
> O `Config.bin` foi levado do modo 8 (`1024×768×32`, cliente `1024×759`) para o modo 7
> (`800×600×32`, cliente `800×600` exato); o erro de render target e o input inerte persistiram.
> Detalhe no runbook §9.5–§10; os dois bugs seguem abertos, com checklist humano pendente.

Arquivos: [`BUG-01`](issues/BUG-01-mensagens-char-mundo.md) e
[`BUG-02`](issues/BUG-02-entrada-no-mundo-pin-opcional.md).

## Mensagem de erro no login dentro do cliente

**Issue:** `BUG-01`

- ao executar o login, com dados incorretos no cliente, dados de usuário e senha, o sistema não exibe uma mensagem de erro clara, apenas retorna para a tela de login sem qualquer feedback.
- aparece a mensagem conectando no servidor depois some e nada acontece.

## Segunda senha dentro do cliente

**Issue:** `BUG-02`

- a segunda senha não está sendo validada corretamente, mesmo com dados incorretos o sistema permite o acesso, o que representa um risco de segurança.

## Apagar personagem dentro do cliente

**Issue:** `BUG-01`

- Ao apagar personagens, o jogo pede para incluir a senha, essa senha não está sendo validada, aceita qualquer senha, mesmo que esteja incorreta, permitindo apagar personagens sem a devida autenticação.

## Nome do char vermelho dentro do cliente

**Issue:** `BUG-01`

- ao criar um novo personagem, o nome do char aparece em vermelho, referente ao sistema de CP
- comando /cp não funciona, quando deveria mostrar as informações do CP do char

## Pontos de atributos incorretos dentro do cliente 

**Issue:** `BUG-01`

- os pontos de atributos FOR/DES/INT/CONS estão zerados, mesmo que o personagem tenha pontos distribuídos, o que impede a visualização correta dos atributos do personagem.
- os pontos de imunidades estão 245, creio que deveriam estar zerados, pois o personagem não possui imunidades.

## Nivel inicial do personagem

**Issue:** `BUG-01`

- O personagem está iniciando level 2, creio que deveria iniciar no level 1, o que pode gerar problemas de balanceamento do jogo.

## Npcs de loja/banco

**Issue:** `BUG-01`

- os npcs de loja não estão funcionando corretamente, não é possível comprar ou vender itens, o que impede a progressão do personagem e a economia do jogo.
- o npc de bancoo não permite guardar nenhum item ou gold do banco, mesmo com o personagem com gold suficiente, o que impede a segurança do gold do personagem.

## Ativar modo andar/correr do personagem

**Issue:** `BUG-01`

- o personagem não consegue alternar entre os modos de andar e correr, não sei o atalho relacionado, mas o personagem não consegue alternar entre os modos, o que pode gerar problemas de movimentação e exploração do mapa.
