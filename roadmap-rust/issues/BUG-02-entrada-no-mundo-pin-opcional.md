# BUG-02 — “Falha ao conectar” ao entrar no mundo: PIN opcional e seleção antes do PIN

**Tipo:** Bug · **Esforço:** S · **Depende de:** FASE-03-17 ✅ · **Bloqueia:** onda 1 (jogável),
FASE-03-18 (Campo de Treinamento), FASE-15-4 (bots)

## Objetivo

O jogador consegue fazer login e criar personagem, mas ao escolher o personagem o cliente 7662
mostra **“Falha ao conectar”**. Fechar essa porta sem inventar pacote: o canal precisa aceitar a
sequência que o cliente realmente manda na entrada do mundo.

## Contexto / evidência (log do canal real, 2026-11-11)

O cliente não repete a senha numérica ao entrar no canal: depois do `_MSG_AccountLogin` ele manda
**`_MSG_CharacterLogin` direto**. O canal exigia `_MSG_AccountSecure` antes de qualquer pacote de
personagem e recusava a conexão:

```
level=warn event=pin result=refused account=qavivo01 reason=missing_pin
level=warn event=handshake result=rejected reason=senha numérica: seleção antes da senha numérica
level=warn event=handshake result=rejected reason=io: failed to fill whole buffer
```

O `io: failed to fill whole buffer` é consequência: o cliente fecha a conexão ao receber a recusa
sem `CNF`. A senha da conta **já foi conferida** no `_MSG_AccountLogin` (argon2id) antes desse
ponto; a cerca de sessão e o `character ok action=select` também já funcionavam. O que faltava era
não tratar o PIN como pré-requisito obrigatório da seleção.

## Correção

1. `Authenticator::pin_registered` (`crates/wyd-server/src/auth.rs`): diz se a conta tem PIN
   cadastrado (`pin_hash`). Conta sem PIN não tem etapa a cumprir.
2. `authorize_pin` (`cpsock.rs`): 
   - conta **sem** PIN → abre a sessão e vai direto para a seleção
     (`pin result=skip reason=not_registered`);
   - pacote de seleção (`CreateCharacter`/`DeleteCharacter`/`CharacterLogin`) antes do PIN →
     abre a sessão e **devolve o frame já decodificado** para o laço de seleção
     (`pin result=deferred reason=selection_first`), em vez de recusar;
   - `_MSG_AccountSecure` continua validando o PIN de quem o envia.
3. `serve_selection` ganhou o parâmetro `pendente: Option<Pendente>`: processa o pacote adiantado
   antes de esperar o próximo, com checksum já verificado.
4. `dispatch` trata `_MSG_AccountSecure` recebido depois da seleção como confirmação
   (`pin result=ok mode=late`), sem abrir sessão nova — cliente que manda o PIN continua servido.

Invariante preservada: a senha é verificada antes de qualquer sessão; PIN errado (quando
informado) continua recusando com `_MSG_AccountSecureFail`. Nada de pacote novo foi inventado.

## Arquivos

- Alterados: `crates/wyd-server/src/auth.rs`, `crates/wyd-server/src/cpsock.rs`.
- Testes: `cargo test -p wyd-server --lib` (43/43) e prova viva `crates/wyd-server/tests/cena_mobs.rs`
  com `WYD_LIVE_SKIP_PIN=1` (sequência sem PIN, igual ao cliente).

## Testes / verificação

```bash
cargo fmt --all --check
cargo clippy -p wyd-server --all-targets -- -D warnings
cargo test -p wyd-server --lib
WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=qavivo01 WYD_LIVE_PASSWORD=... \
  WYD_LIVE_SKIP_PIN=1 cargo test -p wyd-server --test cena_mobs -- --nocapture
```

Saída esperada: `pin result=deferred`/`skip` no log do canal, `character ok action=select`,
rajada de `CreateMob` e diálogo de NPC respondendo.

## Subtasks

- [x] Reproduzir a recusa no log do canal vivo (`seleção antes da senha numérica`)
- [x] PIN opcional quando a conta não tem PIN; seleção antes do PIN devolve o frame
- [x] `_MSG_AccountSecure` tardio confirmado sem abrir segunda sessão
- [x] `clippy -D warnings` 0, `wyd-server` 43/43, teste vivo com PIN ok
- [ ] Prova viva sem PIN (`WYD_LIVE_SKIP_PIN=1`)
- [ ] Prova visual no cliente 7662 (entrar no mundo com o personagem)
- [ ] Versionar, mergear e fechar

## Critérios de aceite

- [x] Conta sem PIN entra sem receber `AccountSecureFail`.
- [x] Seleção antes do PIN não encerra a conexão; o personagem é selecionado.
- [x] PIN informado continua sendo verificado (recusa com `PinIncorreto`/`PinBloqueado`).
- [ ] Cliente 7662 entra no mundo sem “Falha ao conectar”.

## Riscos / rollback

- Risco: PIN deixa de proteger a entrada → mitigação: a senha argon2id continua obrigatória antes;
  o PIN segue verificado quando enviado, e a decisão fica registrada em `docs/wyd/senha-numerica.md`.
- Rollback: voltar a recusar seleção antes do PIN em `authorize_pin` (ponto único).
