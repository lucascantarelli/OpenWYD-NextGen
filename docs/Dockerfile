# Imagem do site de documentação — vive em docs/ junto com a config e o conteúdo.
#
# Não instala nada: parte da imagem oficial do Material for MkDocs. O compose
# monta o repositório em /project (o `mkdocs.yml` está em docs/, então todo
# comando usa `-f docs/mkdocs.yml`) e sobe `serve` na 8000.
#
#   docker compose --profile docs up mkdocs        # http://localhost:8000
#   docker compose --profile docs run --rm mkdocs build -f docs/mkdocs.yml --strict
FROM squidfunk/mkdocs-material:latest

WORKDIR /project
