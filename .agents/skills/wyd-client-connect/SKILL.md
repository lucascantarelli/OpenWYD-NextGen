---
name: wyd-client-connect
description: Liga o cliente Windows 7662 ao servidor Rust local (serverlist.bin, versão de protocolo, CPSock, launcher, portas). Use quando o cliente não listar o servidor, travar no login ou mostrar versão incorreta.
---

# Conectar o cliente ao servidor Rust

> Sucessão 09-2: mantido o nome; servidor alvo migrado para Rust. Fonte canônica: `docs/cliente/`, `docs/arquitetura/protocolo.md`, `crates/wyd-launcher/`.

## Regras

1. **Handshake real.** `WYD.exe` → `serverlist.bin` → `GET http://IP:80/serv00.htm` → TCP `IP:8281` → `INITCODE 0x1F11F311` → `HEADER` 12 bytes → `AccountLogin 0x020D` com `ClientVersion`.
2. **Versão do protocolo.** Build 7662 envia **12000** (medido); divergência responde `0x102` e o login nem abre. Ajuste a flag do servidor, nunca desligue a checagem.
3. **Portas.** `8281/tcp` (jogo) e `80/tcp` (status) acessíveis da máquina do cliente; `serverlist.bin` é binário ofuscado — gere com ferramenta, nunca edite à mão; backup antes de sobrescrever `client/`.
4. **Launcher.** `WYD.exe` direto para teste; `wyd-launcher` (`cargo run -p wyd-launcher -- check`) valida hash/patch/manifesto (`docs/cliente/customizacao.md`).
5. **Checksum.** Servidor Rust usa política `Strict` em produção (erro tipado rejeita); build 7662 tem os 3 sites de checksum desativados (`EB` no disco) — divergência indica cliente errado, não regra para relaxar.

## Diagnóstico

```bash
sha256sum client/WYD.exe   # 04c9d2a1…27ddf8 esperado (docs/cliente/inventario.md)
cargo test -p wyd-protocol  # codec byte-exato antes de culpar o cliente
curl -s localhost:3000/login | head -3   # portal no ar (stack em docs/rust/05-stack-docker.md)
```

## Higiene

- Nunca versionar `serverlist.bin` com IP privado, dumps/logs; nunca redistribuir binários; `client/` fora do git.
