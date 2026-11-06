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
