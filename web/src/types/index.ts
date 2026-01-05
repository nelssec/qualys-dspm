export type Provider = 'AWS' | 'AZURE' | 'GCP';
export type Sensitivity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'UNKNOWN';
export type Category = 'PII' | 'PHI' | 'PCI' | 'SECRETS' | 'CUSTOM';
export type FindingSeverity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO';
export type FindingStatus = 'open' | 'in_progress' | 'resolved' | 'suppressed' | 'false_positive';
export type ScanStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
export type ScanType = 'FULL' | 'INCREMENTAL' | 'ASSET_DISCOVERY' | 'CLASSIFICATION' | 'ACCESS_ANALYSIS';

export interface CloudAccount {
  id: string;
  provider: Provider;
  external_id: string;
  display_name: string;
  status: string;
  last_scan_at?: string;
  created_at: string;
}

export interface DataAsset {
  id: string;
  account_id: string;
  resource_type: string;
  resource_arn: string;
  region: string;
  name: string;
  encryption_status: string;
  public_access: boolean;
  sensitivity_level: Sensitivity;
  data_categories: string[];
  classification_count: number;
  last_scanned_at?: string;
  tags: Record<string, string>;
}

export interface Classification {
  id: string;
  asset_id: string;
  object_path: string;
  rule_name: string;
  category: Category;
  sensitivity: Sensitivity;
  finding_count: number;
  confidence_score: number;
  discovered_at: string;
}

export interface Finding {
  id: string;
  account_id: string;
  asset_id?: string;
  finding_type: string;
  severity: FindingSeverity;
  title: string;
  description: string;
  remediation: string;
  status: FindingStatus;
  compliance_frameworks: string[];
  created_at: string;
  resolved_at?: string;
}

export interface ScanJob {
  id: string;
  account_id: string;
  scan_type: ScanType;
  status: ScanStatus;
  total_assets: number;
  scanned_assets: number;
  findings_count: number;
  classifications_count: number;
  started_at?: string;
  completed_at?: string;
}

export interface DashboardSummary {
  accounts: {
    total: number;
    active: number;
  };
  assets: {
    total: number;
    public: number;
    critical: number;
  };
  findings: {
    total: number;
    open: number;
    critical: number;
  };
  classifications: {
    total: number;
    by_category: Record<string, number>;
  };
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
  meta?: {
    total: number;
    limit: number;
    offset: number;
  };
}
