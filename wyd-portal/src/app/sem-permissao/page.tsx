import Link from "next/link";

export const metadata = { title: "Sem permissão · WYD2" };

export default function SemPermissao() {
  return (
    <main className="page page--narrow">
      <div className="empty">
        <strong>Acesso restrito</strong>
        <p>
          Sua conta não tem papel de moderador nem de administrador. Peça a promoção a um
          administrador do servidor e entre novamente — o papel é lido no login.
        </p>
        <Link className="ui-btn" href="/login">
          Voltar para o login
        </Link>
      </div>
    </main>
  );
}
