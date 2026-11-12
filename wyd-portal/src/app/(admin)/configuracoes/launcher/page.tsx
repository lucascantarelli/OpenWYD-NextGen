import { FileSignature, History, ShieldCheck } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { configuracaoAtual, historicoConfiguracao } from "@/server/launcher-dist";

import { publicarConfiguracaoAction } from "./actions";

export const dynamic = "force-dynamic";

export const metadata = { title: "Launcher · Configurações · WYD2" };

const fmtBytes = (bytes: number): string =>
  `${bytes.toLocaleString("pt-BR")} B`;

const fmtQuando = (iso: string): string => {
  const data = new Date(iso);
  return Number.isNaN(data.getTime()) ? "—" : data.toLocaleString("pt-BR");
};

/** Corpo de exemplo: o assinador anexa a linha `assinatura` ao gerar o arquivo. */
const MODELO = `# wyd-launcher-config v1
revisao = 2
portal.base = http://127.0.0.1:3000
launcher.manifesto = /api/launcher/manifesto
launcher.pacote = /api/launcher/pacote
# cor.accent = #c8a24a
# servidor.1 = WYD
`;

/**
 * Revisão e publicação da configuração do launcher.
 *
 * O fluxo é explícito para não haver surpresa: o operador assina fora do portal
 * (`sign-config.mjs`), cola o texto completo, o portal confere forma e ordem das
 * revisões, grava os bytes exatos e guarda o histórico. O portal nunca vê a chave
 * HMAC — quem autentica a config é o launcher.
 */
export default async function LauncherConfigPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const one = (value: string | string[] | undefined): string =>
    Array.isArray(value) ? (value[0] ?? "") : (value ?? "");
  const erro = one(params.erro).trim();
  const ok = one(params.ok).trim();

  const atual = configuracaoAtual();
  const historico = historicoConfiguracao().slice().reverse();

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Distribuição</p>
          <h1 className="page-head__title">Configuração do launcher</h1>
          <p className="page-head__sub">
            Publique o <code>launcher.conf</code> assinado fora do portal. O portal não
            assina: ele confere a forma, preserva os bytes e guarda quem publicou cada
            revisão.
          </p>
        </div>
        <div className="page-head__actions">
          <Badge tone={atual ? "ok" : "warn"}>
            {atual ? `revisão ${atual.revisao}` : "não publicada"}
          </Badge>
        </div>
      </div>

      <nav className="chips" aria-label="Configurações do launcher" style={{ marginBottom: 16 }}>
        <Link className="chip" href="/configuracoes">
          Configurações do servidor
        </Link>
        <Link className="chip" href="/configuracoes/launcher" aria-current="page">
          Launcher
        </Link>
        <Link className="chip" href="/api/launcher/configuracao">
          Ver bytes públicos
        </Link>
      </nav>

      {erro ? (
        <div className="card">
          <div className="card__body">
            <p role="alert" className="login-error">
              {erro}
            </p>
          </div>
        </div>
      ) : null}
      {ok ? (
        <div className="card">
          <div className="card__body">
            <p role="status">Configuração publicada e registrada no histórico.</p>
          </div>
        </div>
      ) : null}

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <ShieldCheck size={16} strokeWidth={1.8} aria-hidden="true" /> Em vigor
            </h2>
            <p className="card__sub">
              Bytes exatamente como o launcher baixa em{" "}
              <code>/api/launcher/configuracao</code>.
            </p>
          </div>
        </div>
        <div className="card__body">
          {atual ? (
            <dl className="field-grid">
              <div>
                <dt className="muted small">Revisão</dt>
                <dd className="mono">{atual.revisao}</dd>
              </div>
              <div>
                <dt className="muted small">SHA-256</dt>
                <dd className="mono small">{atual.sha256}</dd>
              </div>
              <div>
                <dt className="muted small">Tamanho</dt>
                <dd>{fmtBytes(atual.bytes)}</dd>
              </div>
              <div>
                <dt className="muted small">Publicada por</dt>
                <dd className="small">
                  {atual.autor ?? "—"}
                  {atual.em ? ` em ${fmtQuando(atual.em)}` : ""}
                </dd>
              </div>
            </dl>
          ) : (
            <p className="muted small" style={{ margin: 0 }}>
              Nenhuma configuração publicada nesta instância. O launcher usa a identidade
              embutida até a primeira publicação.
            </p>
          )}
        </div>
      </section>

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <FileSignature size={16} strokeWidth={1.8} aria-hidden="true" /> Publicar revisão
            </h2>
            <p className="card__sub">
              Assine com{" "}
              <code>node tools/launcher/sign-config.mjs corpo.conf chave.bin launcher.conf</code>{" "}
              e cole o conteúdo completo (a última linha precisa ser{" "}
              <code>assinatura = &lt;64 hex&gt;</code>).
            </p>
          </div>
        </div>
        <div className="card__body">
          <form action={publicarConfiguracaoAction} className="field-grid">
            <label className="login-field">
              <span>Arquivo assinado</span>
              <textarea
                className="ui-input"
                name="texto"
                rows={12}
                required
                spellCheck={false}
                placeholder={MODELO}
                aria-label="Conteúdo do launcher.conf assinado"
              />
            </label>
            <div className="card__foot" style={{ padding: 0, border: 0 }}>
              <span className="muted small">
                Revisão repetida ou menor que a publicada é recusada; o arquivo anterior fica
                em <code>launcher.conf.bak</code>.
              </span>
              <button className="ui-btn ui-btn--primary" type="submit">
                Publicar configuração
              </button>
            </div>
          </form>
        </div>
      </section>

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <History size={16} strokeWidth={1.8} aria-hidden="true" /> Histórico de revisões
            </h2>
            <p className="card__sub">
              Append-only, com autor da sessão, SHA-256 e tamanho de cada publicação.
            </p>
          </div>
        </div>
        {historico.length === 0 ? (
          <div className="card__body">
            <div className="empty">
              <strong>Sem publicações</strong>
              <p>A primeira revisão publicada aparece aqui.</p>
            </div>
          </div>
        ) : (
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Revisão</th>
                    <th>Quando</th>
                    <th>Autor</th>
                    <th>Tamanho</th>
                    <th>SHA-256</th>
                  </tr>
                </thead>
                <tbody>
                  {historico.map((item) => (
                    <tr key={`${item.revisao}-${item.sha256}`}>
                      <td className="mono">{item.revisao}</td>
                      <td className="small">{fmtQuando(item.em)}</td>
                      <td className="small">{item.autor}</td>
                      <td>{fmtBytes(item.bytes)}</td>
                      <td className="mono small">{item.sha256.slice(0, 16)}…</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </section>
    </>
  );
}
