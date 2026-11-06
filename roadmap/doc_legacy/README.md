# Documentação superada (`doc_legacy`)

Pasta para **documento que envelheceu**: plano executado, desenho abandonado, instrução que hoje daria
passo errado. Nada aqui é lido para operar o servidor. O documento vivo é o substituto apontado na
tabela abaixo.

## Por que existe

Sem um lugar canônico para o que morreu, o doc velho continua parecendo atual: alguém abre
`docs/`, segue a instrução de 2024 e perde uma tarde. Arquivar é mais barato que reescrever história —
e o histórico do git não serve de aviso para quem lê o arquivo.

## Contrato do arquivamento

Ao mover um documento para cá:

1. Mova o arquivo (não copie) e preserve o caminho relativo original dentro de `_legacy/`, para o
   `git log --follow` continuar achando a história.
2. Acrescente uma linha na tabela **abaixo** dizendo o que substituiu e por quê.
3. Não deixe link apontando para o arquivo movido: o gate `scripts/check-docs-links.sh` reprova link
   quebrado, e o mkdocs `--strict` transforma isso em build vermelho.
4. Se o conteúdo ainda tem valor parcial, salve o trecho útil no documento vivo **antes** de mover.

## O que já foi superado

| Documento original | Onde está hoje | Motivo |
|---|---|---|
| `openspec/` (fluxo de spec/proposal) | `docs/archive/openspec/` | o planejamento passou a viver em `roadmap/` (`index.md` é a fonte da verdade) |
| `Source/Code/` como fonte de verdade de gameplay | `Source/Code/` (referência) + `docs/migration/` | a implementação é Go; o C++ é evidência de wire, formato e fórmula |
| `docs/migration/` como descrição do servidor legado | `docs/migration/` (reescrita em Go) | os documentos foram reescritos para descrever o servidor atual; seção "UNVERIFIED" marca o que não foi confirmado |

Nenhum arquivo foi movido para `_legacy/` ainda: as linhas acima registram decisões já tomadas em
outros lugares. Quando um doc for movido de fato, ele entra nesta tabela com o caminho novo.

## Como conferir que nada ficou órfão

```bash
# 1. Nenhum link relativo quebrado dentro de docs/ (não olha para fora de docs/)
bash scripts/check-docs-links.sh

# 2. O site compila com link e nav válidos
docker compose --profile docs run --rm mkdocs build -f docs/mkdocs.yml --strict

# 3. Nenhum documento do repositório ficou fora do site sem decisão registrada
git ls-files '*.md' | sort > /tmp/todos.txt
git ls-files 'docs/**/*.md' | sed 's|^docs/||' | sort > /tmp/site.txt
comm -23 /tmp/todos.txt /tmp/site.txt   # o que NÃO está no site (roadmap, skills, raiz)
```

O passo 3 é diagnóstico, não gate: `roadmap/`, `.agents/skills/` e os `README.md` de raiz ficam fora do
site de propósito. O que ele pega é `docs/*.md` esquecido — arquivo em `docs/` que não aparece no `nav`
do `docs/mkdocs.yml`.

## Regra de remoção

Arquivar não é apagar. Nada é apagado antes de o documento substituto cobrir o conteúdo; a remoção
definitiva é uma decisão explícita, com o motivo escrito na linha da tabela.
