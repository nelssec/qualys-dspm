import { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Server,
  FolderOpen,
  AlertTriangle,
  Clock,
  FileText,
  Settings,
  ShieldCheck,
  Shield
} from 'lucide-react';
import clsx from 'clsx';

interface LayoutProps {
  children: ReactNode;
}

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Accounts', href: '/accounts', icon: Server },
  { name: 'Assets', href: '/assets', icon: FolderOpen },
  { name: 'Findings', href: '/findings', icon: AlertTriangle },
];

const management = [
  { name: 'Scheduled Jobs', href: '/jobs', icon: Clock },
  { name: 'Classification Rules', href: '/rules', icon: ShieldCheck },
  { name: 'Reports', href: '/reports', icon: FileText },
];

export function Layout({ children }: LayoutProps) {
  const location = useLocation();

  return (
    <div className="min-h-screen bg-qualys-bg">
      <div className="fixed inset-y-0 left-0 w-56 bg-qualys-sidebar flex flex-col">
        <div className="flex h-14 items-center px-4 border-b border-qualys-sidebar-hover">
          <Shield className="h-7 w-7 text-qualys-accent" />
          <div className="ml-2">
            <span className="text-base font-medium text-white">Qualys</span>
            <span className="text-[10px] text-qualys-accent ml-1 uppercase tracking-wider">DSPM</span>
          </div>
        </div>

        <nav className="flex-1 py-4 px-2 overflow-y-auto">
          <div className="mb-1 px-3 text-[10px] font-medium text-qualys-text-muted uppercase tracking-wider">
            Overview
          </div>
          {navigation.map((item) => {
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.name}
                to={item.href}
                className={clsx(
                  'flex items-center px-3 py-2 mt-0.5 rounded text-[13px] transition-colors',
                  isActive
                    ? 'bg-qualys-sidebar-active text-white'
                    : 'text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white'
                )}
              >
                <item.icon className="mr-3 h-4 w-4" />
                {item.name}
              </Link>
            );
          })}

          <div className="mt-6 mb-1 px-3 text-[10px] font-medium text-qualys-text-muted uppercase tracking-wider">
            Management
          </div>
          {management.map((item) => {
            const isActive = location.pathname === item.href;
            return (
              <Link
                key={item.name}
                to={item.href}
                className={clsx(
                  'flex items-center px-3 py-2 mt-0.5 rounded text-[13px] transition-colors',
                  isActive
                    ? 'bg-qualys-sidebar-active text-white'
                    : 'text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white'
                )}
              >
                <item.icon className="mr-3 h-4 w-4" />
                {item.name}
              </Link>
            );
          })}
        </nav>

        <div className="p-2 border-t border-qualys-sidebar-hover">
          <Link
            to="/settings"
            className={clsx(
              'flex items-center px-3 py-2 rounded text-[13px] transition-colors',
              location.pathname === '/settings'
                ? 'bg-qualys-sidebar-active text-white'
                : 'text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white'
            )}
          >
            <Settings className="mr-3 h-4 w-4" />
            Settings
          </Link>
        </div>
      </div>

      <div className="pl-56 min-w-0 overflow-x-hidden">
        <main className="py-5 px-6 max-w-full overflow-hidden">
          {children}
        </main>
      </div>
    </div>
  );
}
