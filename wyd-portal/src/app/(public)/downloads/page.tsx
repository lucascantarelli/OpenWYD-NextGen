import { Download, HardDrive, ShieldCheck, Terminal } from "lucide-react";
import Link from "next/link";

import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import { getLauncherPublicacao } from "@/server/launcher-dist";
import { CLIENT_BUILD, PROTOCOL_VERSION, getStatusPublico } from "@/server/public-site";

export const dynamic = "force-dynamic";

export const metadata = {
  title: "Downloads · WYD2",
  description: "Cliente 7662, launcher do projeto e a revisão de conteúdo vigente.",
};

function fmtBytes(bytes: number): string {
  if (bytes <= 0) return "0 B";
  const unidades = ["B", "KB", "MB", "GB"];
  const indice = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), unidades.length - 1);
  const valor = bytes / 1024 ** indice;
  return `${valor.toFixed(indice === 0 ? 0 : 1)} ${unidades[indice]}`;
}

export default async function DownloadsPage() {
  const [status, launcher] = await Promise.all([
    getStatusPublico(),
    Promise.resolve(getLauncherPublicacao()),
  ]);

  return (
    <>
      <SiteHeader current="/downloads" />

      <main className="site site--narrow">
        <header className="site-page-head">
          <p className="site-section__eyebrow">Downloads</p>
          <h1>Baixe, atualize e entre</h1>
          <p>
            O cliente é o build 7662. O launcher confere o manifesto assinado antes de tocar nos
            arquivos e escreve o <code>serverlist.bin</code> com o endereço do servidor.
          </p>
        </header>

        <section className="download-grid">
          <article className="download-card">
            <HardDrive size={20} strokeWidth={1.7} aria-hidden="true" />
            <h2>Cliente do jogo</h2>
            <dl className="kv">
              <dt>Build</dt>
              <dd className="mono">{CLIENT_BUILD}</dd>
              <dt>Protocolo</dt>
              <dd className="mono">{PROTOCOL_VERSION}</dd>
              <dt>Conteúdo</dt>
              <dd className="mono">rev. {status.configRevision ?? "—"}</dd>
            </dl>
            <p className="muted small">
              Os binários do cliente não são redistribuídos por este site. O launcher baixa os
              arquivos do espelho do projeto a partir do manifesto.
            </p>
          </article>

          <article className="download-card">
            <Download size={20} strokeWidth={1.7} aria-hidden="true" />
            <h2>Launcher</h2>
            {launcher.pacote ? (
              <>
                <dl className="kv">
                  <dt>Pacote</dt>
                  <dd className="mono">{launcher.pacote.nome}</dd>
                  <dt>Tamanho</dt>
                  <dd>{fmtBytes(launcher.pacote.bytes)}</dd>
                  <dt>SHA-256</dt>
                  <dd className="mono" title={launcher.pacote.sha256}>
                    {launcher.pacote.sha256.slice(0, 16)}…
                  </dd>
                </dl>
                <a className="ui-btn ui-btn--primary" href="/api/launcher/pacote" download>
                  Baixar launcher
                </a>
              </>
            ) : (
              <>
                <p className="muted">
                  A distribuição do launcher ainda não foi publicada neste servidor.
                </p>
                <p className="muted small">
                  Publique a pasta de distribuição e o manifesto; a página passa a servir o
                  download real sem mudança de código.
                </p>
              </>
            )}
          </article>

          <article className="download-card">
            <ShieldCheck size={20} strokeWidth={1.7} aria-hidden="true" />
            <h2>Status do serviço</h2>
            <dl className="kv">
              <dt>Servidor</dt>
              <dd>{status.gateway ? "no ar" : "indisponível"}</dd>
              <dt>Banco</dt>
              <dd>
                {status.database ? `conectado (${status.dbLatencyMs ?? "—"} ms)` : "sem leitura"}
              </dd>
            </dl>
            <p className="muted small">
              Se o servidor estiver reiniciando, o launcher continua abrindo e a entrada volta
              assim que o canal responder.
            </p>
          </article>
        </section>

        <section className="site-section">
          <header className="site-section__head">
            <div>
              <p className="site-section__eyebrow">Manifesto</p>
              <h2>Integridade da distribuição</h2>
            </div>
            {launcher.manifestoUrl && (
              <a className="ui-btn ui-btn--ghost ui-btn--sm" href={launcher.manifestoUrl}>
                Abrir manifesto
              </a>
            )}
          </header>

          {launcher.publicada ? (
            <dl className="kv">
              <dt>Build</dt>
              <dd className="mono">{launcher.build ?? "—"}</dd>
              <dt>Protocolo</dt>
              <dd className="mono">{launcher.protocol ?? "—"}</dd>
              <dt>Arquivos</dt>
              <dd>{launcher.arquivos}</dd>
              <dt>Tamanho total</dt>
              <dd>{fmtBytes(launcher.bytes)}</dd>
            </dl>
          ) : (
            <div className="banner">
              <span>
                <strong>Distribuição não publicada.</strong> {launcher.motivo}. Para publicar,
                gere o manifesto da pasta com{" "}
                <code>node tools/launcher/dist-manifest.mjs dist/launcher 7662 12000</code> e
                monte a pasta no portal (<code>LAUNCHER_DIST_DIR</code>).
              </span>
            </div>
          )}
        </section>

        <section className="site-section">
          <header className="site-section__head">
            <div>
              <p className="site-section__eyebrow">Pelo terminal</p>
              <h2>Conferir a instalação</h2>
            </div>
          </header>
          <div className="download-code">
            <Terminal size={16} strokeWidth={1.7} aria-hidden="true" />
            <pre>
              <code>{`wyd-launcher check <pasta-do-jogo> --manifest manifest.txt`}</code>
            </pre>
          </div>
          <p className="muted small" style={{ marginTop: "var(--space-3)" }}>
            O comando compara tamanho e SHA-256 de cada arquivo. Verificação suja não abre o jogo.
          </p>
        </section>
      </main>

      <SiteFooter />
    </>
  );
}
