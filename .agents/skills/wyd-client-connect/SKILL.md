---
name: wyd-client-connect
description: Liga o cliente Windows 7662 ao servidor Rust local (serverlist.bin, versão de protocolo, CPSock, launcher, portas). Use quando o cliente não listar o servidor, travar no login ou mostrar versão incorreta.
---

# Conectar o cliente ao servidor Rust

Norma: `docs/cliente/depuracao.md` (fluxo de conexão), `docs/cliente/formatos.md`
(`serverlist.bin`/`serv00.htm`) e `docs/arquitetura/protocolo.md` (handshake/checksum).
A cadeia completa e os binários estão nesses docs; não edite `serverlist.bin` à mão.

## Checklist de conexão

- `WYD.exe` → `serverlist.bin` → status `:80/serv00.htm` → TCP `:8281` → INITCODE
  `0x1F11F311` → HEADER 12 bytes → `AccountLogin 0x020D` com `ClientVersion`.
- Build 7662 envia **12000**. Divergência responde `0x102` e o login não abre; ajuste a flag
  do servidor, nunca desligue a checagem.
- `8281/tcp` (jogo) e `80/tcp` (status) precisam ser acessíveis da máquina do cliente.
  `serverlist.bin` é gerado por ferramenta; backup antes de sobrescrever `client/`.
- Checksum: o build 7662 já tem os três sites desativados; a stack Rust usa `Strict`.
  Divergência indica cliente errado, não regra para relaxar.
- `WYD.exe` direto serve para teste; `cargo run -p wyd-launcher -- check` valida
  hash/manifesto/patch.

## Prova

```bash
sha256sum client/WYD.exe   # hash do docs/cliente/inventario.md
cargo test -p wyd-protocol # codec antes de culpar o cliente
curl -s localhost:3000/login | head -3
```
