# NORMALIZE-02 — Busca do portal compara Unicode cru (NFD não acha NFC)

**Fase:** revisão · **Depende de:** — · **Esforço:** S · **Estado:** ✅ concluída (05/11/2026)

## Objetivo

O picker de item achar o mesmo item independentemente da forma Unicode em que o termo chega —
`Poção` em NFC (banco) e em NFD (macOS, alguns IMEs de teclado) são a **mesma** palavra para quem
digita.

## Contexto/evidência (05/11/2026)

O catálogo guarda os nomes em NFC; medido no Postgres:

```sql
select name, encode(convert_to(name,'UTF8'),'hex') from item_catalog where item_index=3311;
-- Poção_Combatente | 506fc3a7c3a36f5f436f6d626174656e7465   (c3 a7 = ç NFC)
```

A rota `portal/src/app/api/search/items/route.ts` fazia `item.name.toLowerCase().includes(query)`
sem normalizar. Resultado no portal em execução:

```
?q=po%C3%A7%C3%A3o            (NFC) -> 20 itens
?q=po%63%CC%A7a%CC%83o       (NFD) -> 0 itens     <- mesmo termo, outra forma
```

Quem digita "poção" num teclado que entrega NFD vê o picker vazio e conclui que o item não existe.

## O que foi feito

- `normalizeSearch()` em `portal/src/lib/itemsearch.ts`: `value.normalize("NFC").toLowerCase()`.
- A rota de busca normaliza a consulta e os dois campos comparados (`name`, `display_name`).
- Teste `portal/src/lib/itemsearch.test.ts` cobrindo NFD↔NFC, a busca no nome real do catálogo e a
  garantia de que "Pocao" continua diferente de "Poção" (não virou busca sem acento).

## Verificação (falha antes, passa depois)

```
sem .normalize("NFC"):   Tests  2 failed | 1 passed (3)
   AssertionError: expected 'poção' to be 'poção'
   AssertionError: expected 'poção_combatente' to contain 'poção'
com a correção:          Tests  3 passed (3)
```

HTTP na imagem reconstruída:

```
?q=po%C3%A7%C3%A3o (NFC)  -> 20
?q=po%63%CC%A7a%CC%83o (NFD) -> 20
```

## Critérios de aceite

- [x] Termo em NFD e em NFC devolvem o mesmo conjunto.
- [x] Teste que falha sem a normalização e passa com ela.
- [x] `npx tsc --noEmit` limpo.
- [ ] Mesma normalização nos filtros de cliente que comparam texto (`DataTable`, `Combobox`) — hoje
      normalizam só `toLowerCase`; nenhum caso reportado, fica registrado como follow-up.

## Riscos/rollback

Risco: normalizar em toda busca custa uma passada de string por item (3.220 itens) — desprezível
frente ao RPC do catálogo. Rollback: remover a chamada em `normalizeSearch`, voltando ao
`toLowerCase` cru.
