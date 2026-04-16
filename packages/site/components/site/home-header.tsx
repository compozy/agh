"use client";

import { buttonVariants, cn } from "@agh/ui";
import { usePathname } from "fumadocs-core/framework";
import { useHomeLayout } from "fumadocs-ui/layouts/home";
import Link from "next/link";
import { Logo } from "@/components/logo";
import { baseOptions } from "@/lib/layout.shared";
import type { ComponentProps } from "react";

const primaryLinks = [
  { href: "/", label: "Home" },
  { href: "/runtime", label: "Runtime" },
  { href: "/protocol", label: "AGH Network" },
];

function isActive(pathname: string, href: string) {
  if (href === "/") {
    return pathname === href;
  }

  return pathname === href || pathname.startsWith(`${href}/`);
}

function HeaderLink({ href, label, pathname }: { href: string; label: string; pathname: string }) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex items-center rounded-full px-3 py-1.5 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text-primary)]",
        isActive(pathname, href) && "bg-[rgba(232,87,42,0.12)] text-[var(--color-accent)]"
      )}
    >
      {label}
    </Link>
  );
}

export function HomeHeader(props: ComponentProps<"header">) {
  const pathname = usePathname();
  const { slots } = useHomeLayout();

  return (
    <header
      {...props}
      className={cn(
        "sticky top-0 z-40 border-b border-[var(--color-divider)] bg-[rgba(18,18,18,0.92)] backdrop-blur-xl",
        props.className
      )}
    >
      <div className="mx-auto flex h-14 w-full max-w-[var(--site-layout-width)] items-center gap-3 px-4 sm:px-6 lg:gap-5">
        <Link href="/" className="shrink-0">
          <Logo />
        </Link>

        <nav className="hidden items-center gap-1 md:flex">
          {primaryLinks.map(link => (
            <HeaderLink key={link.href} href={link.href} label={link.label} pathname={pathname} />
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-1.5">
          {slots.searchTrigger && (
            <>
              <slots.searchTrigger.full
                hideIfDisabled
                className="hidden min-w-[220px] rounded-full border border-[var(--color-divider)] bg-[rgba(28,28,30,0.92)] ps-2.5 lg:flex"
              />
              <slots.searchTrigger.sm
                hideIfDisabled
                className={cn(
                  buttonVariants({
                    variant: "ghost",
                    size: "icon-sm",
                    className:
                      "rounded-full text-[var(--color-text-secondary)] hover:bg-[var(--color-hover)] hover:text-[var(--color-text-primary)] lg:hidden",
                  })
                )}
              />
            </>
          )}

          {baseOptions.githubUrl && (
            <a
              href={baseOptions.githubUrl}
              target="_blank"
              rel="noreferrer noopener"
              aria-label="GitHub repository"
              className={cn(
                buttonVariants({
                  variant: "ghost",
                  size: "icon-sm",
                  className:
                    "rounded-full text-[var(--color-text-secondary)] hover:bg-[var(--color-hover)] hover:text-[var(--color-text-primary)]",
                })
              )}
            >
              <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.14em]">
                GH
              </span>
            </a>
          )}
        </div>
      </div>

      <div className="border-t border-[var(--color-divider)] md:hidden">
        <nav className="mx-auto flex w-full max-w-[var(--site-layout-width)] items-center gap-1 overflow-x-auto px-4 py-2">
          {primaryLinks.map(link => (
            <HeaderLink key={link.href} href={link.href} label={link.label} pathname={pathname} />
          ))}
        </nav>
      </div>
    </header>
  );
}
