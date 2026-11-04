# Issue 05 — Configurar o cliente modificado para o servidor local

**Fase:** 1 · **Depende de:** 04 · **Bloqueia:** 06, 07 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

O cliente `client/` lê um `serverlist.bin` que aponta para **este** servidor e mostra o canal
disponível, sem o launcher sobrescrever a configuração.

## Contexto / evidência

- `client/serverlist.bin` (7.044 B) contém canais do servidor remoto (WYD Global/RaidHut) em
  formato binário ofuscado pela tabela estática do CPSock — não é editável como texto.
- `client/.remote_serverlist.bin` (1.245 B, 01/11 04:30) é **uma página HTML de erro**,
  resíduo de tentativa de update do launcher. Isso prova que `Updater.exe`/launcher escreve na pasta
  e pode sobrescrever o `serverlist.bin`.
- `client/config.json` aponta o launcher para
  `https://d16tndoz98nwcc.cloudfront.net/info.json`.
- Ferramenta oficial no repo: `serverlist bin/serverlist editor.exe`; existem bins prontos por
  IP em `serverlist bin/<IP>/serverlist.bin` (referência de formato, IPs privados alheios).

## Passos

1. Descobrir o IP que o cliente vai discar. Se o cliente roda **na mesma máquina** do servidor, pode
   ser `127.0.0.1` (mas o container publica em todas as interfaces do host, então também funciona o
   IP da LAN). Se roda em outra máquina, use o IP da LAN (`ipconfig`).
2. Backup obrigatório:
   ```bash
   cd client
   cp serverlist.bin serverlist.bin.bak
   ```
3. Rodar `serverlist bin/serverlist editor.exe`, configurar `<IP>:8281` (grupo 0, índice 0)
   e exportar `serverlist.bin`.
4. Copiar o arquivo gerado para `client/serverlist.bin`.
5. Garantir que **o launcher não roda**: abrir `WYD.exe` diretamente (atalho próprio). Se o usuário
   insistir no launcher, documentar que ele pode reverter o `serverlist.bin`.
6. Liberar firewall do host para `8281/tcp` e `80/tcp` (o cliente usa as duas).
7. Confirmar do lado do servidor: os logs devem mostrar requisição HTTP de status vinda do IP do
   cliente (`served status (http) ip=…`) e, em seguida, o pacote `0x020d`.

## Arquivos

- Alterado (fora do git, na pasta do cliente): `serverlist.bin` (+ `serverlist.bin.bak`).
- Criado (opcional, versionável): `docs/local/serverlist-local.md` registrando IP usado,
  data e como regenerar — sem versionar o binário gerado.
- Proibido tocar: `WYD.exe`, DLLs, `.sys`, `config.json` do launcher (mudança de URL do launcher
  não é necessária para conectar direto).

## Testes / verificação

```bash
# 1. o status responde do ponto de vista do host do servidor
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1/
# 2. a porta de jogo está aberta
powershell -NoProfile -Command "Test-NetConnection <IP> -Port 8281 | Select TcpTestSucceeded"
# 3. o cliente foi aberto e bateu no status
docker compose logs --since 5m tmserver | grep "served status"
# 4. o cliente abriu o CPSock
docker compose logs --since 5m tmserver | grep -E "recv packet|client connected"
```

Saída esperada: requisição de status do IP do cliente e, na sequência, `recv packet … type=0x020d`.

## Critérios de aceite

- [ ] `serverlist.bin` substituído com backup preservado
- [ ] Cliente lista o canal local (sem "manutenção"/sem canal vazio)
- [ ] Log do tmServer registra o GET de status do IP do cliente
- [ ] Log do tmServer registra conexão CPSock e o pacote `0x020d`
- [ ] Procedimento registrado (IP, data, ferramenta) para repetir em outra máquina

## Riscos / rollback

- **Risco:** launcher sobrescreve o bin. **Mitigação:** não abrir launcher; se abrir, regerar.
- **Risco:** firewall/NAT entre máquinas — status funciona e jogo não (ou vice-versa). **Mitigação:**
  testar as duas portas separadamente a partir da máquina do cliente.
- **Risco:** `serverlist editor.exe` gerar canal com porta errada (cliente sempre disca 8281 pelo
  binário; o campo de porta pode não ser respeitado) → manter 8281 e mudar a porta publicada do host.
- **Rollback:** restaurar `serverlist.bin.bak` (o cliente volta ao estado anterior, remoto).
