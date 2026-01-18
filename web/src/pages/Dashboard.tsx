import { useQuery } from '@tanstack/react-query';
import {
  Server,
  FolderOpen,
  AlertTriangle,
  ShieldAlert,
  Globe,
  Lock
} from 'lucide-react';
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend
} from 'recharts';
import { getDashboardSummary, getClassificationStats, getFindingStats } from '../api/client';
import clsx from 'clsx';

const CATEGORY_COLORS = {
  PII: '#1991e1',
  PHI: '#7c3aed',
  PCI: '#db2777',
  SECRETS: '#e85d04',
  CUSTOM: '#56707e',
};

export function Dashboard() {
  const { data: summary, isLoading: summaryLoading } = useQuery({
    queryKey: ['dashboard-summary'],
    queryFn: getDashboardSummary,
  });

  const { data: classStats } = useQuery({
    queryKey: ['classification-stats'],
    queryFn: () => getClassificationStats(),
  });

  const { data: findingStats } = useQuery({
    queryKey: ['finding-stats'],
    queryFn: () => getFindingStats(),
  });

  if (summaryLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500" />
      </div>
    );
  }

  const classificationData = classStats
    ? Object.entries(classStats).map(([name, value]) => ({
        name,
        value,
        color: CATEGORY_COLORS[name as keyof typeof CATEGORY_COLORS] || '#56707e',
      }))
    : [];

  const findingData = findingStats
    ? Object.entries(findingStats).map(([severity, statuses]) => ({
        severity,
        open: statuses['open'] || 0,
        resolved: statuses['resolved'] || 0,
      }))
    : [];

  return (
    <div>
      <h1 className="text-lg font-medium text-qualys-text-primary mb-5">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
          title="Cloud Accounts"
          value={summary?.accounts.total || 0}
          subtitle={`${summary?.accounts.active || 0} active`}
          icon={Server}
          color="blue"
        />
        <StatCard
          title="Data Assets"
          value={summary?.assets.total || 0}
          subtitle={`${summary?.assets.public || 0} public`}
          icon={FolderOpen}
          color="green"
          alert={summary?.assets.public ? true : false}
        />
        <StatCard
          title="Open Findings"
          value={summary?.findings.open || 0}
          subtitle={`${summary?.findings.critical || 0} critical`}
          icon={AlertTriangle}
          color="orange"
          alert={summary?.findings.critical ? true : false}
        />
        <StatCard
          title="Sensitive Data"
          value={summary?.classifications.total || 0}
          subtitle={`${summary?.assets.critical || 0} critical assets`}
          icon={ShieldAlert}
          color="purple"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
        <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-5 min-w-0 overflow-hidden">
          <h2 className="text-sm font-medium text-qualys-text-primary mb-4">
            Data Classification by Category
          </h2>
          {classificationData.length > 0 ? (
            <div className="h-[280px] max-h-[280px] overflow-hidden">
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie
                    data={classificationData}
                    cx="50%"
                    cy="50%"
                    innerRadius={55}
                    outerRadius={90}
                    paddingAngle={2}
                    dataKey="value"
                    label={({ name, percent }) =>
                      `${name} (${(percent * 100).toFixed(0)}%)`
                    }
                  >
                    {classificationData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="flex items-center justify-center h-64 text-qualys-text-muted text-sm">
              No classification data available
            </div>
          )}
        </div>

        <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-5 min-w-0 overflow-hidden">
          <h2 className="text-sm font-medium text-qualys-text-primary mb-4">
            Findings by Severity
          </h2>
          {findingData.length > 0 ? (
            <div className="h-[280px] max-h-[280px] overflow-hidden">
              <ResponsiveContainer width="100%" height={280}>
                <BarChart data={findingData} layout="vertical">
                  <XAxis type="number" tick={{ fontSize: 11 }} />
                  <YAxis type="category" dataKey="severity" width={70} tick={{ fontSize: 11 }} />
                  <Tooltip />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  <Bar dataKey="open" name="Open" fill="#c41230" />
                  <Bar dataKey="resolved" name="Resolved" fill="#2e7d32" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <div className="flex items-center justify-center h-64 text-qualys-text-muted text-sm">
              No finding data available
            </div>
          )}
        </div>
      </div>

      <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-5">
        <h2 className="text-sm font-medium text-qualys-text-primary mb-4">
          Risk Overview
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <RiskItem
            icon={Globe}
            title="Public Exposure"
            description="Assets accessible from the internet"
            count={summary?.assets.public || 0}
            severity={summary?.assets.public ? 'critical' : 'low'}
          />
          <RiskItem
            icon={Lock}
            title="Encryption Gaps"
            description="Assets without encryption at rest"
            count={0}
            severity="medium"
          />
          <RiskItem
            icon={ShieldAlert}
            title="Sensitive Data at Risk"
            description="Critical data with open findings"
            count={summary?.assets.critical || 0}
            severity={summary?.assets.critical ? 'high' : 'low'}
          />
        </div>
      </div>
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number;
  subtitle: string;
  icon: React.ElementType;
  color: 'blue' | 'green' | 'orange' | 'purple';
  alert?: boolean;
}

function StatCard({ title, value, subtitle, icon: Icon, color, alert }: StatCardProps) {
  const colorClasses = {
    blue: 'bg-primary-50 text-primary-600 border-primary-100',
    green: 'bg-green-50 text-green-600 border-green-100',
    orange: 'bg-orange-50 text-orange-600 border-orange-100',
    purple: 'bg-purple-50 text-purple-600 border-purple-100',
  };

  return (
    <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-4">
      <div className="flex items-center justify-between">
        <div className={clsx('p-2 rounded border', colorClasses[color])}>
          <Icon className="h-5 w-5" />
        </div>
        {alert && (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-severity-critical/10 text-severity-critical border border-severity-critical/20">
            Alert
          </span>
        )}
      </div>
      <div className="mt-3">
        <p className="text-2xl font-medium text-qualys-text-primary">{value.toLocaleString()}</p>
        <p className="text-xs font-medium text-qualys-text-secondary mt-0.5">{title}</p>
        <p className="text-[11px] text-qualys-text-muted mt-1">{subtitle}</p>
      </div>
    </div>
  );
}

interface RiskItemProps {
  icon: React.ElementType;
  title: string;
  description: string;
  count: number;
  severity: 'critical' | 'high' | 'medium' | 'low';
}

function RiskItem({ icon: Icon, title, description, count, severity }: RiskItemProps) {
  const severityClasses = {
    critical: 'bg-severity-critical/5 text-severity-critical border-severity-critical/20',
    high: 'bg-severity-high/5 text-severity-high border-severity-high/20',
    medium: 'bg-severity-medium/5 text-severity-medium border-severity-medium/20',
    low: 'bg-severity-low/5 text-severity-low border-severity-low/20',
  };

  return (
    <div className={clsx('p-4 rounded border', severityClasses[severity])}>
      <div className="flex items-center">
        <Icon className="h-4 w-4 mr-2" />
        <span className="text-sm font-medium">{title}</span>
      </div>
      <p className="text-[11px] mt-1 opacity-80">{description}</p>
      <p className="text-xl font-medium mt-2">{count}</p>
    </div>
  );
}
