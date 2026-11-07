import Link from "next/link";

export const metadata = { title: "Sem permissão · WYD2" };

export default function SemPermissao() {
  return (
    <main style={{ maxWidth: 520, margin: "0 auto", padding: "96px 24px" }}>
      <h1 style={{ fontSize: 22, marginBottom: 8 }}>Sem permissão</h1>
      <p style={{ color: "var(--muted)" }}>
        Sua conta não tem papel de moderador nem de administrador. Peça a promoção a um
        administrador do servidor e entre novamente — o papel só é lido no login.
      </p>
      <p>
        <Link href="/login">Voltar para o login</Link>
      </p>
    </main>
  );
}
