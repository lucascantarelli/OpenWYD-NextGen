# `Source/Code/ClientPatch_v7662` — dependência real e como sair dela

Análise de 06/11/2026. Pergunta: o que ainda depende deste projeto C++ e qual é o caminho para
remover seu uso?

## O que ele é

Um projeto Visual Studio (`ClientPatch_v7662.vcxproj`) que gera a DLL injetada no `WYD.exe`: hooks de
byte, trampolines e os quatro pontos de extensão descritos em
[`../migration/client-patch-hooks.md`](../migration/client-patch-hooks.md).

## Quem referencia

| Referência | Tipo | Dependência real? |
|---|---|---|
| `Source/Cavaleiros de Kersef.sln:24` | entrada de solution | só para abrir no Visual Studio; nada compila isso no CI |
| `docs/migration/client-patch-hooks.md`, `migration-plan.md`, `protocol-spec.md` | documentação de wire | **evidência**: explica por que o servidor manda certo byte |
| `tmserver/internal/protocol/types.go:197`, `checksum.go:13`, `messages.go:517`, `world/world.go:92` | **comentários** | nenhuma — o Go não importa, não compila e não executa o patch |
| `docs/agents/*`, `roadmap/issues/REVIEW-07.md` | retratos históricos | nenhuma |

Verificação que sustenta a tabela:

```bash
grep -rn "ClientPatch" --include=*.go .        # só comentários
grep -rn "ClientPatch" --include=Dockerfile .  # nada
go build ./...                                 # não toca em Source/
```

Ou seja: **o servidor não tem dependência de build nem de runtime** com o ClientPatch. O que existe é
dependência **do cliente**: sem a DLL, o `WYD.exe` original não ganha os hooks (cor do shout, NOP do
checksum, textos).

## Por que ainda não pode ser apagado

Três comportamentos do servidor só fazem sentido por causa do patch, e o comentário no código é a
única prova de que não é bug:

1. **Checksum não rejeitante** (`checksum.go`): o patch dá NOP nos checks do cliente; rejeitar no
   servidor quebraria todo cliente patchado (`world.go`, flag `-reject-checksum` off).
2. **Cor do shout** (`types.go`): o patch pinta `MsgMagicTrumpet` de verde; por isso o shout não reusa
   `MsgMessageChat`.
3. **Corpo do shout com lixo de stack** (`messages.go`): o legado só zerava parte da struct e o patch
   escolhe a cor — o Go zera o corpo de propósito para não reproduzir o lixo.

Apagar o projeto sem registrar isso deixaria três decisões "misteriosas" e convidaria alguém a
"consertar" o checksum.

## Opções e recomendação

| Opção | Efeito | Risco |
|---|---|---|
| **Manter como está** (recomendado até o cliente ser substituído) | evidência disponível; nada entra em build | nenhum de runtime |
| Mover para `Source/Code/_legacy/ClientPatch_v7662/` | mesma evidência, sinaliza "não é fonte viva" | quebra a entrada da `.sln` (arquivo legado, sem build) |
| Remover do repositório | limpeza máxima | perde a prova dos três comportamentos acima |

Recomendação: manter a pasta, **tirar a dependência de leitura** dela. O que dá para fazer hoje, sem
risco:

- já feito antes desta rodada: `Source/Code/_legacy/README.md` classifica o patch como dependência do
  cliente e diz que ele sai só na substituição do cliente (REVIEW-07);
- o que falta: substituir as citações `arquivo.cpp:linha` do patch nos comentários do Go por uma
  referência ao documento (`docs/migration/client-patch-hooks.md`), para que o código não aponte para
  um caminho que pode sair.

## Critério de saída (quando apagar de vez)

O patch só pode sair quando **todos** forem verdadeiros:

1. O cliente usado em produção for um build nosso (REVIEW-07 / issue 52 concluída).
2. `-reject-checksum` estiver ligado por padrão, com captura provando que o cliente manda checksum
   correto.
3. O shout tiver cor definida pelo servidor (ou o cliente novo não precisar do hook).
4. `grep -rn "ClientPatch" --include=*.go` devolver zero — hoje devolve quatro comentários.

