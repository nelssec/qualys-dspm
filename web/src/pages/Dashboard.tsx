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

const SEVERITY_COLORS = {
  CRITICAL: '#dc2626',
  HIGH: '#ea580c',
  MEDIUM: '#ca8a04',
  LOW: '#16a34a',
  INFO: '#6b7280',
};

const CATEGORY_COLORS = {
  PII: '#3b82f6',
  PHI: '#8b5cf6',
  PCI: '#ec4899',
  SECRETS: '#f97316',
  CUSTOM: '#6b7280',
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
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    );
  }

  const classificationData = classStats
    ? Object.entries(classStats).map(([name, value]) => ({
        name,
        value,
        color: CATEGORY_COLORS[name as keyof typeof CATEGORY_COLORS] || '#6b7280',
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
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h1>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
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

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Classification by Category */}
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Data Classification by Category
          </h2>
          {classificationData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={classificationData}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={100}
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
          ) : (
            <div className="flex items-center justify-center h-64 text-gray-500">
              No classification data available
            </div>
          )}
        </div>

        {/* Findings by Severity */}
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Findings by Severity
          </h2>
          {findingData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={findingData} layout="vertical">
                <XAxis type="number" />
                <YAxis type="category" dataKey="severity" width={80} />
                <Tooltip />
                <Legend />
                <Bar dataKey="open" name="Open" fill="#ef4444" />
                <Bar dataKey="resolved" name="Resolved" fill="#22c55e" />
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-64 text-gray-500">
              No finding data available
            </div>
          )}
        </div>
      </div>

      {/* Risk Summary */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          Risk Overview
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
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
    blue: 'bg-blue-50 text-blue-600',
    green: 'bg-green-50 text-green-600',
    orange: 'bg-orange-50 text-orange-600',
    purple: 'bg-purple-50 text-purple-600',
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between">
        <div className={clsx('p-3 rounded-lg', colorClasses[color])}>
          <Icon className="h-6 w-6" />
        </div>
        {alert && (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
            Alert
          </span>
        )}
      </div>
      <div className="mt-4">
        <p className="text-3xl font-bold text-gray-900">{value.toLocaleString()}</p>
        <p className="text-sm font-medium text-gray-500">{title}</p>
        <p className="text-xs text-gray-400 mt-1">{subtitle}</p>
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
    critical: 'bg-red-100 text-red-800 border-red-200',
    high: 'bg-orange-100 text-orange-800 border-orange-200',
    medium: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    low: 'bg-green-100 text-green-800 border-green-200',
  };

  return (
    <div className={clsx('p-4 rounded-lg border', severityClasses[severity])}>
      <div className="flex items-center">
        <Icon className="h-5 w-5 mr-2" />
        <span className="font-medium">{title}</span>
      </div>
      <p className="text-sm mt-1 opacity-75">{description}</p>
      <p className="text-2xl font-bold mt-2">{count}</p>
    </div>
  );
}
