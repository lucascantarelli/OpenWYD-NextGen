import { clearSession } from "@/server/session";

export const dynamic = "force-dynamic";

export async function POST() {
  await clearSession();
  return new Response(null, { status: 204, headers: { "cache-control": "no-store" } });
}
