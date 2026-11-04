export type BadgeTone = "neutral" | "ok" | "warn" | "danger" | "accent";

export function Badge({
  children,
  tone = "neutral",
  title,
}: {
  children: React.ReactNode;
  tone?: BadgeTone;
  title?: string;
}) {
  const cls = tone === "neutral" ? "ui-badge" : `ui-badge ui-badge--${tone}`;
  return (
    <span className={cls} title={title}>
      {children}
    </span>
  );
}
