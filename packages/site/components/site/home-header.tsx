"use client";

import { baseOptions } from "@/lib/layout.shared";
import { Logo, buttonVariants, cn } from "@agh/ui";
import { GithubLogo } from "@agh/ui/logos";
import { useHomeLayout } from "fumadocs-ui/layouts/home";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ComponentProps } from "react";

const primaryLinks = [
  { href: "/", label: "Home" },
  { href: "/runtime", label: "Runtime" },
  { href: "/protocol", label: "AGH Network" },
  { href: "/blog", label: "Blog" },
  { href: "/changelog", label: "Changelog" },
];

function isActive(pathname: string, href: string) {
  if (href === "/") {
    return pathname === href;
  }

  return pathname === href || pathname.startsWith(`${href}/`);
}

function HeaderLink({ href, label, pathname }: { href: string; label: string; pathname: string }) {
  const active = isActive(pathname, href);
  const current = active ? (pathname === href ? "page" : "location") : undefined;

  return (
    <Link
      href={href}
      aria-current={current}
      className={cn(
        "inline-flex items-center rounded-full px-3 py-1.5 text-sm text-(--muted) transition-colors hover:text-(--fg)",
        active && "bg-(--elevated) text-(--fg)"
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
        "sticky top-0 z-40 border-b border-(--line) bg-background/92 px-4 backdrop-blur-xl",
        props.className
      )}
    >
      <div className="mx-auto flex h-14 w-full max-w-(--site-layout-width) items-center gap-3 lg:gap-5">
        <Link href="/" aria-label="AGH home" className="shrink-0">
          <Logo variant="logo" decorative className="h-8 w-auto" />
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
                className="hidden min-w-[220px] rounded-full border border-(--line) bg-(--canvas-soft) ps-2.5 lg:flex"
              />
              <slots.searchTrigger.sm
                hideIfDisabled
                className={cn(
                  buttonVariants({
                    variant: "ghost",
                    size: "icon-sm",
                    className:
                      "rounded-full text-(--muted) hover:bg-(--hover) hover:text-(--fg) lg:hidden",
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
              aria-label="Compozy on GitHub"
              className={cn(
                buttonVariants({
                  variant: "ghost",
                  size: "icon-sm",
                  className: "rounded-full text-(--muted) hover:bg-(--hover) hover:text-(--fg)",
                })
              )}
            >
              <GithubLogo aria-hidden className="size-4" />
            </a>
          )}
        </div>
      </div>

      <div className="border-t border-(--line) md:hidden">
        <nav className="mx-auto flex w-full max-w-(--site-layout-width) items-center gap-1 overflow-x-auto px-4 py-2">
          {primaryLinks.map(link => (
            <HeaderLink key={link.href} href={link.href} label={link.label} pathname={pathname} />
          ))}
        </nav>
      </div>
    </header>
  );
}
