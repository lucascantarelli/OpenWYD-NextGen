import { redirect } from "next/navigation";

import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

/** Entrada do portal: manda quem já tem sessão direto para a primeira tela. */
export default async function Home() {
  const session = await getSession();
  redirect(session && isStaff(session.role) ? "/painel" : "/login");
}
