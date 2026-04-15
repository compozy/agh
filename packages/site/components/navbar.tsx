"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  { href: "/", label: "Home" },
  { href: "/runtime", label: "Runtime" },
  { href: "/protocol", label: "Protocol" },
];

export function Navbar() {
  const pathname = usePathname();

  return (
    <header className="sticky top-0 z-50 border-b border-[var(--color-divider)] bg-[var(--color-canvas)]">
      <nav className="mx-auto flex h-14 max-w-screen-xl items-center gap-6 px-4">
        <Link
          href="/"
          className="font-mono text-sm font-semibold uppercase tracking-widest text-[var(--color-accent)]"
        >
          AGH
        </Link>
        <div className="flex items-center gap-1">
          {links.map(link => {
            const isActive = link.href === "/" ? pathname === "/" : pathname.startsWith(link.href);
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`rounded-md px-3 py-1.5 text-sm transition-colors ${
                  isActive
                    ? "text-[var(--color-text-primary)]"
                    : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
                }`}
              >
                {link.label}
              </Link>
            );
          })}
        </div>
      </nav>
    </header>
  );
}
