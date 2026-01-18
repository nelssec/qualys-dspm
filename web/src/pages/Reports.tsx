import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { FileText, Download, FileSpreadsheet, Calendar, Filter } from 'lucide-react';
import { api } from '../api/client';
import clsx from 'clsx';

export function Reports() {
  const [reportType, setReportType] = useState('findings');
  const [format, setFormat] = useState('csv');
  const [title, setTitle] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [selectedStatuses, setSelectedStatuses] = useState<string[]>([]);

  const { data: reportTypes, isLoading: typesLoading } = useQuery({
    queryKey: ['report-types'],
    queryFn: api.getReportTypes,
  });

  const generateMutation = useMutation({
    mutationFn: api.generateReport,
    onSuccess: (blob: Blob, variables) => {
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${variables.type}_report.${variables.format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      a.remove();
    },
  });

  const handleGenerate = () => {
    generateMutation.mutate({
      type: reportType,
      format,
      title: title || `${reportType} Report`,
      date_from: dateFrom || undefined,
      date_to: dateTo || undefined,
      severities: selectedSeverities.length > 0 ? selectedSeverities : undefined,
      statuses: selectedStatuses.length > 0 ? selectedStatuses : undefined,
    });
  };

  const handleQuickExport = async (type: string) => {
    try {
      const blob = await api.generateReport({ type, format: 'csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${type}_export.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      a.remove();
    } catch (error) {
      console.error('Export failed:', error);
    }
  };

  const severities = ['critical', 'high', 'medium', 'low'];
  const statuses = ['open', 'in_progress', 'resolved', 'false_positive'];

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-lg font-medium text-qualys-text-primary">Generate Reports</h1>
      </div>

      <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-5 mb-4">
        <div className="mb-6">
          <label className="block text-xs font-medium text-qualys-text-secondary uppercase tracking-wider mb-3">
            Report Type
          </label>
          {typesLoading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary-500" />
            </div>
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
              {reportTypes?.map((type: { type: string; name: string; description: string }) => (
                <button
                  key={type.type}
                  onClick={() => setReportType(type.type)}
                  className={clsx(
                    'p-3 rounded border text-left transition-all',
                    reportType === type.type
                      ? 'border-primary-500 bg-primary-50'
                      : 'border-qualys-border hover:border-primary-300'
                  )}
                >
                  <FileText className={clsx(
                    'w-4 h-4 mb-2',
                    reportType === type.type ? 'text-primary-600' : 'text-qualys-text-muted'
                  )} />
                  <div className="text-sm font-medium text-qualys-text-primary">{type.name}</div>
                  <div className="text-[11px] text-qualys-text-muted mt-1">{type.description}</div>
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mb-6">
          <label className="block text-xs font-medium text-qualys-text-secondary uppercase tracking-wider mb-3">
            Format
          </label>
          <div className="flex gap-3">
            <button
              onClick={() => setFormat('csv')}
              className={clsx(
                'flex items-center gap-2 px-4 py-2 rounded border text-sm',
                format === 'csv'
                  ? 'border-primary-500 bg-primary-50 text-primary-700'
                  : 'border-qualys-border hover:border-primary-300 text-qualys-text-primary'
              )}
            >
              <FileSpreadsheet className="w-4 h-4" />
              CSV
            </button>
            <button
              onClick={() => setFormat('pdf')}
              className={clsx(
                'flex items-center gap-2 px-4 py-2 rounded border text-sm',
                format === 'pdf'
                  ? 'border-primary-500 bg-primary-50 text-primary-700'
                  : 'border-qualys-border hover:border-primary-300 text-qualys-text-primary'
              )}
            >
              <FileText className="w-4 h-4" />
              PDF
            </button>
          </div>
        </div>

        <div className="mb-6">
          <label className="block text-xs font-medium text-qualys-text-secondary uppercase tracking-wider mb-2">
            Report Title (optional)
          </label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={`${reportType.charAt(0).toUpperCase() + reportType.slice(1)} Report`}
            className="block w-full px-3 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>

        <div className="mb-6">
          <label className="block text-xs font-medium text-qualys-text-secondary uppercase tracking-wider mb-2">
            <Calendar className="w-3.5 h-3.5 inline mr-1" />
            Date Range (optional)
          </label>
          <div className="flex gap-4">
            <div>
              <label className="block text-[11px] text-qualys-text-muted mb-1">From</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="px-3 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500"
              />
            </div>
            <div>
              <label className="block text-[11px] text-qualys-text-muted mb-1">To</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="px-3 py-2 text-sm border border-qualys-border rounded focus:ring-1 focus:ring-primary-500"
              />
            </div>
          </div>
        </div>

        {reportType === 'findings' && (
          <div className="mb-6 p-4 bg-qualys-bg rounded border border-qualys-border">
            <div className="flex items-center gap-2 mb-3">
              <Filter className="w-4 h-4 text-qualys-text-muted" />
              <span className="text-xs font-medium text-qualys-text-secondary uppercase tracking-wider">Filters</span>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <div>
                <label className="block text-[11px] text-qualys-text-muted mb-2">Severities</label>
                <div className="flex flex-wrap gap-3">
                  {severities.map((sev) => (
                    <label key={sev} className="flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedSeverities.includes(sev)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedSeverities([...selectedSeverities, sev]);
                          } else {
                            setSelectedSeverities(selectedSeverities.filter(s => s !== sev));
                          }
                        }}
                        className="h-3.5 w-3.5 text-primary-500 focus:ring-primary-500 border-qualys-border rounded"
                      />
                      <span className="ml-2 text-sm text-qualys-text-primary capitalize">{sev}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label className="block text-[11px] text-qualys-text-muted mb-2">Statuses</label>
                <div className="flex flex-wrap gap-3">
                  {statuses.map((status) => (
                    <label key={status} className="flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedStatuses.includes(status)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedStatuses([...selectedStatuses, status]);
                          } else {
                            setSelectedStatuses(selectedStatuses.filter(s => s !== status));
                          }
                        }}
                        className="h-3.5 w-3.5 text-primary-500 focus:ring-primary-500 border-qualys-border rounded"
                      />
                      <span className="ml-2 text-sm text-qualys-text-primary capitalize">{status.replace('_', ' ')}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="flex justify-end">
          <button
            onClick={handleGenerate}
            disabled={generateMutation.isPending}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-primary-500 text-white rounded hover:bg-primary-600 disabled:opacity-50 transition-colors"
          >
            <Download className="w-4 h-4" />
            {generateMutation.isPending ? 'Generating...' : 'Generate Report'}
          </button>
        </div>

        {generateMutation.isError && (
          <div className="mt-4 p-3 bg-severity-critical/10 text-severity-critical border border-severity-critical/20 rounded text-sm">
            Failed to generate report. Please try again.
          </div>
        )}
      </div>

      <div className="bg-white border border-qualys-border rounded shadow-qualys-sm p-5">
        <h3 className="text-sm font-medium text-qualys-text-primary mb-2">Quick Export</h3>
        <p className="text-[11px] text-qualys-text-muted mb-4">
          Download raw data exports without filters
        </p>
        <div className="flex gap-3">
          <button
            onClick={() => handleQuickExport('findings')}
            className="flex items-center gap-2 px-4 py-2 text-sm bg-qualys-bg text-qualys-text-primary border border-qualys-border rounded hover:bg-qualys-border/50 transition-colors"
          >
            <FileSpreadsheet className="w-4 h-4" />
            All Findings (CSV)
          </button>
          <button
            onClick={() => handleQuickExport('assets')}
            className="flex items-center gap-2 px-4 py-2 text-sm bg-qualys-bg text-qualys-text-primary border border-qualys-border rounded hover:bg-qualys-border/50 transition-colors"
          >
            <FileSpreadsheet className="w-4 h-4" />
            All Assets (CSV)
          </button>
        </div>
      </div>
    </div>
  );
}
