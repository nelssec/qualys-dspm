import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { FileText, Download, FileSpreadsheet, Calendar, Filter } from 'lucide-react';
import { api } from '../api/client';

export function Reports() {
  const [reportType, setReportType] = useState('findings');
  const [format, setFormat] = useState('csv');
  const [title, setTitle] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>([]);
  const [selectedStatuses, setSelectedStatuses] = useState<string[]>([]);

  const { data: reportTypes } = useQuery({
    queryKey: ['report-types'],
    queryFn: api.getReportTypes,
  });

  const generateMutation = useMutation({
    mutationFn: api.generateReport,
    onSuccess: (blob: Blob, variables) => {
      // Download the file
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

  const severities = ['critical', 'high', 'medium', 'low'];
  const statuses = ['open', 'in_progress', 'resolved', 'false_positive'];

  return (
    <div className="max-w-4xl space-y-6">
      <h1 className="text-2xl font-bold text-gray-900">Generate Reports</h1>

      <div className="bg-white rounded-lg shadow p-6">
        {/* Report Type Selection */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-3">Report Type</label>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {reportTypes?.map((type: any) => (
              <button
                key={type.type}
                onClick={() => setReportType(type.type)}
                className={`p-4 rounded-lg border-2 text-left transition-all ${
                  reportType === type.type
                    ? 'border-blue-500 bg-blue-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <FileText className={`w-5 h-5 mb-2 ${
                  reportType === type.type ? 'text-blue-600' : 'text-gray-400'
                }`} />
                <div className="font-medium text-gray-900">{type.name}</div>
                <div className="text-xs text-gray-500 mt-1">{type.description}</div>
              </button>
            ))}
          </div>
        </div>

        {/* Format Selection */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-3">Format</label>
          <div className="flex gap-3">
            <button
              onClick={() => setFormat('csv')}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg border-2 ${
                format === 'csv'
                  ? 'border-blue-500 bg-blue-50 text-blue-700'
                  : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              <FileSpreadsheet className="w-4 h-4" />
              CSV
            </button>
            <button
              onClick={() => setFormat('pdf')}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg border-2 ${
                format === 'pdf'
                  ? 'border-blue-500 bg-blue-50 text-blue-700'
                  : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              <FileText className="w-4 h-4" />
              PDF
            </button>
          </div>
        </div>

        {/* Report Title */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-1">Report Title (optional)</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={`${reportType.charAt(0).toUpperCase() + reportType.slice(1)} Report`}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
          />
        </div>

        {/* Date Range */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-3">
            <Calendar className="w-4 h-4 inline mr-1" />
            Date Range (optional)
          </label>
          <div className="flex gap-4">
            <div>
              <label className="block text-xs text-gray-500 mb-1">From</label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-500 mb-1">To</label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
              />
            </div>
          </div>
        </div>

        {/* Filters for Findings Report */}
        {reportType === 'findings' && (
          <div className="mb-6 p-4 bg-gray-50 rounded-lg">
            <div className="flex items-center gap-2 mb-3">
              <Filter className="w-4 h-4 text-gray-500" />
              <span className="text-sm font-medium text-gray-700">Filters</span>
            </div>
            <div className="grid md:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs text-gray-500 mb-2">Severities</label>
                <div className="flex flex-wrap gap-2">
                  {severities.map((sev) => (
                    <label key={sev} className="flex items-center">
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
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                      />
                      <span className="ml-2 text-sm capitalize">{sev}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-2">Statuses</label>
                <div className="flex flex-wrap gap-2">
                  {statuses.map((status) => (
                    <label key={status} className="flex items-center">
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
                        className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                      />
                      <span className="ml-2 text-sm capitalize">{status.replace('_', ' ')}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Generate Button */}
        <div className="flex justify-end">
          <button
            onClick={handleGenerate}
            disabled={generateMutation.isPending}
            className="flex items-center gap-2 px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            <Download className="w-4 h-4" />
            {generateMutation.isPending ? 'Generating...' : 'Generate Report'}
          </button>
        </div>

        {generateMutation.isError && (
          <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm">
            Failed to generate report. Please try again.
          </div>
        )}
      </div>

      {/* Quick Export */}
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="font-medium text-gray-900 mb-4">Quick Export</h3>
        <p className="text-sm text-gray-500 mb-4">
          Download raw data exports without filters
        </p>
        <div className="flex gap-3">
          <a
            href="/api/v1/reports/stream?type=findings"
            className="flex items-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded hover:bg-gray-200"
          >
            <FileSpreadsheet className="w-4 h-4" />
            All Findings (CSV)
          </a>
          <a
            href="/api/v1/reports/stream?type=assets"
            className="flex items-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded hover:bg-gray-200"
          >
            <FileSpreadsheet className="w-4 h-4" />
            All Assets (CSV)
          </a>
        </div>
      </div>
    </div>
  );
}
