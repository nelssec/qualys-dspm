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
  CRITICAL: 'bg-severity-critical/10 text-severity-critical border-severity-critical/20',
  HIGH: 'bg-severity-high/10 text-severity-high border-severity-high/20',
  MEDIUM: 'bg-severity-medium/10 text-severity-medium border-severity-medium/20',
  LOW: 'bg-severity-low/10 text-severity-low border-severity-low/20',
  INFO: 'bg-severity-info/10 text-severity-info border-severity-info/20',
};

const STATUS_ICONS = {
  open: AlertTriangle,
  in_progress: Clock,
  resolved: CheckCircle,
  suppressed: XCircle,
  false_positive: XCircle,
};

const PAGE_SIZE_OPTIONS = [25, 50, 100, 250, 0] as const;

export function Findings() {
  const [search, setSearch] = useState('');
  const [severityFilter, setSeverityFilter] = useState<FindingSeverity | ''>('');
  const [statusFilter, setStatusFilter] = useState<FindingStatus | ''>('');
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [pageSize, setPageSize] = useState<number>(50);
  const [currentPage, setCurrentPage] = useState(1);

  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['findings', severityFilter, statusFilter, pageSize, currentPage],
    queryFn: () =>
      getFindings({
        severity: severityFilter || undefined,
        status: statusFilter || undefined,
        limit: pageSize === 0 ? 10000 : pageSize,
        offset: pageSize === 0 ? 0 : (currentPage - 1) * pageSize,
      }),
  });

  const handleFilterChange = (type: 'severity' | 'status', value: string) => {
    if (type === 'severity') {
      setSeverityFilter(value as FindingSeverity | '');
    } else {
      setStatusFilter(value as FindingStatus | '');
    }
    setCurrentPage(1);
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setCurrentPage(1);
  };

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
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-lg font-medium text-qualys-text-primary">Security Findings</h1>
        <div className="text-xs text-qualys-text-muted">
          {data?.total || 0} total findings
        </div>
      </div>

      <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-4 mb-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-qualys-text-muted" />
              <input
                type="text"
                placeholder="Search findings..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-9 pr-4 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>

          <select
            value={severityFilter}
            onChange={(e) => handleFilterChange('severity', e.target.value)}
            className="px-3 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500"
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
            onChange={(e) => handleFilterChange('status', e.target.value)}
            className="px-3 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500"
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

      <div className="space-y-3">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500" />
          </div>
        ) : filteredFindings && filteredFindings.length > 0 ? (
          filteredFindings.map((finding) => {
            const StatusIcon = STATUS_ICONS[finding.status];
            const isExpanded = expandedId === finding.id;

            return (
              <div
                key={finding.id}
                className="bg-white border border-qualys-border rounded shadow-qualys-sm overflow-hidden"
              >
                <div
                  className="p-4 cursor-pointer hover:bg-qualys-bg transition-colors"
                  onClick={() => setExpandedId(isExpanded ? null : finding.id)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      <span
                        className={clsx(
                          'inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium border',
                          SEVERITY_BADGES[finding.severity]
                        )}
                      >
                        {finding.severity}
                      </span>
                      <div>
                        <h3 className="text-sm font-medium text-qualys-text-primary">
                          {finding.title}
                        </h3>
                        <p className="text-[11px] text-qualys-text-muted">
                          {finding.finding_type} •{' '}
                          {formatDistanceToNow(new Date(finding.created_at), {
                            addSuffix: true,
                          })}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-3">
                      <span
                        className={clsx(
                          'inline-flex items-center text-xs',
                          finding.status === 'open'
                            ? 'text-severity-critical'
                            : finding.status === 'resolved'
                            ? 'text-severity-low'
                            : 'text-qualys-text-secondary'
                        )}
                      >
                        <StatusIcon className="h-3.5 w-3.5 mr-1" />
                        {finding.status.replace('_', ' ')}
                      </span>
                      <ChevronDown
                        className={clsx(
                          'h-4 w-4 text-qualys-text-muted transition-transform',
                          isExpanded && 'transform rotate-180'
                        )}
                      />
                    </div>
                  </div>
                </div>

                {isExpanded && (
                  <div className="px-4 pb-4 border-t border-qualys-border bg-qualys-bg">
                    <div className="pt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <h4 className="text-xs font-medium text-qualys-text-secondary mb-1">
                          Description
                        </h4>
                        <p className="text-sm text-qualys-text-primary">
                          {finding.description}
                        </p>
                      </div>
                      <div>
                        <h4 className="text-xs font-medium text-qualys-text-secondary mb-1">
                          Remediation
                        </h4>
                        <p className="text-sm text-qualys-text-primary">
                          {finding.remediation}
                        </p>
                      </div>
                    </div>

                    {finding.compliance_frameworks?.length > 0 && (
                      <div className="mt-4">
                        <h4 className="text-xs font-medium text-qualys-text-secondary mb-2">
                          Compliance Frameworks
                        </h4>
                        <div className="flex flex-wrap gap-2 max-w-full overflow-hidden">
                          {finding.compliance_frameworks.map((framework) => (
                            <span
                              key={framework}
                              className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-primary-50 text-primary-700 border border-primary-100 truncate max-w-[200px]"
                            >
                              {framework}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}

                    <div className="mt-4 flex items-center space-x-2">
                      <span className="text-xs text-qualys-text-muted">Change status:</span>
                      {finding.status !== 'resolved' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'resolved')}
                          className="px-2.5 py-1 text-xs bg-severity-low/10 text-severity-low border border-severity-low/20 rounded hover:bg-severity-low/20 transition-colors"
                        >
                          Resolve
                        </button>
                      )}
                      {finding.status !== 'suppressed' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'suppressed')}
                          className="px-2.5 py-1 text-xs bg-qualys-bg text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-border-light transition-colors"
                        >
                          Suppress
                        </button>
                      )}
                      {finding.status !== 'false_positive' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'false_positive')}
                          className="px-2.5 py-1 text-xs bg-severity-medium/10 text-severity-medium border border-severity-medium/20 rounded hover:bg-severity-medium/20 transition-colors"
                        >
                          False Positive
                        </button>
                      )}
                      {finding.status !== 'open' && (
                        <button
                          onClick={() => handleStatusChange(finding.id, 'open')}
                          className="px-2.5 py-1 text-xs bg-severity-critical/10 text-severity-critical border border-severity-critical/20 rounded hover:bg-severity-critical/20 transition-colors"
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
          <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-8 text-center text-qualys-text-muted">
            <AlertTriangle className="h-10 w-10 mx-auto mb-3 text-qualys-text-muted" />
            <p className="text-sm">No findings found</p>
          </div>
        )}

        {data && data.total > 0 && (
          <div className="mt-4 px-4 py-3 bg-white border border-qualys-border rounded shadow-qualys-sm flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="text-xs text-qualys-text-muted">
                {pageSize === 0 ? (
                  `Showing all ${data.total} findings`
                ) : (
                  `Showing ${Math.min((currentPage - 1) * pageSize + 1, data.total)}-${Math.min(currentPage * pageSize, data.total)} of ${data.total} findings`
                )}
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-qualys-text-muted">Show:</span>
                <select
                  value={pageSize}
                  onChange={(e) => handlePageSizeChange(Number(e.target.value))}
                  className="px-2 py-1 text-xs border border-qualys-border rounded focus:ring-1 focus:ring-primary-500"
                >
                  {PAGE_SIZE_OPTIONS.map((size) => (
                    <option key={size} value={size}>
                      {size === 0 ? 'All' : size}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            {pageSize !== 0 && (
              <div className="flex gap-1">
                <button
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  className="px-3 py-1 text-xs border border-qualys-border rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-qualys-bg"
                >
                  Previous
                </button>
                {(() => {
                  const totalPages = Math.ceil(data.total / pageSize);
                  const pages: (number | string)[] = [];

                  if (totalPages <= 7) {
                    for (let i = 1; i <= totalPages; i++) pages.push(i);
                  } else {
                    pages.push(1);
                    if (currentPage > 3) pages.push('...');
                    for (let i = Math.max(2, currentPage - 1); i <= Math.min(totalPages - 1, currentPage + 1); i++) {
                      pages.push(i);
                    }
                    if (currentPage < totalPages - 2) pages.push('...');
                    pages.push(totalPages);
                  }

                  return pages.map((page, idx) => (
                    typeof page === 'number' ? (
                      <button
                        key={idx}
                        onClick={() => setCurrentPage(page)}
                        className={clsx(
                          'px-3 py-1 text-xs rounded',
                          page === currentPage
                            ? 'bg-primary-500 text-white'
                            : 'border border-qualys-border hover:bg-qualys-bg'
                        )}
                      >
                        {page}
                      </button>
                    ) : (
                      <span key={idx} className="px-2 py-1 text-xs text-qualys-text-muted">...</span>
                    )
                  ));
                })()}
                <button
                  onClick={() => setCurrentPage(p => Math.min(Math.ceil(data.total / pageSize), p + 1))}
                  disabled={currentPage >= Math.ceil(data.total / pageSize)}
                  className="px-3 py-1 text-xs border border-qualys-border rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-qualys-bg"
                >
                  Next
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
