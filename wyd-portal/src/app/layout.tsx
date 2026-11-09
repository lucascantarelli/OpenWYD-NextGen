import type { Metadata } from "next";
import { Cinzel, Manrope } from "next/font/google";

import "./globals.css";

/*
 * Duas famílias, dois papéis:
 * - Cinzel (serifada lapidar) assina títulos, marca e números de destaque;
 * - Manrope (sans geométrica) carrega a operação: tabelas, formulários, legendas.
 * Ambas são expostas como variáveis CSS e consumidas só por `globals.css`.
 */
const display = Cinzel({
  subsets: ["latin"],
  weight: ["500", "600", "700"],
  display: "swap",
  variable: "--font-display",
});

const sans = Manrope({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  display: "swap",
  variable: "--font-sans",
});

export const metadata: Metadata = {
  title: "WYD2 · Administração",
  description: "Portal de administração do servidor WYD2",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR" className={`${display.variable} ${sans.variable}`}>
      <body>{children}</body>
    </html>
  );
}
