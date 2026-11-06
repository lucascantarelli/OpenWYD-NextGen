# Personagem e evolução

Como um personagem nasce, evolui e é salvo. O servidor é a autoridade: nível, EXP e atributos vivem
na memória do mundo e chegam ao cliente por pacote — o cliente não calcula nada disso.

## Fluxo

1. **Criação**: o cliente envia nome e classe; `character.go` valida e monta o `STRUCT_MOB` (816 bytes)
   a partir do template da classe em `tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}` — equipamento inicial
   vem daí, não de código.
2. **Seleção**: `protocol/selchar.go` monta o `STRUCT_SELCHAR` (840 bytes) com a prévia de atributos
   (nível, HP/MP, STR/INT/DEX/CON) que a tela de seleção mostra antes de entrar.
3. **Entrada no mundo**: `login.go` valida conta e versão de protocolo; a sessão passa a existir no
   mundo e o personagem é materializado na cidade de origem.
4. **Evolução**: ganho de EXP em `mobkilled.go` / `exp_bonus.go`, curva e distribuição em
   `docs/migration/game-rules.md` §1; subir de nível recalcula atributos em `score_derive.go` e
   `arch_progression.go` (arquétipo).
5. **Recursos**: HP/MP corrente em `hpmp.go`; peso e espaço em `cargo.go` / `carry.go`.
6. **Save**: ao deslogar, `internal/savefmt` serializa por offset explícito (alinhamento MSVC x86 —
   `docs/migration/data-formats.md`). Posição exata **não** é persistida por design.

## Onde está

| Peça | Arquivo |
|---|---|
| Criação e login | `character.go`, `login.go` |
| Prévia na seleção | `protocol/selchar.go` |
| Curva de EXP e party | `docs/migration/game-rules.md` §1 |
| Atributos derivados | `score_derive.go`, `arch_progression.go`, `hpmp.go` |
| Capacidade e peso | `cargo.go`, `carry.go` |
| Save binário | `internal/savefmt` + `docs/migration/data-formats.md` |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Character|Score|Arch|Cargo|Carry'
```

No jogo: criar personagem, matar um mob e conferir que o EXP aparece **no painel de EXP** (barra
numérica), não como fala no chat.

!!! warning "UNVERIFIED"
    A tabela de EXP por nível usada pelo cliente para desenhar a barra não foi extraída para uma
    tabela consultável; o servidor calcula o ganho (§1) e o cliente só exibe. Se a barra divergir do
    valor real, o suspeito é o pacote de painel, não a curva.

## O que este sistema não faz

- Não persiste posição exata (nasce no spawn da última cidade).
- Não deixa o cliente decidir atributo: `STRUCT_SELCHAR` é montado no servidor.
