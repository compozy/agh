export function Logo() {
  return (
    <span className="flex items-center gap-2">
      <span
        aria-hidden
        className="flex h-7 w-7 items-center justify-center rounded-md bg-[#E8572A] font-bold text-[13px] text-white"
      >
        A
      </span>
      <span className="font-semibold tracking-tight text-fd-foreground">AGH</span>
      <span className="rounded-sm border border-fd-border px-1.5 py-[1px] font-mono text-[9px] font-medium uppercase tracking-widest text-fd-muted-foreground">
        Alpha
      </span>
    </span>
  );
}
