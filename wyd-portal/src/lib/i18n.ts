/**
 * Textos da interface do portal.
 *
 * Dicionário único: formulário e rota usam a mesma string. Quando houver segundo
 * idioma, troca-se o valor aqui — não nos componentes.
 */

export const TEXTO = {
  pin: {
    titulo: "Senha numérica",
    ajuda: "Pedida no jogo depois do login. A troca pode ser feita aqui ou no cliente.",
    estadoDefinida: "Definida",
    estadoNaoDefinida: "Não definida",
    estadoIndisponivel: "Indisponível",
    naoDefinidaTitulo: "Sua segunda senha ainda não foi criada",
    naoDefinidaTexto:
      "Ela é definida no cliente, no primeiro acesso. O portal não cria a primeira senha — só troca uma que já existe.",
    passo1: "Entre no jogo com esta conta.",
    passo2: "Quando o jogo pedir a segunda senha, escolha de 4 a 6 dígitos.",
    passo3: "Confirme no jogo. Depois disso, a troca também pode ser feita aqui.",
    naoDefinidaNota: "Perdeu o acesso à conta? Use “Recuperar acesso” ao lado.",
    consultaIndisponivel:
      "Não foi possível consultar o estado da segunda senha agora. Recarregue a página em instantes.",
    campoAtual: "Segunda senha atual",
    campoNovo: "Nova segunda senha",
    campoConfirmacao: "Repita a nova segunda senha",
    campoSenha: "Senha da conta",
    ajudaSenha: "Confirme com a senha de entrada da conta.",
    acao: "Trocar segunda senha",
    enviando: "Trocando…",
    sucesso: "Segunda senha alterada. Ela já vale na próxima entrada no jogo.",
    sessaoExpirada: "Sua sessão expirou. Entre de novo no portal.",
    regras: "Use de 4 a 6 dígitos.",
    conferecaoDivergente: "As duas senhas numéricas não são iguais.",
    atualInvalido:
      "A segunda senha atual não confere. Depois de 5 tentativas, a conta fica bloqueada por 10 minutos.",
    senhaInvalida: "Informe a senha da conta.",
    indisponivel: "Não foi possível falar com o servidor agora. Tente de novo em instantes.",
    invalido: "Não foi possível alterar a segunda senha. Confira os dados e tente de novo.",
    igualAoAtual: "A nova segunda senha precisa ser diferente da atual.",
  },
} as const;
