# Documento de bugs encontrados

## Mensagem de erro no login dentro do cliente

- ao executar o login, com dados incorretos no cliente, dados de usuário e senha, o sistema não exibe uma mensagem de erro clara, apenas retorna para a tela de login sem qualquer feedback.
- aparece a mensagem conectando no servidor depois some e nada acontece.

## Segunda senha dentro do cliente

- a segunda senha não está sendo validada corretamente, mesmo com dados incorretos o sistema permite o acesso, o que representa um risco de segurança.

## Apagar personagem dentro do cliente

- Ao apagar personagens, o jogo pede para incluir a senha, essa senha não está sendo validada, aceita qualquer senha, mesmo que esteja incorreta, permitindo apagar personagens sem a devida autenticação.

## Nome do char vermelho dentro do cliente

- ao criar um novo personagem, o nome do char aparece em vermelho, referente ao sistema de CP
- comando /cp não funciona, quando deveria mostrar as informações do CP do char

## Pontos de atributos incorretos dentro do cliente 

- os pontos de atributos FOR/DES/INT/CONS estão zerados, mesmo que o personagem tenha pontos distribuídos, o que impede a visualização correta dos atributos do personagem.
- os pontos de imunidades estão 245, creio que deveriam estar zerados, pois o personagem não possui imunidades.

## Nivel inicial do personagem

- O personagem está iniciando level 2, creio que deveria iniciar no level 1, o que pode gerar problemas de balanceamento do jogo.

## Npcs de loja/banco

- os npcs de loja não estão funcionando corretamente, não é possível comprar ou vender itens, o que impede a progressão do personagem e a economia do jogo.
- o npc de bancoo não permite guardar nenhum item ou gold do banco, mesmo com o personagem com gold suficiente, o que impede a segurança do gold do personagem.

## Ativar modo andar/correr do personagem

- o personagem não consegue alternar entre os modos de andar e correr, não sei o atalho relacionado, mas o personagem não consegue alternar entre os modos, o que pode gerar problemas de movimentação e exploração do mapa.