Até lá, "remover o legado" aqui significa remover a **dependência de leitura**, não o arquivo.

## O que falta analisar

- Existe alguma forma de remover a dependencia do cliente ? creio que esse hook de dll no cliente é feito via ollydbg, editando os bytes do executável.
- existe alguma forma de, usarmos o launcher que estamos planejando criar para o projeto, com um função de substituir essa dll ?
- esse hook creio que foi documentado em forums da internet, faça pesquisas sobre essa injeção de dll e veja se existem formas de removermso ela e modernizamos a cominucação com o cliente, sem precisar dessa dll.

## Respostas (pesquisa de 06/11/2026)

Base: os 11 patches de byte e 6 trampolines já mapeados em
[`../migration/client-patch-hooks.md`](../migration/client-patch-hooks.md), mais como outros servidores
privados resolvem o mesmo problema hoje.

### 1. O OllyDbg já fez o trabalho — o que falta é o aplicador

Correção importante de premissa: o OllyDbg/x64dbg **não aplica** o patch, ele só serve para *achar* o
endereço. Quem escreve os bytes hoje é a DLL em `DllMain`, com `VirtualProtect` + `memcpy` na `.text`
in-memory (`main.cpp:35-59`). Como os 17 endereços **já estão documentados**, essa fase de engenharia
reversa está encerrada: o que falta é um aplicador determinístico, não mais OllyDbg.

E dá para dispensar a DLL em **todos os 11 patches de byte**, porque são escritas de valor fixo:
`0xEB` no lugar de um `jz`, `NOP`, zero em operando, uma string de título, e uma divisão por 4 numa
tabela. Nada disso precisa de código novo em memória — logo, nada disso precisa de DLL. Basta
converter VA → offset de arquivo:

```
offset = VA - ImageBase - (SectionVirtualAddress - SectionRawPointer)
```

O `PE_Hook.h` já contém a aritmética de seção para o caso em memória; a versão em arquivo é a mesma
conta, com o cabeçalho do PE em vez da base carregada. Só os 6 trampolines (`NKD_*`) exigem código
executável novo — e mesmo eles podem ir para uma *code cave* em vez de uma DLL.

### 2. Sim, e o launcher é o lugar certo — em três modos, com um recomendado

| Modo | Como | Prós | Contras |
|---|---|---|---|
| **A. Patch estático no install** (recomendado) | o launcher reescreve o `WYD.exe` uma vez, guarda o SHA-256 resultante e reaplica quando o binário de origem mudar | sem injeção, sem DLL, sem AV reclamando; o patch viaja junto com o cliente que o launcher já distribui | o arquivo em disco deixa de ser o original; offset só vale para o build 7662 |
| **B. Patch em memória** | `CreateProcess` suspenso → `WriteProcessMemory` → `ResumeThread` | não toca no disco; aplica também shellcode/trampoline | precisa de código nativo; ASLR e antivírus |
| **C. Injetor** | `CreateRemoteThread` + `LoadLibrary` da DLL atual | mantém o código C++ que já existe | continua sendo injeção de DLL — o perfil que antivírus bloqueiam |

O modo **A** é o que os projetos maduros fazem: o `dota2classic/launcher` aplica o patch *uma vez no
build* e publica a `client.dll` já corrigida com hash no manifesto; o `wow-patcher` tem modo estático
como default e só cai para `WriteProcessMemory` quando a seção está criptografada em disco; o
`ebonhold-utils` removeu a dependência de launcher injetando pela *import table* e passou a validar
integridade por hash contra a API do servidor; o KotOR Patch Manager exige `original_bytes` antes de
escrever. Padrão comum: **aplicar uma vez, verificar sempre**.

Para o nosso caso, o guard anti-execução-avulsa (`main.cpp:49-55`, que mata o processo se não vier do
launcher) também deixa de existir no modo A — ele existia só para forçar o caminho do launcher.

