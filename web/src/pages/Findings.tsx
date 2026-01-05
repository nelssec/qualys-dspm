import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  Search,
  CheckCircle,
  XCircle,
  Clock,
  ChevronDown
} from 'lucide-react';
import { getFindings, updateFindingStatus } from '../api/client';
import type { FindingSeverity, FindingStatus } from '../types';
import clsx from 'clsx';
import { formatDistanceToNow } from 'date-fns';

const SEVERITY_BADGES = {
  CRITICAL: 'bg-red-100 text-red-800 border-red-200',
  HIGH: 'bg-orange-100 text-orange-800 border-orange-200',
  MEDIUM: 'bg-yellow-100 text-yellow-800 border-yellow-200',
  LOW: 'bg-green-100 text-green-800 border-green-200',
  INFO: 'bg-gray-100 text-gray-800 border-gray-200',
};

const STATUS_ICONS = {
  open: AlertTriangle,
  in_progress: Clock,
  resolved: CheckCircle,
  suppressed: XCircle,
  false_positive: XCircle,
};

export function Findings() {
  const [search, setSearch] = useState('');
  const [severityFilter, setSeverityFilter] = useState<FindingSeverity | ''>('');
  const [statusFilter, setStatusFilter] = useState<FindingStatus | ''>('');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['findings', severityFilter, statusFilter],
    queryFn: () =>
      getFindings({
        severity: severityFilter || undefined,
        status: statusFilter || undefined,
        limit: 100,
      }),
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: FindingStatus }) =>
      updateFindingStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['findings'] });
    },
  });

  const filteredFindings = data?.findings.filter((finding) =>
    finding.title.toLowerCase().includes(search.toLowerCase()) ||
    finding.finding_type.toLowerCase().includes(search.toLowerCase())
  );

  const handleStatusChange = (findingId: string, newStatus: FindingStatus) => {
    updateStatusMutation.mutate({ id: findingId, status: newStatus });
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Security Findings</h1>
        <div className="text-sm text-gray-500">
          {data?.total || 0} total findings
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow p-4 mb-6">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
              <input
                type="text"
                placeholder="Search findings..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>

          <select
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value as FindingSeverity | '')}
            className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
          >
            <option value="">All Severities</option>
            <option value="CRITICAL">Critical</option>
            <option value="HIGH">High</option>
            <option value="MEDIUM">Medium</option>
            <option value="LOW">Low</option>
            <option value="INFO">Info</option>
          </select>

          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as FindingStatus | '')}
            className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
          >
            <option value="">All Statuses</option>
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="resolved">Resolved</option>
            <option value="suppressed">Suppressed</option>
            <option value="false_positive">False Positive</option>
          </select>
        </div>
      </div>

      {/* Findings List */}
      <div className="space-y-4">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
          </div>
        ) : filteredFindings && filteredFindings.length > 0 ? (
          filteredFindings.map((finding) => {
            const StatusIcon = STATUS_ICONS[finding.status];
            const isExpanded = expandedId === finding.id;

            return (
              <div
                key={finding.id}
                className="bg-white rounded-lg shadow overflow-hidden"
              >
                <div
                  className="p-4 cursor-pointer hover:bg-gray-50"
                  onClick={() => setExpandedId(isExpanded ? null : finding.id)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                      <span
                        className={clsx(
                          'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border',
                          SEVERITY_BADGES[finding.severity]
                        )}
                      >
                        {finding.severity}
                      </span>
                      <div>
                        <h3 className="text-sm font-medium text-gray-900">
                          {finding.title}
                        </h3>
                        <p className="text-xs text-gray-500">
                          {finding.finding_type} •{' '}
                          {formatDistanceToNow(new Date(finding.created_at), {
                            addSuffix: true,
                          })}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-4">
                      <span
                        className={clsx(
                          'inline-flex items-center text-sm',
                          finding.status === 'open'
                            ? 'text-red-600'
                            : finding.status === 'resolved'
                            ? 'text-green-600'
                            : 'text-gray-600'
                        )}
                      >
                        <StatusIcon className="h-4 w-4 mr-1" />
                        {finding.status.replace('_', ' ')}
                      </span>
                      <ChevronDown
                        className={clsx(
                          'h-5 w-5 text-gray-400 transition-transform',
                          isExpanded && 'transform rotate-180'
                        )}
                      />
                    </div>
                  </div>
                </div>

                {isExpanded && (
                  <div className="px-4 pb-4 border-t bg-gray-50">
                    <div className="pt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <h4 className="text-sm font-medium text-gray-700 mb-2">
                          Description
                        </h4>
                        <p className="text-sm text-gray-600">
                          {finding.description}
                        </p>
                      </div>
                      <div>
                        <h4 className="text-sm font-medium text-gray-700 mb-2">
                          Remediation
                        </h4>
                        <p className="text-sm text-gray-600">
                          {finding.remediation}
                        </p>
                      </div>
                    </div>

                    {finding.compliance_frameworks?.length > 0 && (
                      <div className="mt-4">
                        <h4 className="text-sm font-medium text-gray-700 mb-2">
                          Compliance Frameworks
                        </h4>
                        <div className="flex flex-wrap gap-2">
                          {finding.compliance_frameworks.map((framework) => (
                            <span
                              key={framework}
                              className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800"
                            >
                              {framework}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                    <div className="mt-4 flex items-center space-x-2">
                      <span className="text-sm text-gray-500">Change status:</span>
                      {finding.status !== 'resolved' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'resolved')}
                          className="px-3 py-1 text-sm bg-green-100 text-green-700 rounded hover:bg-green-200"
                        >
                          Resolve
                        </button>
                      )}
                      {finding.status !== 'suppressed' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'suppressed')}
                          className="px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded hover:bg-gray-200"
                        >
                          Suppress
                        </button>
                      )}
                      {finding.status !== 'false_positive' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'false_positive')}
                          className="px-3 py-1 text-sm bg-yellow-100 text-yellow-700 rounded hover:bg-yellow-200"
                        >
                          False Positive
                        </button>
                      )}
                      {finding.status !== 'open' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'open')}
                          className="px-3 py-1 text-sm bg-red-100 text-red-700 rounded hover:bg-red-200"
                        >
                          Reopen
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            );
          })
        ) : (
          <div className="bg-white rounded-lg shadow p-8 text-center text-gray-500">
            <AlertTriangle className="h-12 w-12 mx-auto mb-4" />
            <p>No findings found</p>
          </div>
        )}
      </div>
    </div>
  );
}
