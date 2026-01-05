import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Shield, Plus, Trash2, Edit, Play, ChevronDown, ChevronUp } from 'lucide-react';
import { api, CustomRule } from '../api/client';

export function Rules() {
  const queryClient = useQueryClient();
  const [showModal, setShowModal] = useState(false);
  const [editingRule, setEditingRule] = useState<CustomRule | null>(null);
  const [expandedRule, setExpandedRule] = useState<string | null>(null);
  const [testContent, setTestContent] = useState('');
  const [testResult, setTestResult] = useState<any>(null);

  const { data: rules, isLoading } = useQuery({
    queryKey: ['rules'],
    queryFn: api.listRules,
  });

  const { data: templates } = useQuery({
    queryKey: ['rule-templates'],
    queryFn: api.getRuleTemplates,
  });

  const createMutation = useMutation({
    mutationFn: api.createRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rules'] });
      setShowModal(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: api.deleteRule,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['rules'] }),
  });

  const testMutation = useMutation({
    mutationFn: ({ rule, content }: { rule: Partial<CustomRule>; content: string }) =>
      api.testRule(rule, content),
    onSuccess: (data) => setTestResult(data),
  });

  const categories = ['pii', 'phi', 'pci', 'secrets', 'custom'];
  const sensitivities = ['low', 'medium', 'high', 'critical'];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Classification Rules</h1>
        <button
          onClick={() => { setEditingRule(null); setShowModal(true); }}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Plus className="w-4 h-4" />
          Create Rule
        </button>
      </div>

      {/* Templates Section */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-medium text-blue-900 mb-2">Quick Start Templates</h3>
        <div className="flex flex-wrap gap-2">
          {templates?.map((template: any) => (
            <button
              key={template.name}
              onClick={() => {
                setEditingRule({
                  name: template.name,
                  description: template.description,
                  patterns: template.patterns,
                  category: 'pii',
                  sensitivity: 'medium',
                  enabled: true,
                } as CustomRule);
                setShowModal(true);
              }}
              className="px-3 py-1 bg-white border border-blue-300 rounded text-sm text-blue-700 hover:bg-blue-100"
            >
              {template.name}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="text-center py-8">Loading...</div>
      ) : (
        <div className="space-y-4">
          {rules?.map((rule) => (
            <div key={rule.id} className="bg-white rounded-lg shadow overflow-hidden">
              <div
                className="px-6 py-4 flex items-center justify-between cursor-pointer hover:bg-gray-50"
                onClick={() => setExpandedRule(expandedRule === rule.id ? null : rule.id)}
              >
                <div className="flex items-center gap-4">
                  <Shield className={`w-5 h-5 ${rule.enabled ? 'text-green-600' : 'text-gray-400'}`} />
                  <div>
                    <h3 className="font-medium text-gray-900">{rule.name}</h3>
                    <p className="text-sm text-gray-500">{rule.description}</p>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${
                    {
                      critical: 'bg-red-100 text-red-800',
                      high: 'bg-orange-100 text-orange-800',
                      medium: 'bg-yellow-100 text-yellow-800',
                      low: 'bg-green-100 text-green-800',
                    }[rule.sensitivity]
                  }`}>
                    {rule.sensitivity}
                  </span>
                  <span className="px-2 py-1 bg-gray-100 rounded text-xs text-gray-700">
                    {rule.category}
                  </span>
                  {expandedRule === rule.id ? (
                    <ChevronUp className="w-5 h-5 text-gray-400" />
                  ) : (
                    <ChevronDown className="w-5 h-5 text-gray-400" />
                  )}
                </div>
              </div>

              {expandedRule === rule.id && (
                <div className="px-6 py-4 border-t border-gray-200 bg-gray-50">
                  <div className="grid grid-cols-2 gap-6">
                    <div>
                      <h4 className="text-sm font-medium text-gray-700 mb-2">Patterns</h4>
                      <div className="space-y-1">
                        {rule.patterns?.map((pattern, i) => (
                          <code key={i} className="block bg-white px-2 py-1 rounded border text-sm font-mono">
                            {pattern}
                          </code>
                        ))}
                      </div>
                      {rule.context_patterns && rule.context_patterns.length > 0 && (
                        <>
                          <h4 className="text-sm font-medium text-gray-700 mt-4 mb-2">
                            Context Patterns {rule.context_required && '(Required)'}
                          </h4>
                          <div className="space-y-1">
                            {rule.context_patterns.map((pattern, i) => (
                              <code key={i} className="block bg-white px-2 py-1 rounded border text-sm font-mono">
                                {pattern}
                              </code>
                            ))}
                          </div>
                        </>
                      )}
                    </div>
                    <div>
                      <h4 className="text-sm font-medium text-gray-700 mb-2">Test Rule</h4>
                      <textarea
                        value={testContent}
                        onChange={(e) => setTestContent(e.target.value)}
                        placeholder="Enter sample content to test..."
                        className="w-full h-24 px-3 py-2 border rounded text-sm"
                      />
                      <button
                        onClick={() => testMutation.mutate({ rule, content: testContent })}
                        className="mt-2 px-3 py-1 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 flex items-center gap-1"
                      >
                        <Play className="w-4 h-4" /> Test
                      </button>
                      {testResult && (
                        <div className={`mt-2 p-2 rounded text-sm ${
                          testResult.matched ? 'bg-green-50 text-green-800' : 'bg-gray-50 text-gray-600'
                        }`}>
                          {testResult.matched ? (
                            <>
                              Match found! Confidence: {(testResult.match.confidence * 100).toFixed(0)}%
                              <br />
                              Matches: {testResult.match.matches?.join(', ')}
                            </>
                          ) : 'No match'}
                        </div>
                      )}
                    </div>
                  </div>
                  <div className="flex justify-end gap-2 mt-4 pt-4 border-t">
                    <button
                      onClick={() => { setEditingRule(rule); setShowModal(true); }}
                      className="px-3 py-1 text-gray-600 hover:bg-gray-100 rounded flex items-center gap-1"
                    >
                      <Edit className="w-4 h-4" /> Edit
                    </button>
                    <button
                      onClick={() => deleteMutation.mutate(rule.id)}
                      className="px-3 py-1 text-red-600 hover:bg-red-50 rounded flex items-center gap-1"
                    >
                      <Trash2 className="w-4 h-4" /> Delete
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      {showModal && (
        <RuleModal
          rule={editingRule}
          categories={categories}
          sensitivities={sensitivities}
          onClose={() => setShowModal(false)}
          onSave={(data) => createMutation.mutate(data)}
        />
      )}
    </div>
  );
}

interface RuleModalProps {
  rule: CustomRule | null;
  categories: string[];
  sensitivities: string[];
  onClose: () => void;
  onSave: (data: Partial<CustomRule>) => void;
}

function RuleModal({ rule, categories, sensitivities, onClose, onSave }: RuleModalProps) {
  const [formData, setFormData] = useState({
    name: rule?.name || '',
    description: rule?.description || '',
    category: rule?.category || 'pii',
    sensitivity: rule?.sensitivity || 'medium',
    patterns: rule?.patterns?.join('\n') || '',
    context_patterns: rule?.context_patterns?.join('\n') || '',
    context_required: rule?.context_required || false,
    priority: rule?.priority || 50,
    enabled: rule?.enabled ?? true,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({
      ...formData,
      patterns: formData.patterns.split('\n').filter(p => p.trim()),
      context_patterns: formData.context_patterns.split('\n').filter(p => p.trim()),
    });
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-bold mb-4">{rule ? 'Edit Rule' : 'Create Classification Rule'}</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Priority</label>
              <input
                type="number"
                value={formData.priority}
                onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                min="0"
                max="100"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
              rows={2}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Category</label>
              <select
                value={formData.category}
                onChange={(e) => setFormData({ ...formData, category: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
              >
                {categories.map((cat) => (
                  <option key={cat} value={cat}>{cat.toUpperCase()}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Sensitivity</label>
              <select
                value={formData.sensitivity}
                onChange={(e) => setFormData({ ...formData, sensitivity: e.target.value })}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
              >
                {sensitivities.map((sens) => (
                  <option key={sens} value={sens}>{sens.charAt(0).toUpperCase() + sens.slice(1)}</option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Patterns (one regex per line)
            </label>
            <textarea
              value={formData.patterns}
              onChange={(e) => setFormData({ ...formData, patterns: e.target.value })}
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 font-mono text-sm"
              rows={4}
              placeholder="\\b\\d{3}-\\d{2}-\\d{4}\\b"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Context Patterns (optional, one regex per line)
            </label>
            <textarea
              value={formData.context_patterns}
              onChange={(e) => setFormData({ ...formData, context_patterns: e.target.value })}
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 font-mono text-sm"
              rows={3}
              placeholder="(?i)ssn|social\\s*security"
            />
          </div>
          <div className="flex items-center gap-6">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={formData.context_required}
                onChange={(e) => setFormData({ ...formData, context_required: e.target.checked })}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <span className="ml-2 text-sm text-gray-900">Context Required</span>
            </label>
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
              />
              <span className="ml-2 text-sm text-gray-900">Enabled</span>
            </label>
          </div>
          <div className="flex justify-end gap-2 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
            >
              {rule?.id ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
