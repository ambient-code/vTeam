import * as React from 'react';
import { cn } from '@/lib/utils';

type SidebarLayoutProps = {
  sidebar: React.ReactNode;
  children: React.ReactNode;
  sidebarWidth?: string;
  className?: string;
};

export function SidebarLayout({
  sidebar,
  children,
  sidebarWidth = '16rem',
  className,
}: SidebarLayoutProps) {
  return (
    <div className={cn('flex min-h-screen', className)}>
      <aside
        className="hidden md:block border-r bg-muted/10"
        style={{ width: sidebarWidth }}
      >
        <div className="sticky top-0 h-screen overflow-y-auto">
          {sidebar}
        </div>
      </aside>
      <main className="flex-1">
        {children}
      </main>
    </div>
  );
}

type MobileSidebarProps = {
  sidebar: React.ReactNode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function MobileSidebar({ sidebar, open, onOpenChange }: MobileSidebarProps) {
  React.useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [open]);

  if (!open) return null;

  return (
    <div className="md:hidden">
      <div
        className="fixed inset-0 z-40 bg-black/50"
        onClick={() => onOpenChange(false)}
      />
      <aside className="fixed inset-y-0 left-0 z-50 w-64 bg-background border-r">
        <div className="h-full overflow-y-auto">
          {sidebar}
        </div>
      </aside>
    </div>
  );
}
