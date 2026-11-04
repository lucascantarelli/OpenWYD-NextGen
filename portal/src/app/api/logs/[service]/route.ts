import {
  isLogTarget,
  openContainerLogs,
  parseSince,
  parseTail,
  redact,
} from "@/server/docker";
import { getSession, isStaff } from "@/server/session";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;

/** Acumula pedaços e emite linha por linha (o stream do Docker corta no meio). */
function makeSplitter(onLine: (line: string) => void) {
  let rest = "";
  return {
    push(chunk: string) {
      const parts = (rest + chunk).split(/\r?\n/);
      rest = parts.pop() ?? "";
      for (const part of parts) onLine(part);
    },
    flush() {
      if (rest) onLine(rest);
      rest = "";
    },
  };
}

/**
 * `GET /api/logs/<serviço>` — `text/event-stream` com o stdout/stderr do
 * container.
 *
 * A sessão é a única autorização aqui: não existe RPC admin para "ver log". Sem
 * sessão 401, `player` 403, serviço fora da whitelist 404 — antes de qualquer
 * conexão com o Docker (logpipe ou CLI) nascer.
 */
export async function GET(
  request: Request,
  context: { params: Promise<{ service: string }> },
) {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada: entre novamente" }, { status: 401, headers: NO_STORE });
  }
  if (!isStaff(session.role)) {
    return Response.json(
      { error: "sua conta não tem permissão para ver os logs dos containers" },
      { status: 403, headers: NO_STORE },
    );
  }

  const { service } = await context.params;
  if (!isLogTarget(service)) {
    return Response.json({ error: `serviço desconhecido: ${service}` }, { status: 404, headers: NO_STORE });
  }

  const url = new URL(request.url);
  const tail = parseTail(url.searchParams.get("tail"));
  const since = parseSince(url.searchParams.get("since"));

  const handle = openContainerLogs(service, { tail, since, signal: request.signal });
  const encoder = new TextEncoder();

  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      let closed = false;
      const send = (payload: string) => {
        if (closed) return;
        try {
          controller.enqueue(encoder.encode(payload));
        } catch {
          closed = true;
        }
      };

      const splitter = makeSplitter((line) => send(`data: ${redact(line)}\n\n`));
      console.info(`[logs] stream open service=${service} account=${session.accountId}`);
      send("retry: 2000\n\n");

      try {
        const reader = handle.stream.getReader();
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          if (value) splitter.push(value);
        }
        splitter.flush();
      } catch (error) {
        splitter.flush();
        send(`data: [portal] falha ao ler o stream: ${(error as Error).message}\n\n`);
      }

      send(`data: [portal] o stream do serviço terminou\n\n`);
      closed = true;
      try {
        controller.close();
      } catch {
        // já fechado pelo cliente
      }
      console.info(`[logs] stream closed service=${service} reason=${request.signal.aborted ? "client-abort" : "source-end"}`);
    },
    cancel() {
      handle.close();
    },
  });

  return new Response(stream, {
    headers: {
      "content-type": "text/event-stream; charset=utf-8",
      "cache-control": "no-store, no-transform",
      "x-accel-buffering": "no",
      connection: "keep-alive",
    },
  });
}
