# Campo de Treinamento — pesquisa 2026-11-11 (guardas, recompensas e sequências)

> Complemento de `pesquisa-campo-de-treinamento-2026-11-10.md`. Fontes: guia oficial PT
> (<https://wydglobal.raidhut.com/pt-br/2402>), guia oficial EN
> (<https://wydglobal.raidhut.com/en/2403>), FAQ oficial “Campo de Treinamento (Lv 1~35)”
> (<https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/20369-campo-de-treinamento-lv-135>),
> guia oficial antigo “Mortal Quests (1~355)”
> (<https://feedback.kersef.com/en/knowledge-bases/21/articles/20037-mortal-quests-1-355>),
> droplist oficial (<https://droplist.raidhut.com>), privado WYD Brazuca
> (<https://wyd.net.br/guias-do-wyd/66-FAQ-Iniciantes-no-WYD-Brazuca>) e WYD Winter
> (<https://wydwinter.wixsite.com/game/drop-list>).

## 1. NPCs e coordenadas (oficial)

| NPC | Coord. | Função |
|---|---|---|
| Unicorn | 2130,2040 | arma inicial por classe, de graça |
| CabuncleWind | 2131,2033 | buff ao clicar; **três instâncias** (uma por área, após cada portão) |
| FirstTrainer | 2077,2017 | 1ª etapa |
| SecondTrainer | 2144,1989 | 2ª etapa |
| ThirdTrainer | 2082,1966 | 3ª etapa |
| TrainerChief | 2122,2041 | Emblema Orc → recompensa final |

Portões: 2075,2016 · 2143,1986 · 2081,1962.

## 2. Guardiões e boss — stats e drops (droplist oficial)

| Monstro | Nv | HP | MP | EXP | Drops (faixa oficial) |
|---|---:|---:|---:|---:|---|
| GremlinDecoy | 30 | 600 | 150 | 120 | Chave da 1ª Porta (Muito Alta); Bastão Serpente, Matadora de Dragão, Maça, Manoplas Douradas (Alta) |
| GiantCrill | 50 | 2.000 | 150 | 140 | Rubi do Carbúnculo (Muito Alta) + Chave da 2ª Porta (Muito Alta) |
| OrcSniper | 50 | 2.500 | 0 | 160 | Chave da Última Porta (Muito Alta); Ovo de Javali (Muito Alta); Resto de Oriharucon (Baixíssima) |
| OrcWarrior | 40 | 2.000 | 0 | 240 | Âmago de Javali (Muito Alta); **Emblema Orc** (Muito Alta) |

Spawn real difere do ponto citado no guia: GremlinDecoy (2082,2028), GiantCrill (2147,2005),
OrcSniper (2079,1976), OrcWarrior (2138,1941).

**Emblema Orc (descrição oficial do item):** “Entregue o emblema ao Trainer Chief para receber
recompensas. Recompensa (Aleatória): Rubi de Carbúnculo, Olho crescente, Poção de Cura (M)”. O
guia PT diz só “Olho Crescente (3.000 EXP)” — divergência real entre versões.

## 3. Sequências de recompensa encontradas

| ID | Origem | 1ª | 2ª | 3ª | Final (Chefe) |
|---|---|---|---|---|---|
| **A** | Legado OpenWYD (código) | item 682 (20) + affect 44 | encanta `Equip[6]` | encanta `Equip[1..7]` + sanc 3 | sorteio 682 / 481 / 652 |
| **B** | Oficial atual (PT) | 20 Poção de Cura (M) | adicionais aleatórios nas armas | refino até +6 | Olho Crescente (3.000 EXP) |
| **C** | Oficial EN antigo + DB | 20 poções | adicional aleatório na arma | refino +0~+2 (teto +6) | 20 pot + Rubi do Carbúnculo + Olho Crescente |
| **D** | Privado (WYD Brazuca) | **Rei dos Nbs** vende **Arma Sephira +9** | — | — | — |

Regras transversais oficiais: só nível **abaixo de 35** (FAQ: 1~35); **cada NPC recompensa uma
vez por personagem**; com duas armas, o bônus vai só na arma à esquerda; o adicional do 2º
treinador pode sair pior que o atual; o 3º não faz nada em item não refinável.

## 4. Buff do Cabuncle/Carbunkle Wind

- Sem custo; **pode ser reclicado quando o buff acaba**.
- Tipos relatados: velocidade/defesa/ataque/técnica; 1ª instância defensiva, 3ª ofensiva.
- Duração: nenhuma fonte oficial publica número; relato de época diz “~3 minutos”.

## 5. Portões e chaves

- A chave **é consumida no instante do uso** e o portão **fecha depois de algum tempo** (sem
  número oficial).
- Perdeu a chave? O guardião **volta a dropar** enquanto a quest está ativa.
- Chance oficial é faixa “Muito Alta”, não 100%.

## 6. Boss

OrcWarrior fica na 4ª área, atrás do portão final, e aparece **depois do Arqueiro Orc**; é
preciso esperar o spawn. Depois do Emblema, o jogador volta ao início (Esc → Personagem) e fala
com o TrainerChief.

## 7. Decisões do dono (2026-11-11)

| # | Decisão |
|---|---|
| D-7 | Sequência **A — legado OpenWYD** (682+affect 44 → encanta `Equip[6]` → encanta `Equip[1..7]`+sanc 3 → sorteio 682/481/652) |
| D-8 | **NPC de armas Sephira +9 entra no Campo** (dano full, refinada +9), ao lado do Unicorn oficial. Reverte o “sem Sephira” de D-6 |
| D-1 | Faixa **1 a 35** (nível 36 recusa) |
| D-9 | TrainerChief entrega **um item sorteado** entre Rubi do Carbúnculo, Olho Crescente e 20 Poções |

## 8. O que este documento **não** fecha

1. Segundos exatos do fechamento do portão (nenhuma fonte publica).
2. Percentual numérico de drop (oficial só publica faixas).
3. Duração oficial do buff.
4. Lista completa de armas do Unicorn (a página de mercador do droplist não carrega itens).
5. ~~Templates `STRUCT_MOB` de NPCs/monstros do campo~~ — **resolvido**: não vêm do cliente.
   Ver §9.

## 9. Origem dos blobs `STRUCT_MOB` e seed do campo (2026-11-11)

O cliente 7662 **não embarca tabela de NPC/monstro**: `docs/cliente/tabelas-bin.md` §61 registra
que nenhum `.bin` do cliente tem tabela por monstro (o cliente só recebe `CreateMob` do servidor),
e `client/WYD.exe` não contém nenhum dos nomes do campo (`FirstTrainer`, `GremlinDecoy`,
`OrcWarrior`, `TrainerChief` → 0 ocorrências). Portanto **não há extração possível a partir do
cliente**: `STRUCT_MOB` é dado de servidor.

Os blobs reais já estão no banco, vindos do dump legado (`0020_seed_mundo.sql`,
`seed_npc_template.up.sql`: 1991 templates). Os dez do campo, todos com 816 B:

| Template | sha256 (12 primeiros) |
|---|---|
| `FirstTrainer` / `SecondTrainer` / `ThirdTrainer` | `8ce700d4bfb1` / `0aff87b93f69` / `bb978c79ec76` |
| `TrainerChief` / `Unicorn` / `Cabuncle` | `50dc8749b639` / `8a9b1b12b1f7` / `06a379ccfa9b` |
| `GremlinDecoy` / `GiantCrill` | `8031792240ef` / `5611531fe7eb` |
| `OrcSniper` / `OrcWarrior` | `495cf62ba6c8` / `2377bd2b7ee5` |

O seed `crates/wyd-db/migrations/0046_seed_campo_treinamento.sql` posiciona esses templates no
mapa e registra os drops 451/452/453/524, sem criar blob nem tabela de conteúdo nova.

## 10. Estado vivo do Campo (2026-11-11)

- **Sem duplicatas:** a `0047_seed_limpeza_fixtures_campo.sql` removeu as fixtures `teste-campo-*`
  (6201–6208) e os drops `fixture-campo-teste` (o `down` as restaura). Cada NPC do Campo tem
  uma única instância na cena, verificada por coordenada no harness vivo.
- **Aceite da quest:** o legado só começa a contar o `Newbie` quando o jogador fala com o
  primeiro treinador. O servidor passou a gravar `character_quest` (`step=1, progress=0`) no
  primeiro contato e a responder `Campo de Treinamento: etapa 1/4 (aceita)`; sem isso o
  primeiro clique de um personagem recém-criado era recusado por falta da prova 451.
- **Prova no servidor rodando:** personagem novo criado pelo `_MSG_CreateCharacter`, entrada
  no mundo e aceite lidos do banco (`character_quest`), com o segundo clique recusando por
  falta da 451 sem alterar o estado.
## 11. Teste manual do dono (2026-11-11)

- **Corrigido — 5 pontos no nível 1:** `starting_scores` somava um nível a mais; agora os pontos
  são só por nível subido (0 no recém-criado).
- **Corrigido — nome vermelho (causa real, 2026-11-11):** `ChaosRate@15` **não** é stat de
  combate: é o **CP** (−75..+75) codificado em `+75`. O cliente lê `pMob+0x11e` em
  `WYD.exe 0x4e2393`, limita a `10..150` e interpola a tabela `0x6188a4` (vermelho → laranja →
  amarelo → verde → azul → branco); abaixo de 10 cai no ramo de criminoso. O próprio cliente
  inicializa mob com `0x4B` (75 = CP 0) em `0x4d9ae2`. O servidor mandava **0** em
  `BaseScore`/`CurrentScore` e no `CreateMob` — por isso o nome vermelho. Agora os três usam
  `CP_NEUTRO = 0x4B` (`crates/wyd-server/src/personagens.rs`), com teste
  `personagem_nasce_com_cp_neutro_e_nao_com_o_lixo_do_template`.
- **Resolvido — gold inicial:** a regra é o **`BaseMob` da classe**: `Coin@28` é **5 000 000**
  nas quatro classes (`assets/basemob/{TK,FM,BM,HT}`, provado por
  `templates_de_classe_tem_816_bytes_e_campos_nos_offsets_do_c`). O banco já recebia esse valor
  na criação; o que faltava era o **pacote**: o login enviava `_MSG_UpdateScore` e não o
  `_MSG_UpdateEtc`, que carrega `Coin@28` e desenha o ouro. Agora o par sai junto do
  `CNFCharacterLogin` (`cpsock`), e `tests/gold_inicial.rs` falha se o cliente entrar sem ele.
- **Resolvido — NPCs na tela:** causa raiz era a **entrega**, não a varredura: 6 110
  `_MSG_CreateMob` de 232 B (~1,4 MB) no login não são desenhados pelo cliente 7662. A sessão
  agora recebe a cena por proximidade (`world::scene_around`, raio de 24 tiles,
  `WYD_SCENE_VIEW_TILES`) e o delta é recalculado a cada poll (andar troca o conjunto mesmo
  sem mudar a geração). Prova: `tests/cena_armia.rs` → `33 mobs`, `32 NPCs` em volta do spawn,
  ids 1080..6941; unitário `cena_por_proximidade_corta_o_que_esta_longe`.
- **Aberto — distribuir pontos:** `_MSG_UpdateScore` só existe J→C.
- **Aberto — baú de XP:** `_MSG_UseItem` cobre refino/pó; sem affect/efeito visual.
- **Aberto — lojas de NPC:** `npc_shop_item` já tem 6 706 linhas / 718 itens no snapshot; faltam
  `_MSG_REQShopList` (0x027B) e `_MSG_ShopList` (0x017C), catalogados mas não portados.
- **Aberto — dano do personagem no Campo:** `event=attack damage=1 hp_left=597`; o ataque usa o
  `Damage` do template da classe, sem equipamento (é o papel da arma Sephira +9).

- **Corrigido — clique de movimento/NPC (2026-11-11, teste manual):** todo `_MSG_Action`
  apontando mais de 4 tiles do personagem era **recusado**
  (`combate recusado: passo de 22 tiles acima do permitido (4)`) e, como o NPC só respondia
  dentro de 5 tiles do clique, a loja nunca abria — daí "só o banco funciona". Agora o clique
  é **destino de caminhada**: o motor anda o orçamento do tick e continua nos ticks seguintes
  (`world::Engine::walk_pending`), e clicar no sprite (`CLICK_RANGE_TILES = 2`) resolve o NPC
  mesmo a distância. Testes: `clique_longe_e_destino_de_caminhada_e_nao_recusa`,
  `clique_no_sprite_do_npc_longe_devolve_alvo`.
- **Corrigido — NPCs faltando em Armia:** o raio da cena subiu de 24 para **56** tiles
  (`DEFAULT_SCENE_VIEW_TILES`). Medido no banco em volta do save (2096/2096): raio 24 = 32
  NPCs, 40 = 45, **56 = 71**; o anel de mercadores em `y >= 2121` e os skillmasters
  (`ForeLearner` em 2090,2047) ficavam fora. A entrega continua por proximidade — o mapa
  inteiro (6 110) não vai no login.
- **Aberto — mensagens do cliente:** o build 7662 **não trata `_MSG_NewAccountFail` (0x0117)**
  (zero `cmp` no binário), por isso senha errada não mostra nada; ele trata `MessagePanel`
  (0x0101), `DeleteCharacterFail` (0x011b) e `CharacterLoginFail` (0x0119). Inventário em
  `docs/cliente/pacotes.md` §"O que o build 7662 realmente trata". Trocar a recusa de login
  exige o layout de texto do `MessagePanel` no contexto de login — pendente de captura.
- **Continua aberto:** movimento visível para outros jogadores (`_MSG_Move` não portado), affect do buff do `Cabuncle`,
  armas Sephira +9 (não existem no `item_catalog`) e a entrega das provas pelos guardiões no
  combate real (drop no abate já existe, falta a jornada até eles).
