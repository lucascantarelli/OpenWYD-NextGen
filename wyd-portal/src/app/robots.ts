import type { MetadataRoute } from "next";

/**
 * Área administrativa não é indexável.
 *
 * O bloqueio aqui é higiene: as rotas de staff continuam protegidas pelo
 * `proxy.ts`, pelo layout de `(admin)` e pela autorização do gateway. Nenhuma
 * página pública linka para elas.
 */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: [
          "/login",
          "/painel",
          "/contas",
          "/entregas",
          "/correio",
          "/financeiro",
          "/itens",
          "/monitoramento",
          "/configuracoes",
          "/auditoria",
          "/economia",
          "/sem-permissao",
        ],
      },
    ],
  };
}
