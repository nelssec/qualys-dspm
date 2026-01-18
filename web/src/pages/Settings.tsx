import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Bell, Mail, MessageSquare, Save, Send } from 'lucide-react';
import { api } from '../api/client';

export function Settings() {
  const queryClient = useQueryClient();

  const { data: settings, isLoading } = useQuery({
    queryKey: ['notification-settings'],
    queryFn: api.getNotificationSettings,
  });

  const [formData, setFormData] = useState({
    slack_enabled: false,
    slack_webhook_url: '',
    slack_channel: '',
    email_enabled: false,
    email_recipients: '',
    min_severity: 'high',
  });

  useState(() => {
    if (settings) {
      setFormData({
        slack_enabled: settings.slack_enabled || false,
        slack_webhook_url: '',
        slack_channel: settings.slack_channel || '',
        email_enabled: settings.email_enabled || false,
        email_recipients: settings.email_recipients?.join(', ') || '',
        min_severity: settings.min_severity || 'high',
      });
    }
  });

  const updateMutation = useMutation({
    mutationFn: api.updateNotificationSettings,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notification-settings'] });
    },
  });

  const testMutation = useMutation({
    mutationFn: (channel: string) => api.testNotification(channel),
  });

  const handleSave = () => {
    updateMutation.mutate({
      ...formData,
      email_recipients: formData.email_recipients.split(',').map(e => e.trim()).filter(Boolean),
    });
  };

  if (isLoading) {
    return <div className="text-center py-8">Loading...</div>;
  }

  return (
    <div className="max-w-4xl space-y-8">
      <h1 className="text-2xl font-bold text-gray-900">Settings</h1>

      <div className="bg-white rounded-lg shadow p-6">
        <div className="flex items-center gap-2 mb-6">
          <Bell className="w-5 h-5 text-primary-500" />
          <h2 className="text-lg font-semibold">Notification Settings</h2>
        </div>

        <div className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Minimum Severity to Notify
            </label>
            <select
              value={formData.min_severity}
              onChange={(e) => setFormData({ ...formData, min_severity: e.target.value })}
              className="block w-full max-w-xs rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
            >
              <option value="low">Low and above</option>
              <option value="medium">Medium and above</option>
              <option value="high">High and above</option>
              <option value="critical">Critical only</option>
            </select>
          </div>

          <div className="border-t pt-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <MessageSquare className="w-5 h-5 text-purple-600" />
                <h3 className="font-medium">Slack Notifications</h3>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.slack_enabled}
                  onChange={(e) => setFormData({ ...formData, slack_enabled: e.target.checked })}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
              </label>
            </div>

            {formData.slack_enabled && (
              <div className="grid gap-4 pl-7">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Webhook URL</label>
                  <input
                    type="password"
                    value={formData.slack_webhook_url}
                    onChange={(e) => setFormData({ ...formData, slack_webhook_url: e.target.value })}
                    placeholder="https://hooks.slack.com/services/..."
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Channel</label>
                  <input
                    type="text"
                    value={formData.slack_channel}
                    onChange={(e) => setFormData({ ...formData, slack_channel: e.target.value })}
                    placeholder="#security-alerts"
                    className="mt-1 block w-full max-w-xs rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                  />
                </div>
                <button
                  onClick={() => testMutation.mutate('slack')}
                  className="w-fit flex items-center gap-2 px-3 py-1.5 text-sm bg-purple-50 text-purple-700 rounded hover:bg-purple-100"
                >
                  <Send className="w-4 h-4" /> Send Test Message
                </button>
              </div>
            )}
          </div>

          <div className="border-t pt-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Mail className="w-5 h-5 text-primary-500" />
                <h3 className="font-medium">Email Notifications</h3>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.email_enabled}
                  onChange={(e) => setFormData({ ...formData, email_enabled: e.target.checked })}
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-500"></div>
              </label>
            </div>

            {formData.email_enabled && (
              <div className="grid gap-4 pl-7">
                <div>
                  <label className="block text-sm font-medium text-gray-700">Recipients</label>
                  <input
                    type="text"
                    value={formData.email_recipients}
                    onChange={(e) => setFormData({ ...formData, email_recipients: e.target.value })}
                    placeholder="security@company.com, alerts@company.com"
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500"
                  />
                  <p className="mt-1 text-xs text-gray-500">Comma-separated email addresses</p>
                </div>
                <button
                  onClick={() => testMutation.mutate('email')}
                  className="w-fit flex items-center gap-2 px-3 py-1.5 text-sm bg-primary-50 text-primary-600 rounded hover:bg-primary-100"
                >
                  <Send className="w-4 h-4" /> Send Test Email
                </button>
              </div>
            )}
          </div>
        </div>

        <div className="mt-8 pt-6 border-t flex justify-end">
          <button
            onClick={handleSave}
            disabled={updateMutation.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 disabled:opacity-50"
          >
            <Save className="w-4 h-4" />
            {updateMutation.isPending ? 'Saving...' : 'Save Settings'}
          </button>
        </div>

        {updateMutation.isSuccess && (
          <div className="mt-4 p-3 bg-green-50 text-green-700 rounded-lg text-sm">
            Settings saved successfully!
          </div>
        )}

        {testMutation.isSuccess && (
          <div className="mt-4 p-3 bg-primary-50 text-primary-600 rounded-lg text-sm">
            Test notification sent!
          </div>
        )}
      </div>
    </div>
  );
}
