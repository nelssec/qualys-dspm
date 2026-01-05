import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  FolderOpen,
  Globe,
  Lock,
  Search,
  Filter,
  ChevronRight
} from 'lucide-react';
import { getAssets } from '../api/client';
import type { Sensitivity } from '../types';
import clsx from 'clsx';

const SENSITIVITY_BADGES = {
  CRITICAL: 'bg-red-100 text-red-800',
  HIGH: 'bg-orange-100 text-orange-800',
  MEDIUM: 'bg-yellow-100 text-yellow-800',
  LOW: 'bg-green-100 text-green-800',
  UNKNOWN: 'bg-gray-100 text-gray-800',
};

export function Assets() {
  const [search, setSearch] = useState('');
  const [sensitivityFilter, setSensitivityFilter] = useState<Sensitivity | ''>('');
  const [publicOnly, setPublicOnly] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ['assets', sensitivityFilter, publicOnly],
    queryFn: () =>
      getAssets({
        sensitivity: sensitivityFilter || undefined,
        public_only: publicOnly || undefined,
        limit: 100,
      }),
  });

  const filteredAssets = data?.assets.filter((asset) =>
    asset.name.toLowerCase().includes(search.toLowerCase()) ||
    asset.resource_arn.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Data Assets</h1>
        <div className="text-sm text-gray-500">
          {data?.total || 0} total assets
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
                placeholder="Search assets..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>

          <select
            value={sensitivityFilter}
            onChange={(e) => setSensitivityFilter(e.target.value as Sensitivity | '')}
            className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
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
              className="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span className="ml-2 text-sm text-gray-700">Public only</span>
          </label>
        </div>
      </div>

      {/* Assets Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
          </div>
        ) : filteredAssets && filteredAssets.length > 0 ? (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Asset
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Sensitivity
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Classifications
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="relative px-6 py-3">
                  <span className="sr-only">View</span>
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredAssets.map((asset) => (
                <tr key={asset.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <div className="flex items-center">
                      <FolderOpen className="h-5 w-5 text-gray-400 mr-3" />
                      <div>
                        <div className="text-sm font-medium text-gray-900">
                          {asset.name}
                        </div>
                        <div className="text-sm text-gray-500 truncate max-w-md">
                          {asset.resource_arn}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm text-gray-900">
                      {asset.resource_type.replace('_', ' ')}
                    </span>
                    <div className="text-xs text-gray-500">{asset.region}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={clsx(
                        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
                        SENSITIVITY_BADGES[asset.sensitivity_level]
                      )}
                    >
                      {asset.sensitivity_level}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-900">
                      {asset.classification_count} findings
                    </div>
                    <div className="text-xs text-gray-500">
                      {asset.data_categories?.join(', ') || 'None'}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      {asset.public_access ? (
                        <span className="inline-flex items-center text-red-600">
                          <Globe className="h-4 w-4 mr-1" />
                          Public
                        </span>
                      ) : (
                        <span className="inline-flex items-center text-green-600">
                          <Lock className="h-4 w-4 mr-1" />
                          Private
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <Link
                      to={`/assets/${asset.id}`}
                      className="text-primary-600 hover:text-primary-900"
                    >
                      <ChevronRight className="h-5 w-5" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="flex flex-col items-center justify-center h-64 text-gray-500">
            <FolderOpen className="h-12 w-12 mb-4" />
            <p>No assets found</p>
          </div>
        )}
      </div>
    </div>
  );
}
