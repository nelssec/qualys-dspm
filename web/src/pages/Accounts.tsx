import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Server,
  Plus,
  Trash2,
  Play,
  CheckCircle,
  XCircle,
  Clock
} from 'lucide-react';
import { getAccounts, createAccount, deleteAccount, triggerScan } from '../api/client';
import type { Provider, ScanType } from '../types';
import clsx from 'clsx';
import { formatDistanceToNow } from 'date-fns';

const PROVIDER_ICONS = {
  AWS: '🔶',
  AZURE: '🔷',
  GCP: '🔴',
};

export function Accounts() {
  const [showAddModal, setShowAddModal] = useState(false);
  const queryClient = useQueryClient();

  const { data: accounts, isLoading } = useQuery({
    queryKey: ['accounts'],
    queryFn: getAccounts,
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });

  const scanMutation = useMutation({
    mutationFn: ({ accountId, scanType }: { accountId: string; scanType: ScanType }) =>
      triggerScan(accountId, scanType),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this account?')) {
      deleteMutation.mutate(id);
    }
  };

  const handleScan = (accountId: string) => {
    scanMutation.mutate({ accountId, scanType: 'FULL' });
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Cloud Accounts</h1>
        <button
          onClick={() => setShowAddModal(true)}
          className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-primary-600 hover:bg-primary-700"
        >
          <Plus className="h-5 w-5 mr-2" />
          Add Account
        </button>
      </div>

      {/* Accounts Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {isLoading ? (
          <div className="col-span-full flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
          </div>
        ) : accounts && accounts.length > 0 ? (
          accounts.map((account) => (
            <div
              key={account.id}
              className="bg-white rounded-lg shadow overflow-hidden"
            >
              <div className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center">
                    <span className="text-2xl mr-2">
                      {PROVIDER_ICONS[account.provider]}
                    </span>
                    <div>
                      <h3 className="text-lg font-medium text-gray-900">
                        {account.display_name || account.external_id}
                      </h3>
                      <p className="text-sm text-gray-500">{account.provider}</p>
                    </div>
                  </div>
                  <span
                    className={clsx(
                      'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
                      account.status === 'active'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-red-100 text-red-800'
                    )}
                  >
                    {account.status === 'active' ? (
                      <CheckCircle className="h-3 w-3 mr-1" />
                    ) : (
                      <XCircle className="h-3 w-3 mr-1" />
                    )}
                    {account.status}
                  </span>
                </div>

                <div className="space-y-2 text-sm text-gray-600">
                  <p>
                    <span className="font-medium">Account ID:</span>{' '}
                    {account.external_id}
                  </p>
                  <p>
                    <span className="font-medium">Last Scan:</span>{' '}
                    {account.last_scan_at
                      ? formatDistanceToNow(new Date(account.last_scan_at), {
                          addSuffix: true,
                        })
                      : 'Never'}
                  </p>
                </div>
              </div>

              <div className="bg-gray-50 px-6 py-3 flex justify-between">
                <button
                  onClick={() => handleScan(account.id)}
                  disabled={scanMutation.isPending}
                  className="inline-flex items-center text-sm text-primary-600 hover:text-primary-800"
                >
                  <Play className="h-4 w-4 mr-1" />
                  Scan Now
                </button>
                <button
                  onClick={() => handleDelete(account.id)}
                  disabled={deleteMutation.isPending}
                  className="inline-flex items-center text-sm text-red-600 hover:text-red-800"
                >
                  <Trash2 className="h-4 w-4 mr-1" />
                  Delete
                </button>
              </div>
            </div>
          ))
        ) : (
          <div className="col-span-full bg-white rounded-lg shadow p-8 text-center text-gray-500">
            <Server className="h-12 w-12 mx-auto mb-4" />
            <p>No accounts connected</p>
            <button
              onClick={() => setShowAddModal(true)}
              className="mt-4 text-primary-600 hover:text-primary-800"
            >
              Add your first account
            </button>
          </div>
        )}
      </div>

      {/* Add Account Modal */}
      {showAddModal && (
        <AddAccountModal onClose={() => setShowAddModal(false)} />
      )}
    </div>
  );
}

function AddAccountModal({ onClose }: { onClose: () => void }) {
  const [provider, setProvider] = useState<Provider>('AWS');
  const [externalId, setExternalId] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [roleArn, setRoleArn] = useState('');

  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: createAccount,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
      onClose();
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate({
      provider,
      external_id: externalId,
      display_name: displayName,
      connector_config: {
        role_arn: roleArn,
      },
    });
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">
            Add Cloud Account
          </h2>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Provider
              </label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value as Provider)}
                className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
              >
                <option value="AWS">AWS</option>
                <option value="AZURE">Azure</option>
                <option value="GCP">GCP</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                {provider === 'AWS'
                  ? 'Account ID'
                  : provider === 'AZURE'
                  ? 'Subscription ID'
                  : 'Project ID'}
              </label>
              <input
                type="text"
                value={externalId}
                onChange={(e) => setExternalId(e.target.value)}
                required
                className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Display Name
              </label>
              <input
                type="text"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
              />
            </div>

            {provider === 'AWS' && (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Role ARN
                </label>
                <input
                  type="text"
                  value={roleArn}
                  onChange={(e) => setRoleArn(e.target.value)}
                  placeholder="arn:aws:iam::123456789012:role/DSPMRole"
                  className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-primary-500"
                />
              </div>
            )}

            <div className="flex justify-end space-x-3 pt-4">
              <button
                type="button"
                onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={mutation.isPending}
                className="px-4 py-2 text-sm font-medium text-white bg-primary-600 hover:bg-primary-700 rounded-lg disabled:opacity-50"
              >
                {mutation.isPending ? 'Adding...' : 'Add Account'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
