# syntax=docker/dockerfile:1
# Build de produção em dois estágios (guidelines §4.8). Parametrizado por serviço:
#
#   docker build --build-arg SVC=tmserver -t openwyd-nextgen1-tmsrv .
#
# SVC é um de {tmserver, dbserver, binserver, webserver}; cada um tem
# cmd/<SVC>/main.go. O `image:`/`container_name` (compose) dá o nome final, então
# o mesmo Dockerfile serve para os dois canais (`openwyd-nextgen1-tmsrv` e
# `openwyd-nextgen2-tmsrv` são o MESMO binário com -server-index diferente).
FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git

# Dependências primeiro: o cache de camadas só é invalidado quando go.mod/go.sum
# mudam, não a cada alteração de código.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG SVC
# Falha cedo e com mensagem clara quando o build esqueceu o --build-arg.
RUN test -n "$SVC" || (echo "o build-arg SVC é obrigatório (tmserver|dbserver|binserver|webserver)" && false)
# Binário estático (CGO off) para rodar na imagem distroless; o cache de build
# evita recompilar o mundo inteiro em cada `docker compose build`.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./${SVC}/cmd/${SVC}

# Distroless static: mínimo, com CA certs, roda como nonroot (UID 65532). Sem
# shell de propósito — o healthcheck é `CMD ["/app","healthcheck"]`.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app

# Conteúdo do jogo embutido: o tmServer (W2PP_CONTENT=/content) lê taxas,
# catálogos e templates de BaseMob no boot, e o webserver varre os templates de
# mercador + ItemList.csv da UI de moderação. Sem ele o login de personagem cai
# num CNFCharacterLogin sem template, que o cliente real recusa (crash ao entrar
# no mundo). dbserver/binserver ignoram o diretório. Os artefatos legados
# pesados ficam de fora pelo .dockerignore.
COPY --from=build /src/tmserver/content /content

# O gerenciador de storage atribui nomes opacos, então o manifesto publicado
# validado é o mapa imutável de URLs consumido pelo webserver. Fixar o digest faz
# o build de produção falhar em vez de aceitar dados trocados em silêncio.
ADD --chown=65532:65532 --chmod=0444 --checksum=sha256:e6e10d3bac61792d9bae048715ff9bf5f85168c1fe18ec43c5dc900e4d57a4da https://jeanluca-teste.s3.amazonaws.com/602dce1d-bdf1-4ce6-a329-7e1fa630bec6file /published-item-icons-manifest.json
ENV W2PP_ITEM_ICONS_MANIFEST=/published-item-icons-manifest.json
ENV W2PP_CONTENT=/content

ENTRYPOINT ["/app"]