Duas obrigações vêm junto: (a) o patch estático precisa **conferir os bytes originais** antes de
escrever e **recusar build desconhecido**, senão um `WYD.exe` de outra versão vira crash silencioso;
(b) o ClientPatch é **GPL-3.0** — reaproveitar código dele mantém a obrigação; um aplicador escrito
do zero, a partir dos offsets documentados, não.

### 3. O que dá para eliminar **pelo servidor**, sem patch nenhum

Esta é a parte que "moderniza a comunicação" de verdade. Dos 17 hooks, vários existem **porque o
servidor legado era assim**, não porque o cliente exige:

| Hook | Dá para eliminar? | Como |
|---|---|---|
| 1-3, checksum `jz→jmp` | **sim, se o servidor emitir checksum correto** | hoje o cliente recusa pacote com checksum inválido e o patch desliga essa checagem. Se o CPSock do Go calcular o checksum como o cliente espera, os três patches somem — e o `-reject-checksum` (hoje `false`, `main.go:104`) passa a poder ser ligado |
| `SendChat` (comandos `@`) | **sim** | comando de chat server-side já é o padrão do projeto (`docs/game.md`) |
| `AddVolatileMessageItem` + `SetVolatileMessageBoxText` (item 3314) | **provavelmente** | diálogo dirigido por mensagem existente do protocolo; exige uma captura para confirmar |
| `ReadMessage` (tipos novos) | **não de graça** | só some se todo pacote novo reusar ID/payload existente — perde expressividade |
| `FixMageMacro` | **não** | é correção de lógica do cliente; continua como patch (de byte, no modo A) |
| XTrap (comentado) | irrelevante | anticheat desligado |

Ou seja: **8 dos 17 hooks podem sumir sem DLL** — 11 patches de byte viram patch estático e 3 de
checksum viram trabalho de protocolo. Sobram, no pior caso, o `ReadMessage` e o `FixMageMacro` como
código injetado.

> **UNVERIFIED:** que o cliente aceite um checksum calculado pelo servidor Go. É hipótese testável:
> enviar um pacote com checksum computado a um cliente **sem** o patch e ver se a sessão sobrevive.
> Enquanto não houver essa captura, o patch 1-3 continua pré-requisito.

### Plano recomendado (ordem de menor risco)

1. **Medir**: rodar o cliente sem patch contra o servidor atual e capturar onde ele desconecta. Se for
   só checksum, o item 2 resolve três hooks de uma vez.
2. **Checksum correto** no CPSock do Go; teste e2e com cliente não patchado. Se passar, remover os
   patches 1-3 do mapa de hooks e ligar `-reject-checksum`.
3. **Aplicador estático** no launcher: offsets documentados + verificação de bytes + SHA-256 no
   manifesto. Cobre SkillDelay, título, força-gráficos, buffs e macro de mago sem DLL.
4. **Só então** decidir se `ReadMessage` vira injeção (modo B) ou se o recurso vai para o servidor.

### Critério para considerar a dependência removida

- `WYD.exe` distribuído pelo launcher já contém os patches de byte (nenhuma DLL em `client/`).
- Nenhum hook de checksum no mapa — o servidor emite checksum válido e `-reject-checksum=true`.
- `Source/Code/ClientPatch_v7662` vira referência histórica (ou sai), sem nada em runtime.

## Fontes

- [`ProjectEbonhold/ebonhold-utils`](https://github.com/ProjectEbonhold/ebonhold-utils) — injeção por import table, gate de integridade por hash.
- [`wowemulation-dev/wow-patcher`](https://github.com/wowemulation-dev/wow-patcher) — patch estático como default, `WriteProcessMemory` quando a seção está criptografada.
- [`dota2classic/launcher`](https://github.com/dota2classic/launcher/blob/master/memory-bank/docs/client-dll-patching.md) — aplicar uma vez no build e distribuir o artefato com hash.
- [`DvHyper/ConquerLoader-Launcher`](https://github.com/DvHyper/ConquerLoader-Launcher) — injeção clássica `CreateRemoteThread`/`LoadLibrary` (o que o modo C reproduz).
- [`LaneDibello/Kotor-Patch-Manager`](https://github.com/LaneDibello/Kotor-Patch-Manager) — `original_bytes` obrigatório antes de patch estático.