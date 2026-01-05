import axios from 'axios';
import type {
  ApiResponse,
  CloudAccount,
  DataAsset,
  Classification,
  Finding,
  ScanJob,
  DashboardSummary,
  ScanType,
  FindingStatus,
  Sensitivity
} from '../types';

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle token refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      try {
        const refreshToken = localStorage.getItem('refresh_token');
        const { data } = await axios.post('/api/v1/auth/refresh', { refresh_token: refreshToken });
        localStorage.setItem('access_token', data.data.access_token);
        localStorage.setItem('refresh_token', data.data.refresh_token);
        originalRequest.headers.Authorization = `Bearer ${data.data.access_token}`;
        return apiClient(originalRequest);
      } catch {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

// Types for new features
export interface ScheduledJob {
  id: string;
  name: string;
  description: string;
  schedule: string;
  job_type: string;
  config?: Record<string, string>;
  enabled: boolean;
  last_run?: string;
  next_run?: string;
  created_at: string;
}

export interface CustomRule {
  id: string;
  name: string;
  description: string;
  category: string;
  sensitivity: string;
  patterns: string[];
  context_patterns?: string[];
  context_required: boolean;
  enabled: boolean;
  priority: number;
  created_by?: string;
  created_at: string;
}

export interface RuleTemplate {
  name: string;
  description: string;
  patterns: string[];
  hint: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  token_type: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
}

export interface NotificationSettings {
  slack_enabled: boolean;
  slack_channel: string;
  email_enabled: boolean;
  email_recipients: string[];
  min_severity: string;
}

export interface ReportType {
  type: string;
  name: string;
  description: string;
}

// Auth API
const login = async (email: string, password: string): Promise<TokenPair> => {
  const { data } = await axios.post<ApiResponse<TokenPair>>('/api/v1/auth/login', { email, password });
  return data.data!;
};

const logout = async (): Promise<void> => {
  await apiClient.post('/auth/logout');
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
};

const getCurrentUser = async (): Promise<User> => {
  const { data } = await apiClient.get<ApiResponse<User>>('/auth/me');
  return data.data!;
};

// Accounts
const getAccounts = async (): Promise<CloudAccount[]> => {
  const { data } = await apiClient.get<ApiResponse<CloudAccount[]>>('/accounts');
  return data.data || [];
};

const getAccount = async (id: string): Promise<CloudAccount> => {
  const { data } = await apiClient.get<ApiResponse<CloudAccount>>(`/accounts/${id}`);
  return data.data!;
};

const createAccount = async (account: {
  provider: string;
  external_id: string;
  display_name: string;
  connector_config: Record<string, unknown>;
}): Promise<CloudAccount> => {
  const { data } = await apiClient.post<ApiResponse<CloudAccount>>('/accounts', account);
  return data.data!;
};

const deleteAccount = async (id: string): Promise<void> => {
  await apiClient.delete(`/accounts/${id}`);
};

const triggerScan = async (accountId: string, scanType: ScanType): Promise<ScanJob> => {
  const { data } = await apiClient.post<ApiResponse<ScanJob>>(`/accounts/${accountId}/scan`, {
    scan_type: scanType,
  });
  return data.data!;
};

// Assets
const getAssets = async (params?: {
  account_id?: string;
  resource_type?: string;
  sensitivity?: Sensitivity;
  public_only?: boolean;
  limit?: number;
  offset?: number;
}): Promise<{ assets: DataAsset[]; total: number }> => {
  const { data } = await apiClient.get<ApiResponse<DataAsset[]>>('/assets', { params });
  return {
    assets: data.data || [],
    total: data.meta?.total || 0,
  };
};

const getAsset = async (id: string): Promise<DataAsset> => {
  const { data } = await apiClient.get<ApiResponse<DataAsset>>(`/assets/${id}`);
  return data.data!;
};

const getAssetClassifications = async (id: string): Promise<Classification[]> => {
  const { data } = await apiClient.get<ApiResponse<Classification[]>>(`/assets/${id}/classifications`);
  return data.data || [];
};

// Findings
const getFindings = async (params?: {
  account_id?: string;
  asset_id?: string;
  severity?: string;
  status?: FindingStatus;
  type?: string;
  limit?: number;
  offset?: number;
}): Promise<{ findings: Finding[]; total: number }> => {
  const { data } = await apiClient.get<ApiResponse<Finding[]>>('/findings', { params });
  return {
    findings: data.data || [],
    total: data.meta?.total || 0,
  };
};

const getFinding = async (id: string): Promise<Finding> => {
  const { data } = await apiClient.get<ApiResponse<Finding>>(`/findings/${id}`);
  return data.data!;
};

const updateFindingStatus = async (
  id: string,
  status: FindingStatus,
  reason?: string
): Promise<Finding> => {
  const { data } = await apiClient.patch<ApiResponse<Finding>>(`/findings/${id}/status`, {
    status,
    reason,
  });
  return data.data!;
};

// Scans
const getScans = async (): Promise<ScanJob[]> => {
  const { data } = await apiClient.get<ApiResponse<ScanJob[]>>('/scans');
  return data.data || [];
};

const getScan = async (id: string): Promise<ScanJob> => {
  const { data } = await apiClient.get<ApiResponse<ScanJob>>(`/scans/${id}`);
  return data.data!;
};

// Dashboard
const getDashboardSummary = async (): Promise<DashboardSummary> => {
  const { data } = await apiClient.get<ApiResponse<DashboardSummary>>('/dashboard/summary');
  return data.data!;
};

const getClassificationStats = async (accountId?: string): Promise<Record<string, number>> => {
  const params = accountId ? { account_id: accountId } : {};
  const { data } = await apiClient.get<ApiResponse<Record<string, number>>>('/dashboard/classification-stats', { params });
  return data.data || {};
};

const getFindingStats = async (accountId?: string): Promise<Record<string, Record<string, number>>> => {
  const params = accountId ? { account_id: accountId } : {};
  const { data } = await apiClient.get<ApiResponse<Record<string, Record<string, number>>>>('/dashboard/finding-stats', { params });
  return data.data || {};
};

// Scheduled Jobs API
const listScheduledJobs = async (): Promise<ScheduledJob[]> => {
  const { data } = await apiClient.get<ApiResponse<ScheduledJob[]>>('/jobs');
  return data.data || [];
};

const createScheduledJob = async (job: Partial<ScheduledJob>): Promise<ScheduledJob> => {
  const { data } = await apiClient.post<ApiResponse<ScheduledJob>>('/jobs', job);
  return data.data!;
};

const updateScheduledJob = async (id: string, job: Partial<ScheduledJob>): Promise<ScheduledJob> => {
  const { data } = await apiClient.put<ApiResponse<ScheduledJob>>(`/jobs/${id}`, job);
  return data.data!;
};

const deleteScheduledJob = async (id: string): Promise<void> => {
  await apiClient.delete(`/jobs/${id}`);
};

const runScheduledJobNow = async (id: string): Promise<void> => {
  await apiClient.post(`/jobs/${id}/run`);
};

// Custom Rules API
const listRules = async (): Promise<CustomRule[]> => {
  const { data } = await apiClient.get<ApiResponse<CustomRule[]>>('/rules');
  return data.data || [];
};

const createRule = async (rule: Partial<CustomRule>): Promise<CustomRule> => {
  const { data } = await apiClient.post<ApiResponse<CustomRule>>('/rules', rule);
  return data.data!;
};

const updateRule = async (id: string, rule: Partial<CustomRule>): Promise<CustomRule> => {
  const { data } = await apiClient.put<ApiResponse<CustomRule>>(`/rules/${id}`, rule);
  return data.data!;
};

const deleteRule = async (id: string): Promise<void> => {
  await apiClient.delete(`/rules/${id}`);
};

const testRule = async (rule: Partial<CustomRule>, content: string): Promise<{ matched: boolean; match?: any }> => {
  const { data } = await apiClient.post<ApiResponse<{ matched: boolean; match?: any }>>('/rules/test', { rule, content });
  return data.data!;
};

const getRuleTemplates = async (): Promise<RuleTemplate[]> => {
  const { data } = await apiClient.get<ApiResponse<RuleTemplate[]>>('/rules/templates');
  return data.data || [];
};

// Reports API
const getReportTypes = async (): Promise<ReportType[]> => {
  const { data } = await apiClient.get<ApiResponse<ReportType[]>>('/reports/types');
  return data.data || [];
};

const generateReport = async (request: {
  type: string;
  format: string;
  title?: string;
  account_ids?: string[];
  date_from?: string;
  date_to?: string;
  severities?: string[];
  categories?: string[];
  statuses?: string[];
}): Promise<Blob> => {
  const response = await apiClient.post('/reports/generate', request, {
    responseType: 'blob',
  });
  return response.data;
};

// Notification Settings API
const getNotificationSettings = async (): Promise<NotificationSettings> => {
  const { data } = await apiClient.get<ApiResponse<NotificationSettings>>('/notifications/settings');
  return data.data!;
};

const updateNotificationSettings = async (settings: Partial<NotificationSettings>): Promise<void> => {
  await apiClient.put('/notifications/settings', settings);
};

const testNotification = async (channel: string): Promise<void> => {
  await apiClient.post(`/notifications/test?channel=${channel}`);
};

// Export all functions
export const api = {
  // Auth
  login,
  logout,
  getCurrentUser,

  // Accounts
  getAccounts,
  getAccount,
  createAccount,
  deleteAccount,
  triggerScan,

  // Assets
  getAssets,
  getAsset,
  getAssetClassifications,

  // Findings
  getFindings,
  getFinding,
  updateFindingStatus,

  // Scans
  getScans,
  getScan,

  // Dashboard
  getDashboardSummary,
  getClassificationStats,
  getFindingStats,

  // Scheduled Jobs
  listScheduledJobs,
  createScheduledJob,
  updateScheduledJob,
  deleteScheduledJob,
  runScheduledJobNow,

  // Custom Rules
  listRules,
  createRule,
  updateRule,
  deleteRule,
  testRule,
  getRuleTemplates,

  // Reports
  getReportTypes,
  generateReport,

  // Notifications
  getNotificationSettings,
  updateNotificationSettings,
  testNotification,
};

export default apiClient;
