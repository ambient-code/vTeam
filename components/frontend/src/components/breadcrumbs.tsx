"use client";

/**
 * Breadcrumbs Component
 * Navigation breadcrumbs for hierarchical navigation
 */

import * as React from 'react';
import Link from 'next/link';
import { ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

export type BreadcrumbItem = {
  label: string;
  href?: string;
  icon?: React.ReactNode;
  rightIcon?: React.ReactNode;
};

export type BreadcrumbsProps = {
  items: BreadcrumbItem[];
  className?: string;
  separator?: React.ReactNode;
};

export function Breadcrumbs({
  items,
  className,
  separator = <ChevronRight className="h-4 w-4" />,
}: BreadcrumbsProps) {
  // Temporarily hiding Home from breadcrumbs
  // const allItems: BreadcrumbItem[] = showHome
  //   ? [{ label: 'Home', href: '/', icon: <Home className="h-4 w-4" /> }, ...items]
  //   : items;
  const allItems: BreadcrumbItem[] = items;

  return (
    <nav aria-label="Breadcrumb" className={cn('flex items-center space-x-1 text-sm', className)}>
      <ol className="flex items-center space-x-1">
        {allItems.map((item, index) => {
          const isLast = index === allItems.length - 1;

          return (
            <li key={index} className="flex items-center space-x-1">
              {index > 0 && (
                <span className="text-muted-foreground" aria-hidden="true">
                  {separator}
                </span>
              )}
              {isLast ? (
                <span
                  className="flex items-center gap-1.5 font-medium text-foreground"
                  aria-current="page"
                >
                  {item.icon}
                  {item.label}
                  {item.rightIcon}
                </span>
              ) : (
                <Link
                  href={item.href || '#'}
                  className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
                >
                  {item.icon}
                  {item.label}
                  {item.rightIcon}
                </Link>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

/**
 * Compact breadcrumbs that collapse middle items on mobile
 */
export function CompactBreadcrumbs({ items, className }: BreadcrumbsProps) {
  // Temporarily hiding Home from breadcrumbs
  // const allItems: BreadcrumbItem[] = showHome
  //   ? [{ label: 'Home', href: '/', icon: <Home className="h-4 w-4" /> }, ...items]
  //   : items;
  const allItems: BreadcrumbItem[] = items;

  // On mobile, show first, last, and ellipsis if there are many items
  const shouldCollapse = allItems.length > 3;

  return (
    <nav aria-label="Breadcrumb" className={cn('flex items-center space-x-1 text-sm', className)}>
      <ol className="flex items-center space-x-1">
        {shouldCollapse ? (
          <>
            {/* First item */}
            <li className="flex items-center space-x-1">
              <Link
                href={allItems[0].href || '#'}
                className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
              >
                {allItems[0].icon}
                <span className="hidden sm:inline">{allItems[0].label}</span>
              </Link>
            </li>

            {/* Separator */}
            <span className="text-muted-foreground" aria-hidden="true">
              <ChevronRight className="h-4 w-4" />
            </span>

            {/* Ellipsis on mobile, middle items on desktop */}
            <li className="flex items-center space-x-1">
              <span className="text-muted-foreground sm:hidden">...</span>
              <span className="hidden sm:flex sm:items-center sm:space-x-1">
                {allItems.slice(1, -1).map((item, index) => (
                  <React.Fragment key={index}>
                    {index > 0 && (
                      <ChevronRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    )}
                    <Link
                      href={item.href || '#'}
                      className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {item.icon}
                      {item.label}
                    </Link>
                  </React.Fragment>
                ))}
              </span>
            </li>

            {/* Separator */}
            <span className="text-muted-foreground" aria-hidden="true">
              <ChevronRight className="h-4 w-4" />
            </span>

            {/* Last item */}
            <li>
              <span className="flex items-center gap-1.5 font-medium text-foreground" aria-current="page">
                {allItems[allItems.length - 1].icon}
                {allItems[allItems.length - 1].label}
                {allItems[allItems.length - 1].rightIcon}
              </span>
            </li>
          </>
        ) : (
          <Breadcrumbs items={items} className={className} />
        )}
      </ol>
    </nav>
  );
}
