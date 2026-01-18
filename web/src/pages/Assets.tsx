import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  FolderOpen,
  Globe,
  Lock,
  Search,
  ChevronRight
} from 'lucide-react';
import { getAssets } from '../api/client';
import type { Sensitivity } from '../types';
import clsx from 'clsx';

const SENSITIVITY_BADGES = {
  CRITICAL: 'bg-severity-critical/10 text-severity-critical border-severity-critical/20',
  HIGH: 'bg-severity-high/10 text-severity-high border-severity-high/20',
  MEDIUM: 'bg-severity-medium/10 text-severity-medium border-severity-medium/20',
  LOW: 'bg-severity-low/10 text-severity-low border-severity-low/20',
  UNKNOWN: 'bg-severity-info/10 text-severity-info border-severity-info/20',
};

const PAGE_SIZE_OPTIONS = [25, 50, 100, 250, 0] as const;

export function Assets() {
  const [search, setSearch] = useState('');
  const [sensitivityFilter, setSensitivityFilter] = useState<Sensitivity | ''>('');
  const [publicOnly, setPublicOnly] = useState(false);
  const [pageSize, setPageSize] = useState<number>(50);
  const [currentPage, setCurrentPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ['assets', sensitivityFilter, publicOnly, pageSize, currentPage],
    queryFn: () =>
      getAssets({
        sensitivity: sensitivityFilter || undefined,
        public_only: publicOnly || undefined,
        limit: pageSize === 0 ? 10000 : pageSize,
        offset: pageSize === 0 ? 0 : (currentPage - 1) * pageSize,
      }),
  });

  const handleFilterChange = (newFilter: Sensitivity | '') => {
    setSensitivityFilter(newFilter);
    setCurrentPage(1);
  };

  const handlePageSizeChange = (newSize: number) => {
    setPageSize(newSize);
    setCurrentPage(1);
  };

  const filteredAssets = data?.assets.filter((asset) =>
    asset.name.toLowerCase().includes(search.toLowerCase()) ||
    asset.resource_arn.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-lg font-medium text-qualys-text-primary">Data Assets</h1>
        <div className="text-xs text-qualys-text-muted">
          {data?.total || 0} total assets
        </div>
      </div>

      <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-4 mb-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-qualys-text-muted" />
              <input
                type="text"
                placeholder="Search assets..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-9 pr-4 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>

          <select
            value={sensitivityFilter}
            onChange={(e) => handleFilterChange(e.target.value as Sensitivity | '')}
            className="px-3 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500"
          >
            <option value="">All Sensitivities</option>
            <option value="CRITICAL">Critical</option>
            <option value="HIGH">High</option>
            <option value="MEDIUM">Medium</option>
            <option value="LOW">Low</option>
          </select>

          <label className="flex items-center">
            <input
              type="checkbox"
              checked={publicOnly}
              onChange={(e) => setPublicOnly(e.target.checked)}
              className="rounded border-qualys-border text-primary-500 focus:ring-primary-500"
            />
            <span className="ml-2 text-sm text-qualys-text-secondary">Public only</span>
          </label>
        </div>
      </div>

      <div className="bg-white border border-qualys-border rounded shadow-qualys-sm overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500" />
          </div>
        ) : filteredAssets && filteredAssets.length > 0 ? (
          <table className="min-w-full divide-y divide-qualys-border">
            <thead className="bg-qualys-bg">
              <tr>
                <th className="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase tracking-wider">
                  Asset
                </th>
                <th className="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase tracking-wider">
                  Type
                </th>
                <th className="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase tracking-wider">
                  Sensitivity
                </th>
                <th className="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase tracking-wider">
                  Classifications
                </th>
                <th className="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase tracking-wider">
                  Status
                </th>
                <th className="relative px-4 py-3">
                  <span className="sr-only">View</span>
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-qualys-border">
              {filteredAssets.map((asset) => (
                <tr key={asset.id} className="hover:bg-qualys-bg transition-colors">
                  <td className="px-4 py-3">
                    <div className="flex items-center">
                      <FolderOpen className="h-4 w-4 text-qualys-text-muted mr-2.5" />
                      <div>
                        <div className="text-sm font-medium text-qualys-text-primary">
                          {asset.name}
                        </div>
                        <div className="text-[11px] text-qualys-text-muted truncate max-w-md">
                          {asset.resource_arn}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span className="text-sm text-qualys-text-primary">
                      {asset.resource_type.replace('_', ' ')}
                    </span>
                    <div className="text-[11px] text-qualys-text-muted">{asset.region}</div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <span
                      className={clsx(
                        'inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium border',
                        SENSITIVITY_BADGES[asset.sensitivity_level]
                      )}
                    >
                      {asset.sensitivity_level}
                    </span>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className="text-sm text-qualys-text-primary">
                      {asset.classification_count} findings
                    </div>
                    <div className="text-[11px] text-qualys-text-muted">
                      {asset.data_categories?.join(', ') || 'None'}
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      {asset.public_access ? (
                        <span className="inline-flex items-center text-xs text-severity-critical">
                          <Globe className="h-3.5 w-3.5 mr-1" />
                          Public
                        </span>
                      ) : (
                        <span className="inline-flex items-center text-xs text-severity-low">
                          <Lock className="h-3.5 w-3.5 mr-1" />
                          Private
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
                    <Link
                      to={`/assets/${asset.id}`}
                      className="text-primary-500 hover:text-primary-700"
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="flex flex-col items-center justify-center h-64 text-qualys-text-muted">
            <FolderOpen className="h-10 w-10 mb-3" />
            <p className="text-sm">No assets found</p>
          </div>
        )}

        {data && data.total > 0 && (
          <div className="px-4 py-3 border-t border-qualys-border flex items-center justify-between bg-white">
            <div className="flex items-center gap-4">
              <div className="text-xs text-qualys-text-muted">
                {pageSize === 0 ? (
                  `Showing all ${data.total} assets`
                ) : (
                  `Showing ${Math.min((currentPage - 1) * pageSize + 1, data.total)}-${Math.min(currentPage * pageSize, data.total)} of ${data.total} assets`
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
