import Link from "next/link";

/** Rodapé público: só navegação de jogador e créditos do projeto. */
export function SiteFooter() {
  return (
    <footer className="site-foot">
      <div className="site-foot__brand">
        <span className="brand__mark" aria-hidden="true">
          W
        </span>
        <div>
          <strong>WYD2</strong>
          <p>Servidor privado · cliente 7662 · protocolo 12000</p>
        </div>
      </div>

      <nav className="site-foot__links" aria-label="Rodapé">
        <Link href="/noticias">Notícias</Link>
        <Link href="/ranking">Ranking</Link>
        <Link href="/downloads">Downloads</Link>
        <Link href="/guias">Guias</Link>
        <Link href="/enciclopedia">Enciclopédia</Link>
        <Link href="/conta">Minha conta</Link>
      </nav>

      <p className="site-foot__note">
        Projeto de servidor próprio, sem vínculo com o WYD oficial. Nenhum binário do
        cliente é redistribuído por este site.
      </p>
    </footer>
  );
}
