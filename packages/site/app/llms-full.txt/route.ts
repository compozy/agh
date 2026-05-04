import { getLLMText } from "@/lib/get-llm-text";
import { protocolDocs, runtimeDocs } from "@/lib/source";

export const dynamic = "force-static";
export const revalidate = false;

export async function GET() {
  const pages = [...runtimeDocs.getPages(), ...protocolDocs.getPages()];
  const sections = await Promise.all(pages.map(page => getLLMText(page)));
  return new Response(sections.join("\n\n---\n\n"), {
    headers: { "Content-Type": "text/plain; charset=utf-8" },
  });
}
