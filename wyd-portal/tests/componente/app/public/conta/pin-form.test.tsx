// @vitest-environment jsdom
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TEXTO } from "@/lib/i18n";

import PinForm from "@/app/(public)/conta/pin-form";

/**
 * Componente **real** da tela de conta (`/conta` usa este arquivo), sem mock do
 * formulário: o teste monta o `PinForm` e conversa com ele como o jogador.
 *
 * O que fica mockado é só a borda de rede (`fetch`). Assim o teste prova o que o
 * componente faz de verdade: os três estados da segunda senha, os textos do i18n,
 * validação de formato/confirmação, o POST para `/api/panel/pin` e a mensagem de
 * produto exibida em cada falha — sem nunca calcular troca nem hash por conta própria.
 */

function resposta(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response;
}

async function preencher(campos: {
  senha?: string;
  atual?: string;
  novo?: string;
  confirmacao?: string;
}) {
  const user = userEvent.setup();
  if (campos.senha !== undefined) {
    await user.type(screen.getByLabelText(TEXTO.pin.campoSenha), campos.senha);
  }
  if (campos.atual !== undefined) {
    await user.type(screen.getByLabelText(TEXTO.pin.campoAtual), campos.atual);
  }
  if (campos.novo !== undefined) {
    await user.type(screen.getByLabelText(TEXTO.pin.campoNovo), campos.novo);
  }
  if (campos.confirmacao !== undefined) {
    await user.type(screen.getByLabelText(TEXTO.pin.campoConfirmacao), campos.confirmacao);
  }
}

async function enviar() {
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: TEXTO.pin.acao }));
}

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("PinForm da tela de conta", () => {
  it("com a segunda senha definida, mostra os quatro campos", () => {
    render(<PinForm login="jogador" defined={true} />);

    expect(screen.getByText(TEXTO.pin.ajuda)).toBeTruthy();
    expect(screen.getByText(TEXTO.pin.ajudaSenha)).toBeTruthy();
    expect(screen.getByLabelText(TEXTO.pin.campoSenha)).toBeTruthy();
    expect(screen.getByLabelText(TEXTO.pin.campoAtual)).toBeTruthy();
    expect(screen.getByLabelText(TEXTO.pin.campoNovo)).toBeTruthy();
    expect(screen.getByLabelText(TEXTO.pin.campoConfirmacao)).toBeTruthy();
    expect(screen.getByRole("button", { name: TEXTO.pin.acao })).toBeTruthy();
  });

  it("com a segunda senha ainda não definida, explica o caminho no cliente e não oferece troca", () => {
    render(<PinForm login="jogador" defined={false} />);

    expect(screen.getByText(TEXTO.pin.naoDefinidaTitulo)).toBeTruthy();
    expect(screen.getByText(TEXTO.pin.naoDefinidaTexto)).toBeTruthy();
    expect(screen.getByText(TEXTO.pin.passo1)).toBeTruthy();
    expect(screen.getByText(TEXTO.pin.passo2)).toBeTruthy();
    expect(screen.getByText(TEXTO.pin.passo3)).toBeTruthy();
    expect(screen.queryByRole("button", { name: TEXTO.pin.acao })).toBeNull();
    expect(screen.queryByLabelText(TEXTO.pin.campoAtual)).toBeNull();
  });

  it("quando a consulta falha, diz isso em linguagem de produto e não inventa estado", () => {
    render(<PinForm login="jogador" defined={null} />);

    expect(screen.getByText(TEXTO.pin.consultaIndisponivel)).toBeTruthy();
    expect(screen.queryByText(TEXTO.pin.naoDefinidaTitulo)).toBeNull();
    expect(screen.queryByRole("button", { name: TEXTO.pin.acao })).toBeNull();
  });

  it("envia POST /api/panel/pin com os quatro campos e mostra sucesso limpando os sensíveis", async () => {
    fetchMock.mockResolvedValue(resposta(200, { ok: true, changed: true }));
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "1357", novo: "8642", confirmacao: "8642" });
    await enviar();

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("/api/panel/pin");
    expect(init.method).toBe("POST");
    expect(JSON.parse(String(init.body))).toEqual({
      login: "jogador",
      password: "SenhaForte1",
      current_pin: "1357",
      new_pin: "8642",
    });

    expect(await screen.findByText(TEXTO.pin.sucesso)).toBeTruthy();
    for (const texto of [
      TEXTO.pin.campoSenha,
      TEXTO.pin.campoAtual,
      TEXTO.pin.campoNovo,
      TEXTO.pin.campoConfirmacao,
    ]) {
      expect((screen.getByLabelText(texto) as HTMLInputElement).value).toBe("");
    }
  });

  it("recusa PIN fora de 4–6 dígitos sem chamar o servidor", async () => {
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "1357", novo: "12", confirmacao: "12" });
    await enviar();

    expect(await screen.findByText(TEXTO.pin.regras)).toBeTruthy();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("recusa confirmação divergente sem chamar o servidor", async () => {
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "1357", novo: "8642", confirmacao: "8643" });
    await enviar();

    expect(await screen.findByText(TEXTO.pin.conferecaoDivergente)).toBeTruthy();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("recusa nova senha igual à atual sem chamar o servidor", async () => {
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "1357", novo: "1357", confirmacao: "1357" });
    await enviar();

    expect(await screen.findByText(TEXTO.pin.igualAoAtual)).toBeTruthy();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("mostra o PIN atual errado/bloqueio como texto de produto quando a rota responde 422", async () => {
    fetchMock.mockResolvedValue(resposta(422, { error: "PIN atual inválido ou conta bloqueada" }));
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "0000", novo: "8642", confirmacao: "8642" });
    await enviar();

    expect(await screen.findByText(TEXTO.pin.atualInvalido)).toBeTruthy();
    // A mensagem crua do backend nunca chega à tela.
    expect(screen.queryByText(/bloqueada por tentativas/i)).toBeNull();
    // Senha não some no erro (o jogador precisa reenviar), mas nada é persistido.
    expect((screen.getByLabelText(TEXTO.pin.campoSenha) as HTMLInputElement).value).toBe(
      "SenhaForte1",
    );
  });

  it("mostra sessão expirada quando a rota responde 401", async () => {
    fetchMock.mockResolvedValue(resposta(401, { error: "sessão expirada" }));
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "1357", novo: "8642", confirmacao: "8642" });
    await enviar();

    expect(await screen.findByText(TEXTO.pin.sessaoExpirada)).toBeTruthy();
  });

  it("mostra indisponibilidade sem jargão quando a rota responde 503", async () => {
    fetchMock.mockResolvedValue(resposta(503, { error: "servidor de jogo indisponível" }));
    render(<PinForm login="jogador" defined={true} />);

    await preencher({ senha: "SenhaForte1", atual: "1357", novo: "8642", confirmacao: "8642" });
    await enviar();

    expect(await screen.findByText(TEXTO.pin.indisponivel)).toBeTruthy();
    expect(screen.queryByText(/servidor de jogo/i)).toBeNull();
  });
});
