export function Logo() {
  return (
    <span className="flex items-center gap-2">
      <span
        className="text-[22px] leading-none tracking-tight text-fd-foreground"
        style={{ fontFamily: "NuixyberNext, sans-serif" }}
      >
        agh
      </span>
      <span className="rounded-sm border border-fd-border px-1.5 py-[1px] font-mono text-[9px] font-medium uppercase tracking-widest text-fd-muted-foreground">
        Alpha
      </span>
    </span>
  );
}
