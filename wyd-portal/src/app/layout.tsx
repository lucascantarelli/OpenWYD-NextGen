import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "WYD2 · Administração",
  description: "Portal de administração do servidor WYD2",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
