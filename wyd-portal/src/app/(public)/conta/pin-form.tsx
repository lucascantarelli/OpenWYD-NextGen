"use client";

import { KeyRound, ShieldCheck } from "lucide-react";
import { useState } from "react";

import { TEXTO } from "@/lib/i18n";

/** 4 a 6 dígitos — a mesma regra do servidor (`service::pin::valid_pin`). */
const PIN = /^\d{4,6}$/;

/**
 * Segunda senha da conta, em três estados:
 *
 * - `defined === null`: o portal não conseguiu ler a conta — não há o que oferecer;
 * - `defined === false`: a conta ainda não tem PIN. O primeiro é criado no cliente,
 *   no primeiro acesso; o portal explica o caminho em vez de exibir um formulário
 *   que o servidor recusaria (o gateway só troca PIN existente);
 * - `defined === true`: troca pelo portal, passando pelo mesmo fluxo do cliente.
 *
 * A validação daqui só evita uma ida inútil ao servidor. Quem decide se a senha
 * atual vale, se a conta está bloqueada e se a nova é aceita é o servidor do jogo.
 * Nenhum hash, senha ou PIN é guardado no browser.
 */
export default function PinForm({ login, defined }: { login: string; defined: boolean | null }) {
  const [senha, setSenha] = useState("");
  const [atual, setAtual] = useState("");
  const [novo, setNovo] = useState("");
  const [confirmacao, setConfirmacao] = useState("");
  const [pending, setPending] = useState(false);
  const [erro, setErro] = useState("");
  const [ok, setOk] = useState("");

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErro("");
    setOk("");
    if (!PIN.test(atual) || !PIN.test(novo)) {
      setErro(TEXTO.pin.regras);
      return;
    }
    if (novo !== confirmacao) {
      setErro(TEXTO.pin.conferecaoDivergente);
      return;
    }
    if (atual === novo) {
      setErro(TEXTO.pin.igualAoAtual);
      return;
    }
    if (senha === "") {
      setErro(TEXTO.pin.senhaInvalida);
      return;
    }

    setPending(true);
    try {
      const res = await fetch("/api/panel/pin", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          login,
          password: senha,
          current_pin: atual,
          new_pin: novo,
        }),
        cache: "no-store",
      });
      const body = (await res.json().catch(() => ({}))) as { changed?: boolean };
      if (res.ok && body.changed) {
        setOk(TEXTO.pin.sucesso);
        setSenha("");
        setAtual("");
        setNovo("");
        setConfirmacao("");
      } else if (res.status === 401) {
        setErro(TEXTO.pin.sessaoExpirada);
      } else if (res.status === 422) {
        setErro(TEXTO.pin.atualInvalido);
      } else if (res.status === 503) {
        setErro(TEXTO.pin.indisponivel);
      } else {
        setErro(TEXTO.pin.invalido);
      }
    } catch {
      setErro(TEXTO.pin.indisponivel);
    } finally {
      setPending(false);
    }
  }

  if (defined === null) {
    return <p className="muted small">{TEXTO.pin.consultaIndisponivel}</p>;
  }

  if (!defined) {
    return (
      <div className="pin-intro">
        <div className="pin-intro__head">
          <span className="pin-intro__seal" aria-hidden="true">
            <ShieldCheck size={18} strokeWidth={1.6} />
          </span>
          <div>
            <h3 className="pin-intro__title">{TEXTO.pin.naoDefinidaTitulo}</h3>
            <p className="pin-intro__text">{TEXTO.pin.naoDefinidaTexto}</p>
          </div>
        </div>
        <ol className="pin-steps">
          <li>{TEXTO.pin.passo1}</li>
          <li>{TEXTO.pin.passo2}</li>
          <li>{TEXTO.pin.passo3}</li>
        </ol>
        <p className="pin-intro__note">{TEXTO.pin.naoDefinidaNota}</p>
      </div>
    );
  }

  return (
    // `noValidate`: as mensagens em português vêm do handler, não do balão nativo.
    <form onSubmit={submit} noValidate className="ui-form">
      <p className="ui-field__hint">{TEXTO.pin.ajuda}</p>

      <div className="ui-field">
        <label className="ui-field__label" htmlFor="pin-senha">
          {TEXTO.pin.campoSenha}
        </label>
        <input
          id="pin-senha"
          className="ui-input"
          type="password"
          autoComplete="current-password"
          value={senha}
          maxLength={12}
          onChange={(event) => setSenha(event.target.value)}
        />
        <span className="ui-field__hint">{TEXTO.pin.ajudaSenha}</span>
      </div>

      <div className="field-grid">
        <div className="ui-field">
          <label className="ui-field__label" htmlFor="pin-atual">
            {TEXTO.pin.campoAtual}
          </label>
          <input
            id="pin-atual"
            className="ui-input"
            inputMode="numeric"
            autoComplete="one-time-code"
            value={atual}
            maxLength={6}
            onChange={(event) => setAtual(event.target.value.replace(/\D/g, ""))}
          />
        </div>

        <div className="ui-field">
          <label className="ui-field__label" htmlFor="pin-novo">
            {TEXTO.pin.campoNovo}
          </label>
          <input
            id="pin-novo"
            className="ui-input"
            inputMode="numeric"
            autoComplete="new-password"
            value={novo}
            maxLength={6}
            onChange={(event) => setNovo(event.target.value.replace(/\D/g, ""))}
          />
        </div>
      </div>

      <div className="ui-field">
        <label className="ui-field__label" htmlFor="pin-confirmacao">
          {TEXTO.pin.campoConfirmacao}
        </label>
        <input
          id="pin-confirmacao"
          className="ui-input"
          inputMode="numeric"
          autoComplete="new-password"
          value={confirmacao}
          maxLength={6}
          onChange={(event) => setConfirmacao(event.target.value.replace(/\D/g, ""))}
        />
      </div>

      <div className="ui-form__actions">
        <button type="submit" className="ui-btn ui-btn--primary" disabled={pending}>
          <KeyRound size={15} strokeWidth={1.8} aria-hidden="true" />
          {pending ? TEXTO.pin.enviando : TEXTO.pin.acao}
        </button>
      </div>

      {erro ? (
        <p role="alert" className="ui-field__error">
          {erro}
        </p>
      ) : null}
      {ok ? (
        <p role="status" className="text-ok">
          {ok}
        </p>
      ) : null}
    </form>
  );
}
