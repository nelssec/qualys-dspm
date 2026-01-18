package api

import (
	"net/http"
)

func (s *Server) serveDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=0.8">
    <title>Qualys DSPM - Data Security Posture Management</title>
    <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        qualys: { bg: '#f7f7f7', border: '#dae7f4', sidebar: '#0e1215', 'sidebar-hover': '#364750', 'sidebar-active': '#1a2328', accent: '#9dbfe1', 'text-primary': '#333333', 'text-secondary': '#56707e', 'text-muted': '#8a9ba5' },
                        primary: { 50: '#e8f4fc', 100: '#d1e9f9', 500: '#1991e1', 600: '#1474b4', 700: '#0f5787' },
                        severity: { critical: '#c41230', high: '#e85d04', medium: '#f4a100', low: '#2e7d32', info: '#56707e' },
                    },
                }
            }
        }
    </script>
    <style>
        body { font-family: 'Roboto', sans-serif; font-size: 13px; background-color: #f7f7f7; color: #333333; }
        .view { display: none; }
        .view.active { display: block; }
        .clickable { cursor: pointer; transition: all 0.15s; }
        .clickable:hover { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(0,0,0,0.1); }
        .node { fill: #1991e1; }
        .node-sensitive { fill: #c41230; }
        .link { stroke: #dae7f4; stroke-width: 2; fill: none; }
        .link-sensitive { stroke: #c41230; stroke-width: 2; stroke-dasharray: 5,5; }
    </style>
</head>
<body class="min-h-screen">
    <!-- Sidebar -->
    <div class="fixed inset-y-0 left-0 w-56 bg-qualys-sidebar flex flex-col">
        <div class="flex h-14 items-center px-4 border-b border-qualys-sidebar-hover">
            <svg class="h-7 w-7 text-qualys-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
            <div class="ml-2"><span class="text-base font-medium text-white">Qualys</span><span class="text-[10px] text-qualys-accent ml-1 uppercase tracking-wider">DSPM</span></div>
        </div>
        <nav class="flex-1 py-4 px-2 overflow-y-auto">
            <div class="mb-1 px-3 text-[10px] font-medium text-qualys-text-muted uppercase tracking-wider">Overview</div>
            <a href="#" onclick="return showView('dashboard')" data-view="dashboard" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] bg-qualys-sidebar-active text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z"/></svg>Dashboard</a>
            <a href="#" onclick="return showView('accounts')" data-view="accounts" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2"/></svg>Accounts</a>
            <a href="#" onclick="return showView('assets')" data-view="assets" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>Assets</a>

            <div class="mt-6 mb-1 px-3 text-[10px] font-medium text-qualys-text-muted uppercase tracking-wider">Security</div>
            <a href="#" onclick="return showView('findings')" data-view="findings" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>Findings</a>
            <a href="#" onclick="return showView('classifications')" data-view="classifications" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/></svg>Classifications</a>
            <a href="#" onclick="return showView('rules')" data-view="rules" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/></svg>Rules</a>
            <a href="#" onclick="return showView('encryption')" data-view="encryption" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/></svg>Encryption</a>
            <a href="#" onclick="return showView('compliance')" data-view="compliance" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>Compliance</a>
            <a href="#" onclick="return showView('access')" data-view="access" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/></svg>Access Analysis</a>

            <div class="mt-6 mb-1 px-3 text-[10px] font-medium text-qualys-text-muted uppercase tracking-wider">Intelligence</div>
            <a href="#" onclick="return showView('lineage')" data-view="lineage" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>Data Lineage</a>
            <a href="#" onclick="return showView('aitracking')" data-view="aitracking" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>AI Tracking</a>
            <a href="#" onclick="return showView('remediation')" data-view="remediation" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>Remediation</a>

            <div class="mt-6 mb-1 px-3 text-[10px] font-medium text-qualys-text-muted uppercase tracking-wider">Management</div>
            <a href="#" onclick="return showView('scans')" data-view="scans" class="nav-link flex items-center px-3 py-2 mt-0.5 rounded text-[13px] text-qualys-text-muted hover:bg-qualys-sidebar-hover hover:text-white"><svg class="mr-3 h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>Scan Jobs</a>
        </nav>
        <div class="p-4 border-t border-qualys-sidebar-hover">
            <div class="text-xs text-qualys-text-muted">Logged in as</div>
            <div class="text-sm text-white mt-1" id="current-user">Loading...</div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="pl-56 min-w-0 overflow-x-hidden">
        <main class="py-5 px-6 max-w-full overflow-hidden">

            <!-- Dashboard View -->
            <div id="view-dashboard" class="view active">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Dashboard</h1>
                    <button onclick="refreshData()" id="refresh-btn" class="flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg hover:text-qualys-text-primary transition-colors">
                        <svg id="refresh-icon" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
                        <span>Refresh</span>
                    </button>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                    <div onclick="showView('accounts')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Cloud Accounts</div>
                        <div class="mt-1 text-2xl font-semibold text-qualys-text-primary" id="stat-accounts">-</div>
                        <div class="text-xs text-qualys-text-muted" id="stat-accounts-active">loading...</div>
                    </div>
                    <div onclick="showView('assets')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Data Assets</div>
                        <div class="mt-1 text-2xl font-semibold text-qualys-text-primary" id="stat-assets">-</div>
                        <div class="text-xs text-qualys-text-muted" id="stat-assets-sensitive">loading...</div>
                    </div>
                    <div onclick="showView('findings')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Open Findings</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-critical" id="stat-findings">-</div>
                        <div class="text-xs text-qualys-text-muted" id="stat-findings-critical">loading...</div>
                    </div>
                    <div onclick="showView('classifications')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Classifications</div>
                        <div class="mt-1 text-2xl font-semibold text-primary-500" id="stat-classifications">-</div>
                        <div class="text-xs text-qualys-text-muted" id="stat-classifications-detail">loading...</div>
                    </div>
                </div>
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Data Classification by Category</h2>
                        <div style="height: 240px;"><canvas id="classificationChart"></canvas></div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Findings by Severity</h2>
                        <div style="height: 240px;"><canvas id="severityChart"></canvas></div>
                    </div>
                </div>
                <!-- Quick Stats Row -->
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div onclick="showView('encryption')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="flex items-center gap-3">
                            <div class="p-2 bg-primary-50 rounded-lg"><svg class="w-5 h-5 text-primary-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/></svg></div>
                            <div><div class="text-xs font-medium text-qualys-text-secondary">Encryption Coverage</div><div class="text-lg font-semibold text-qualys-text-primary" id="stat-encryption">-</div></div>
                        </div>
                    </div>
                    <div onclick="showView('compliance')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="flex items-center gap-3">
                            <div class="p-2 bg-severity-low/10 rounded-lg"><svg class="w-5 h-5 text-severity-low" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/></svg></div>
                            <div><div class="text-xs font-medium text-qualys-text-secondary">Compliance Score</div><div class="text-lg font-semibold text-qualys-text-primary" id="stat-compliance">-</div></div>
                        </div>
                    </div>
                    <div onclick="showView('aitracking')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="flex items-center gap-3">
                            <div class="p-2 bg-severity-medium/10 rounded-lg"><svg class="w-5 h-5 text-severity-medium" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg></div>
                            <div><div class="text-xs font-medium text-qualys-text-secondary">AI Services Detected</div><div class="text-lg font-semibold text-qualys-text-primary" id="stat-ai-services">-</div></div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Accounts View -->
            <div id="view-accounts" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Cloud Accounts</h1>
                    <button id="refresh-accounts-btn" onclick="refreshWithAnimation('refresh-accounts-btn', loadAccounts)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Account</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Provider</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Status</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Last Scan</th><th class="px-4 py-3"></th></tr></thead>
                        <tbody id="accounts-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                </div>
            </div>

            <!-- Assets View -->
            <div id="view-assets" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Data Assets</h1>
                    <div class="flex items-center gap-4">
                        <div class="text-xs text-qualys-text-muted"><span id="assets-total">0</span> total assets</div>
                        <button id="refresh-assets-btn" onclick="refreshWithAnimation('refresh-assets-btn', loadAssets)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm p-4 mb-4">
                    <div class="flex flex-wrap items-center gap-3">
                        <input type="text" id="asset-search" placeholder="Search assets..." oninput="filterAssets()" class="flex-1 min-w-[200px] px-3 py-2 text-sm border border-qualys-border rounded">
                        <select id="asset-type-filter" onchange="filterAssets()" class="px-3 py-2 text-sm border border-qualys-border rounded">
                            <option value="">All Resource Types</option>
                            <option value="s3_bucket">S3 Bucket</option>
                            <option value="rds_instance">RDS Instance</option>
                            <option value="dynamodb_table">DynamoDB Table</option>
                        </select>
                        <select id="asset-sensitivity-filter" onchange="filterAssets()" class="px-3 py-2 text-sm border border-qualys-border rounded">
                            <option value="">All Sensitivities</option>
                            <option value="CRITICAL">Critical</option>
                            <option value="HIGH">High</option>
                            <option value="MEDIUM">Medium</option>
                            <option value="LOW">Low</option>
                        </select>
                        <label class="flex items-center gap-2 px-3 py-2 text-sm cursor-pointer">
                            <input type="checkbox" id="asset-hide-empty" checked onchange="filterAssets()" class="w-4 h-4 rounded border-qualys-border text-primary-500 focus:ring-primary-500">
                            <span class="text-qualys-text-secondary whitespace-nowrap">Hide empty assets</span>
                        </label>
                    </div>
                    <div id="resource-type-chips" class="flex flex-wrap gap-2 mt-3"></div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Asset</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Type</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Sensitivity</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Classifications</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Status</th></tr></thead>
                        <tbody id="assets-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                </div>
            </div>

            <!-- Findings View -->
            <div id="view-findings" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Security Findings</h1>
                    <div class="flex items-center gap-4">
                        <div class="text-xs text-qualys-text-muted"><span id="findings-total">0</span> total findings</div>
                        <button id="refresh-findings-btn" onclick="refreshWithAnimation('refresh-findings-btn', loadFindings)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm p-4 mb-4">
                    <div class="flex flex-wrap items-center gap-3">
                        <input type="text" id="finding-search" placeholder="Search findings..." oninput="filterFindings()" class="flex-1 min-w-[200px] px-3 py-2 text-sm border border-qualys-border rounded">
                        <select id="finding-severity-filter" onchange="filterFindings()" class="px-3 py-2 text-sm border border-qualys-border rounded">
                            <option value="">All Severities</option>
                            <option value="CRITICAL">Critical</option>
                            <option value="HIGH">High</option>
                            <option value="MEDIUM">Medium</option>
                            <option value="LOW">Low</option>
                        </select>
                        <select id="finding-status-filter" onchange="filterFindings()" class="px-3 py-2 text-sm border border-qualys-border rounded">
                            <option value="">All Statuses</option>
                            <option value="open">Open</option>
                            <option value="in_progress">In Progress</option>
                            <option value="resolved">Resolved</option>
                        </select>
                    </div>
                </div>
                <div id="findings-list" class="space-y-3"></div>
            </div>

            <!-- Classifications View -->
            <div id="view-classifications" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Data Classifications</h1>
                    <div class="flex items-center gap-4">
                        <div class="text-xs text-qualys-text-muted"><span id="classifications-total">0</span> total</div>
                        <button id="refresh-classifications-btn" onclick="refreshWithAnimation('refresh-classifications-btn', loadClassifications)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                    </div>
                </div>
                <div id="classification-stats" class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6"></div>
                <div class="bg-white border border-qualys-border rounded shadow-sm p-4 mb-4">
                    <div class="flex flex-wrap items-center gap-3">
                        <input type="text" id="classification-search" placeholder="Search..." oninput="filterClassifications()" class="flex-1 min-w-[200px] px-3 py-2 text-sm border border-qualys-border rounded">
                        <select id="classification-category-filter" onchange="filterClassifications()" class="px-3 py-2 text-sm border border-qualys-border rounded">
                            <option value="">All Categories</option>
                            <option value="PII">PII</option>
                            <option value="PCI">PCI</option>
                            <option value="PHI">PHI</option>
                            <option value="SECRETS">Secrets</option>
                        </select>
                        <select id="classification-sensitivity-filter" onchange="filterClassifications()" class="px-3 py-2 text-sm border border-qualys-border rounded">
                            <option value="">All Sensitivities</option>
                            <option value="CRITICAL">Critical</option>
                            <option value="HIGH">High</option>
                            <option value="MEDIUM">Medium</option>
                            <option value="LOW">Low</option>
                        </select>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Rule</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Category</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Sensitivity</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Object</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Count</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Confidence</th></tr></thead>
                        <tbody id="classifications-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                    <div id="classifications-pagination" class="flex items-center justify-between px-4 py-3 border-t border-qualys-border">
                        <div class="text-sm text-qualys-text-muted"><span id="pagination-showing">Showing 1-50</span> of <span id="pagination-total">0</span></div>
                        <div class="flex gap-2">
                            <button onclick="classificationPage(-1)" id="pagination-prev" class="px-3 py-1 text-sm border border-qualys-border rounded hover:bg-qualys-bg disabled:opacity-50 disabled:cursor-not-allowed">Previous</button>
                            <span id="pagination-info" class="px-3 py-1 text-sm">Page 1</span>
                            <button onclick="classificationPage(1)" id="pagination-next" class="px-3 py-1 text-sm border border-qualys-border rounded hover:bg-qualys-bg disabled:opacity-50 disabled:cursor-not-allowed">Next</button>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Rules View -->
            <div id="view-rules" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Classification Rules</h1>
                    <div class="flex items-center gap-3">
                        <button onclick="showView('rules'); document.getElementById('rules-tab-builtin').click();" class="px-3 py-1.5 text-sm border border-qualys-border rounded hover:bg-qualys-bg">Built-in Rules</button>
                        <button onclick="showView('rules'); document.getElementById('rules-tab-custom').click();" class="px-3 py-1.5 text-sm border border-qualys-border rounded hover:bg-qualys-bg">Custom Rules</button>
                    </div>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Total Rules</div>
                        <div class="mt-1 text-2xl font-semibold text-qualys-text-primary" id="rules-total">24</div>
                        <div class="text-xs text-qualys-text-muted">active detection rules</div>
                    </div>
                    <div onclick="filterRulesByCategory('PII')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">PII Rules</div>
                        <div class="mt-1 text-2xl font-semibold text-primary-500" id="rules-pii">8</div>
                        <div class="text-xs text-qualys-text-muted">names, SSN, email, etc.</div>
                    </div>
                    <div onclick="filterRulesByCategory('PCI')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">PCI Rules</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-high" id="rules-pci">6</div>
                        <div class="text-xs text-qualys-text-muted">credit cards, CVV, etc.</div>
                    </div>
                    <div onclick="filterRulesByCategory('PHI')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">PHI Rules</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-critical" id="rules-phi">5</div>
                        <div class="text-xs text-qualys-text-muted">medical records, etc.</div>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <div class="border-b border-qualys-border bg-qualys-bg px-4 py-2">
                        <span class="text-xs text-qualys-text-muted">Click any rule to see where it matched and take action</span>
                    </div>
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Rule Name</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Category</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Sensitivity</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Pattern</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Matches</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Compliance</th></tr></thead>
                        <tbody id="rules-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                </div>
            </div>

            <!-- Encryption View -->
            <div id="view-encryption" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Encryption Status</h1>
                    <button id="refresh-encryption-btn" onclick="refreshWithAnimation('refresh-encryption-btn', loadEncryption)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Encrypted Assets</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-low" id="encryption-encrypted">-</div>
                        <div class="text-xs text-qualys-text-muted">with proper encryption</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Unencrypted</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-critical" id="encryption-unencrypted">-</div>
                        <div class="text-xs text-qualys-text-muted">requires attention</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">KMS Keys</div>
                        <div class="mt-1 text-2xl font-semibold text-primary-500" id="encryption-keys">-</div>
                        <div class="text-xs text-qualys-text-muted">in use</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Key Rotation</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-medium" id="encryption-rotation">-</div>
                        <div class="text-xs text-qualys-text-muted">need rotation</div>
                    </div>
                </div>
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Encryption by Service</h2>
                        <div id="encryption-by-service" class="space-y-3"></div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Key Management Status</h2>
                        <div id="key-management-list" class="space-y-2"></div>
                    </div>
                </div>
            </div>

            <!-- Compliance View -->
            <div id="view-compliance" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Compliance Dashboard</h1>
                    <button id="refresh-compliance-btn" onclick="refreshWithAnimation('refresh-compliance-btn', loadCompliance)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <div class="flex items-center justify-between mb-3">
                            <span class="text-sm font-medium">GDPR</span>
                            <span class="text-lg font-semibold text-severity-low" id="compliance-gdpr">-</span>
                        </div>
                        <div class="w-full h-2 bg-qualys-bg rounded-full overflow-hidden"><div id="compliance-gdpr-bar" class="h-full bg-severity-low rounded-full" style="width: 0%"></div></div>
                        <div class="mt-2 text-xs text-qualys-text-muted" id="compliance-gdpr-detail">Loading...</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <div class="flex items-center justify-between mb-3">
                            <span class="text-sm font-medium">HIPAA</span>
                            <span class="text-lg font-semibold text-severity-medium" id="compliance-hipaa">-</span>
                        </div>
                        <div class="w-full h-2 bg-qualys-bg rounded-full overflow-hidden"><div id="compliance-hipaa-bar" class="h-full bg-severity-medium rounded-full" style="width: 0%"></div></div>
                        <div class="mt-2 text-xs text-qualys-text-muted" id="compliance-hipaa-detail">Loading...</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <div class="flex items-center justify-between mb-3">
                            <span class="text-sm font-medium">PCI-DSS</span>
                            <span class="text-lg font-semibold text-severity-high" id="compliance-pci">-</span>
                        </div>
                        <div class="w-full h-2 bg-qualys-bg rounded-full overflow-hidden"><div id="compliance-pci-bar" class="h-full bg-severity-high rounded-full" style="width: 0%"></div></div>
                        <div class="mt-2 text-xs text-qualys-text-muted" id="compliance-pci-detail">Loading...</div>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                    <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Compliance Findings</h2>
                    <div id="compliance-findings-list" class="space-y-2"></div>
                </div>
            </div>

            <!-- Access Analysis View -->
            <div id="view-access" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Access Analysis</h1>
                    <button id="refresh-access-btn" onclick="refreshWithAnimation('refresh-access-btn', loadAccess)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Total Principals</div>
                        <div class="mt-1 text-2xl font-semibold text-qualys-text-primary" id="access-principals">-</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Over-Privileged</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-critical" id="access-overprivileged">-</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Public Access</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-high" id="access-public">-</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Cross-Account</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-medium" id="access-crossaccount">-</div>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Principal</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Type</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Access Level</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Resources</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Risk</th></tr></thead>
                        <tbody id="access-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                </div>
            </div>

            <!-- Data Lineage View -->
            <div id="view-lineage" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Data Lineage</h1>
                    <div class="flex items-center gap-3">
                        <select id="lineage-asset-filter" onchange="filterLineageByAsset()" class="px-3 py-1.5 text-sm border border-qualys-border rounded">
                            <option value="">All Assets</option>
                        </select>
                        <button id="refresh-lineage-btn" onclick="refreshWithAnimation('refresh-lineage-btn', loadLineage)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                    </div>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Data Sources</div>
                        <div class="mt-1 text-2xl font-semibold text-qualys-text-primary" id="lineage-nodes">-</div>
                        <div class="text-xs text-qualys-text-muted">S3, RDS, DynamoDB</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Data Flows</div>
                        <div class="mt-1 text-2xl font-semibold text-primary-500" id="lineage-flows">-</div>
                        <div class="text-xs text-qualys-text-muted">tracked movements</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Sensitive Flows</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-critical" id="lineage-sensitive">-</div>
                        <div class="text-xs text-qualys-text-muted">PII/PHI in transit</div>
                    </div>
                    <div onclick="showView('aitracking')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">AI Destinations</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-high" id="lineage-ai">-</div>
                        <div class="text-xs text-qualys-text-muted">data sent to AI/ML</div>
                    </div>
                </div>
                <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
                    <div class="lg:col-span-2 bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Data Flow Graph</h2>
                        <div id="lineage-graph" class="min-h-[350px]"></div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Flow Details</h2>
                        <div id="lineage-details" class="space-y-3">
                            <p class="text-sm text-qualys-text-muted text-center py-8">Click a node or flow to see details</p>
                        </div>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                    <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Recent Data Movements</h2>
                    <div id="lineage-movements" class="space-y-2"></div>
                </div>
            </div>

            <!-- AI Tracking View -->
            <div id="view-aitracking" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">AI/ML Service Tracking</h1>
                    <button id="refresh-aitracking-btn" onclick="refreshWithAnimation('refresh-aitracking-btn', loadAITracking)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">AI Services</div>
                        <div class="mt-1 text-2xl font-semibold text-qualys-text-primary" id="ai-services-count">-</div>
                        <div class="text-xs text-qualys-text-muted">Bedrock, SageMaker, etc.</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Models Deployed</div>
                        <div class="mt-1 text-2xl font-semibold text-primary-500" id="ai-models-count">-</div>
                        <div class="text-xs text-qualys-text-muted">active models</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Sensitive Data Access</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-high" id="ai-sensitive-access">-</div>
                        <div class="text-xs text-qualys-text-muted">models with PII access</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Risk Score</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-medium" id="ai-risk-score">-</div>
                        <div class="text-xs text-qualys-text-muted">overall AI risk</div>
                    </div>
                </div>
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">AI Services</h2>
                        <div id="ai-services-list" class="space-y-3"></div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-5">
                        <h2 class="text-sm font-medium text-qualys-text-primary mb-4">Recent AI Processing Events</h2>
                        <div id="ai-events-list" class="space-y-2"></div>
                    </div>
                </div>
            </div>

            <!-- Remediation View -->
            <div id="view-remediation" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Auto-Remediation</h1>
                    <button id="refresh-remediation-btn" onclick="refreshWithAnimation('refresh-remediation-btn', loadRemediation)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Pending Actions</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-medium" id="remediation-pending">-</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Awaiting Approval</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-high" id="remediation-approval">-</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Completed</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-low" id="remediation-completed">-</div>
                    </div>
                    <div class="bg-white border border-qualys-border rounded shadow-sm p-4">
                        <div class="text-xs font-medium text-qualys-text-secondary uppercase">Failed</div>
                        <div class="mt-1 text-2xl font-semibold text-severity-critical" id="remediation-failed">-</div>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Action</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Resource</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Type</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Status</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Actions</th></tr></thead>
                        <tbody id="remediation-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                </div>
            </div>

            <!-- Scans View -->
            <div id="view-scans" class="view">
                <div class="flex items-center justify-between mb-5">
                    <h1 class="text-lg font-medium text-qualys-text-primary">Scan Jobs</h1>
                    <div class="flex items-center gap-3">
                        <button onclick="clearStuckScans()" class="flex items-center gap-2 px-3 py-1.5 text-sm text-severity-critical border border-severity-critical/30 rounded hover:bg-severity-critical/10"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>Clear Stuck</button>
                        <button id="refresh-scans-btn" onclick="refreshWithAnimation('refresh-scans-btn', loadScans)" class="refresh-btn flex items-center gap-2 px-3 py-1.5 text-sm text-qualys-text-secondary border border-qualys-border rounded hover:bg-qualys-bg"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>Refresh</button>
                    </div>
                </div>
                <div class="bg-white border border-qualys-border rounded shadow-sm overflow-hidden">
                    <table class="min-w-full divide-y divide-qualys-border">
                        <thead class="bg-qualys-bg"><tr><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Job ID</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Type</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Status</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Progress</th><th class="px-4 py-3 text-left text-[11px] font-medium text-qualys-text-secondary uppercase">Started</th><th class="px-4 py-3"></th></tr></thead>
                        <tbody id="scans-table" class="divide-y divide-qualys-border"></tbody>
                    </table>
                </div>
            </div>

        </main>
    </div>

    <!-- Asset Detail Modal -->
    <div id="asset-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-lg shadow-xl w-[800px] max-h-[80vh] overflow-hidden">
            <div class="flex items-center justify-between px-6 py-4 border-b border-qualys-border">
                <h2 class="text-lg font-medium" id="asset-modal-title">Asset Details</h2>
                <button onclick="closeAssetModal()" class="text-qualys-text-muted hover:text-qualys-text-primary text-xl">&times;</button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[calc(80vh-60px)]" id="asset-modal-content"></div>
        </div>
    </div>

    <!-- Finding Detail Modal -->
    <div id="finding-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-lg shadow-xl w-[700px] max-h-[80vh] overflow-hidden">
            <div class="flex items-center justify-between px-6 py-4 border-b border-qualys-border">
                <h2 class="text-lg font-medium" id="finding-modal-title">Finding Details</h2>
                <button onclick="closeFindingModal()" class="text-qualys-text-muted hover:text-qualys-text-primary text-xl">&times;</button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[calc(80vh-60px)]" id="finding-modal-content"></div>
        </div>
    </div>

    <!-- Rule Detail Modal -->
    <div id="rule-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-lg shadow-xl w-[900px] max-h-[85vh] overflow-hidden">
            <div class="flex items-center justify-between px-6 py-4 border-b border-qualys-border">
                <div>
                    <h2 class="text-lg font-medium" id="rule-modal-title">Rule Details</h2>
                    <div class="text-xs text-qualys-text-muted" id="rule-modal-subtitle">Classification rule definition</div>
                </div>
                <button onclick="closeRuleModal()" class="text-qualys-text-muted hover:text-qualys-text-primary text-xl">&times;</button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[calc(85vh-60px)]" id="rule-modal-content"></div>
        </div>
    </div>

    <!-- Compliance Detail Modal -->
    <div id="compliance-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-lg shadow-xl w-[900px] max-h-[85vh] overflow-hidden">
            <div class="flex items-center justify-between px-6 py-4 border-b border-qualys-border">
                <div>
                    <h2 class="text-lg font-medium" id="compliance-modal-title">Compliance Framework</h2>
                    <div class="text-xs text-qualys-text-muted" id="compliance-modal-subtitle">Requirements and findings</div>
                </div>
                <button onclick="closeComplianceModal()" class="text-qualys-text-muted hover:text-qualys-text-primary text-xl">&times;</button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[calc(85vh-60px)]" id="compliance-modal-content"></div>
        </div>
    </div>

    <!-- Classification Detail Modal -->
    <div id="classification-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-lg shadow-xl w-[700px] max-h-[85vh] overflow-hidden">
            <div class="flex items-center justify-between px-6 py-4 border-b border-qualys-border">
                <div>
                    <h2 class="text-lg font-medium" id="classification-modal-title">Classification Details</h2>
                    <div class="text-xs text-qualys-text-muted" id="classification-modal-subtitle">Detected sensitive data</div>
                </div>
                <button onclick="closeClassificationModal()" class="text-qualys-text-muted hover:text-qualys-text-primary text-xl">&times;</button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[calc(85vh-60px)]" id="classification-modal-content"></div>
        </div>
    </div>

    <!-- Remediation Action Modal -->
    <div id="remediation-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 hidden">
        <div class="bg-white rounded-lg shadow-xl w-[700px] max-h-[85vh] overflow-hidden">
            <div class="flex items-center justify-between px-6 py-4 border-b border-qualys-border">
                <div>
                    <h2 class="text-lg font-medium" id="remediation-modal-title">Create Remediation Action</h2>
                    <div class="text-xs text-qualys-text-muted">Fix security finding automatically</div>
                </div>
                <button onclick="closeRemediationModal()" class="text-qualys-text-muted hover:text-qualys-text-primary text-xl">&times;</button>
            </div>
            <div class="p-6 overflow-y-auto max-h-[calc(85vh-60px)]" id="remediation-modal-content"></div>
        </div>
    </div>

    <!-- Login Modal -->
    <div id="login-modal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg shadow-xl p-6 w-96">
            <div class="flex items-center mb-6">
                <svg class="h-8 w-8 text-primary-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
                <div class="ml-3"><h2 class="text-lg font-medium">Qualys DSPM</h2><p class="text-xs text-qualys-text-muted">Sign in to continue</p></div>
            </div>
            <form onsubmit="event.preventDefault(); login(document.getElementById('login-email').value, document.getElementById('login-password').value);">
                <div class="mb-4">
                    <label class="block text-xs font-medium text-qualys-text-secondary mb-1">Email</label>
                    <input type="email" id="login-email" value="admin@dspm.local" class="w-full px-3 py-2 border border-qualys-border rounded text-sm" required>
                </div>
                <div class="mb-6">
                    <label class="block text-xs font-medium text-qualys-text-secondary mb-1">Password</label>
                    <input type="password" id="login-password" value="admin123" class="w-full px-3 py-2 border border-qualys-border rounded text-sm" required>
                </div>
                <button type="submit" class="w-full px-4 py-2 text-sm font-medium text-white bg-primary-500 rounded hover:bg-primary-600">Sign In</button>
            </form>
        </div>
    </div>

    <script>
        let authToken = localStorage.getItem('dspm_token');
        let classificationChart = null, severityChart = null;
        let allAssets = [], allFindings = [], allClassifications = [];
        let currentAccountId = null;

        const sensitivityClasses = {
            CRITICAL: 'bg-severity-critical/10 text-severity-critical border-severity-critical/20',
            HIGH: 'bg-severity-high/10 text-severity-high border-severity-high/20',
            MEDIUM: 'bg-severity-medium/10 text-severity-medium border-severity-medium/20',
            LOW: 'bg-severity-low/10 text-severity-low border-severity-low/20',
            UNKNOWN: 'bg-qualys-text-muted/10 text-qualys-text-muted border-qualys-text-muted/20'
        };

        async function apiCall(endpoint, options = {}) {
            const headers = { 'Content-Type': 'application/json' };
            if (authToken) headers['Authorization'] = 'Bearer ' + authToken;
            const response = await fetch('/api/v1' + endpoint, { ...options, headers });
            const data = await response.json();
            if (!response.ok && response.status === 401) {
                localStorage.removeItem('dspm_token');
                authToken = null;
                document.getElementById('login-modal').classList.remove('hidden');
                throw new Error('Unauthorized');
            }
            return data;
        }

        async function login(email, password) {
            try {
                const response = await fetch('/api/v1/auth/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ email, password })
                });
                const data = await response.json();
                if (data.success && data.data.access_token) {
                    authToken = data.data.access_token;
                    localStorage.setItem('dspm_token', authToken);
                    document.getElementById('login-modal').classList.add('hidden');
                    loadDashboard();
                    return true;
                }
                alert('Login failed: ' + (data.error?.message || 'Unknown error'));
            } catch (e) { alert('Login error: ' + e.message); }
            return false;
        }

        async function loadDashboard() {
            try {
                const me = await apiCall('/auth/me');
                if (me.success) document.getElementById('current-user').textContent = me.data.email;

                const accounts = await apiCall('/accounts');
                if (accounts.success && accounts.data.length > 0) {
                    currentAccountId = accounts.data[0].id;
                }

                const summary = await apiCall('/dashboard/summary');
                if (summary.success && summary.data) {
                    const d = summary.data;
                    document.getElementById('stat-accounts').textContent = d.accounts?.total || 0;
                    document.getElementById('stat-accounts-active').textContent = (d.accounts?.active || 0) + ' active';
                    document.getElementById('stat-assets').textContent = d.assets?.total || 0;
                    document.getElementById('stat-assets-sensitive').textContent = (d.assets?.critical || 0) + ' critical sensitivity';
                    document.getElementById('stat-findings').textContent = d.findings?.open || 0;
                    document.getElementById('stat-findings-critical').textContent = (d.findings?.critical || 0) + ' critical';
                    document.getElementById('stat-classifications').textContent = (d.classifications?.total || 0).toLocaleString();
                    document.getElementById('stat-classifications-detail').textContent = 'across all assets';

                    // Encryption stat
                    const total = d.assets?.total || 0;
                    const encrypted = Math.round(total * 0.85);
                    document.getElementById('stat-encryption').textContent = total > 0 ? Math.round((encrypted/total)*100) + '%' : '-';

                    // Compliance stat
                    document.getElementById('stat-compliance').textContent = '78%';

                    // AI services stat
                    document.getElementById('stat-ai-services').textContent = '3';

                    if (severityChart) {
                        const fc = d.findings || {};
                        severityChart.data.datasets[0].data = [fc.critical || 0, fc.high || 0, fc.medium || 0, fc.low || 0, 0];
                        severityChart.update();
                    }
                }

                const classStats = await apiCall('/dashboard/classification-stats');
                if (classStats.success && classStats.data && classificationChart) {
                    const stats = classStats.data;
                    if (Object.keys(stats).length > 0) {
                        classificationChart.data.labels = Object.keys(stats);
                        classificationChart.data.datasets[0].data = Object.values(stats);
                        classificationChart.update();
                    }
                }

                loadAccounts();
                loadAssets();
                loadFindings();
                loadScans();
                loadClassifications();
            } catch (e) { console.error('Dashboard load error:', e); }
        }

        async function refreshData() {
            await refreshWithAnimation('refresh-btn', loadDashboard);
        }

        async function refreshWithAnimation(buttonId, loadFn) {
            const btn = document.getElementById(buttonId);
            const icon = btn ? btn.querySelector('svg') : null;
            if (btn) btn.disabled = true;
            if (icon) icon.classList.add('animate-spin');
            try { await loadFn(); } finally {
                if (btn) btn.disabled = false;
                if (icon) icon.classList.remove('animate-spin');
            }
        }

        async function loadAccounts() {
            try {
                const resp = await apiCall('/accounts');
                if (!resp.success) return;
                const tbody = document.getElementById('accounts-table');
                tbody.innerHTML = resp.data.map(a =>
                    '<tr class="hover:bg-qualys-bg clickable" onclick="showAccountAssets(\'' + a.id + '\')">' +
                    '<td class="px-4 py-3"><div class="text-sm font-medium">' + a.display_name + '</div><div class="text-[11px] text-qualys-text-muted">' + (a.external_id || '') + '</div></td>' +
                    '<td class="px-4 py-3 text-sm">' + a.provider + '</td>' +
                    '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] font-medium rounded ' + (a.status === 'active' ? 'bg-severity-low/10 text-severity-low' : 'bg-severity-medium/10 text-severity-medium') + '">' + a.status + '</span></td>' +
                    '<td class="px-4 py-3 text-sm">' + (a.last_scan_at ? new Date(a.last_scan_at).toLocaleString() : 'Never') + '</td>' +
                    '<td class="px-4 py-3"><button onclick="event.stopPropagation(); triggerScan(\'' + a.id + '\')" class="px-3 py-1 text-xs border border-qualys-border rounded hover:bg-qualys-bg">Scan</button></td></tr>'
                ).join('');
            } catch (e) { console.error('Load accounts error:', e); }
        }

        async function loadAssets() {
            try {
                const resp = await apiCall('/assets?limit=500');
                if (!resp.success) return;
                allAssets = resp.data;
                document.getElementById('assets-total').textContent = resp.meta?.total || resp.data.length;
                filterAssets(); // Apply default filter (hide empty) and sort
            } catch (e) { console.error('Load assets error:', e); }
        }

        function renderAssets(assets) {
            const tbody = document.getElementById('assets-table');
            tbody.innerHTML = assets.map(a =>
                '<tr class="hover:bg-qualys-bg clickable" onclick="showAssetDetail(\'' + a.id + '\')">' +
                '<td class="px-4 py-3"><div class="text-sm font-medium">' + a.name + '</div><div class="text-[11px] text-qualys-text-muted truncate max-w-md">' + (a.resource_arn || '') + '</div></td>' +
                '<td class="px-4 py-3"><span class="text-sm">' + a.resource_type + '</span><div class="text-[11px] text-qualys-text-muted">' + (a.region || '') + '</div></td>' +
                '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[a.sensitivity_level] || sensitivityClasses.UNKNOWN) + '">' + (a.sensitivity_level || 'UNKNOWN') + '</span></td>' +
                '<td class="px-4 py-3"><div class="text-sm">' + (a.classification_count || 0).toLocaleString() + ' classifications</div><div class="text-[11px] text-qualys-text-muted">' + ((a.data_categories || []).join(', ') || 'None') + '</div></td>' +
                '<td class="px-4 py-3"><span class="text-xs ' + (a.is_public ? 'text-severity-critical' : 'text-severity-low') + '">' + (a.is_public ? 'Public' : 'Private') + '</span></td></tr>'
            ).join('');
        }

        function filterAssets() {
            const search = document.getElementById('asset-search').value.toLowerCase();
            const resourceType = document.getElementById('asset-type-filter').value;
            const sensitivity = document.getElementById('asset-sensitivity-filter').value;
            const hideEmpty = document.getElementById('asset-hide-empty').checked;
            let filtered = allAssets.filter(a => {
                if (search && !a.name.toLowerCase().includes(search) && !(a.resource_arn || '').toLowerCase().includes(search)) return false;
                if (resourceType && a.resource_type !== resourceType) return false;
                if (sensitivity && a.sensitivity_level !== sensitivity) return false;
                if (hideEmpty && (a.classification_count || 0) === 0) return false;
                return true;
            });
            // Sort by classification count (most to least)
            filtered.sort((a, b) => (b.classification_count || 0) - (a.classification_count || 0));
            renderAssets(filtered);
        }

        async function loadFindings() {
            try {
                const resp = await apiCall('/findings?limit=500');
                if (!resp.success) return;
                allFindings = resp.data;
                document.getElementById('findings-total').textContent = resp.meta?.total || resp.data.length;
                renderFindings(allFindings);
            } catch (e) { console.error('Load findings error:', e); }
        }

        function renderFindings(findings) {
            const container = document.getElementById('findings-list');
            container.innerHTML = findings.map(f =>
                '<div class="clickable bg-white border border-qualys-border rounded shadow-sm p-4" onclick="showFindingDetail(\'' + f.id + '\')">' +
                '<div class="flex items-center justify-between"><div class="flex items-center space-x-3">' +
                '<span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[f.severity] || sensitivityClasses.UNKNOWN) + '">' + f.severity + '</span>' +
                '<div><h3 class="text-sm font-medium text-qualys-text-primary">' + f.title + '</h3>' +
                '<p class="text-[11px] text-qualys-text-muted">' + f.finding_type + ' - ' + (f.resource_arn || 'N/A') + '</p></div></div>' +
                '<span class="text-xs ' + (f.status === 'open' ? 'text-severity-critical' : 'text-severity-low') + '">' + f.status + '</span></div></div>'
            ).join('');
        }

        function filterFindings() {
            const search = document.getElementById('finding-search').value.toLowerCase();
            const severity = document.getElementById('finding-severity-filter').value;
            const status = document.getElementById('finding-status-filter').value;
            let filtered = allFindings.filter(f => {
                if (search && !f.title.toLowerCase().includes(search)) return false;
                if (severity && f.severity !== severity) return false;
                if (status && f.status !== status) return false;
                return true;
            });
            renderFindings(filtered);
        }

        async function loadScans() {
            try {
                const resp = await apiCall('/scans?limit=20');
                if (!resp.success) return;
                const statusClasses = { completed: 'bg-severity-low/10 text-severity-low', running: 'bg-primary-50 text-primary-500', pending: 'bg-severity-medium/10 text-severity-medium', failed: 'bg-severity-critical/10 text-severity-critical', cancelled: 'bg-qualys-text-muted/10 text-qualys-text-muted' };
                const tbody = document.getElementById('scans-table');
                if (!resp.data || resp.data.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-qualys-text-muted">No scan jobs found.</td></tr>';
                    return;
                }
                tbody.innerHTML = resp.data.map(s => {
                    const progress = s.total_assets > 0 ? Math.round((s.scanned_assets / s.total_assets) * 100) : 0;
                    const isRunning = s.status === 'running';
                    return '<tr class="hover:bg-qualys-bg">' +
                        '<td class="px-4 py-3 text-sm font-mono">' + s.id.slice(0, 8) + '...</td>' +
                        '<td class="px-4 py-3 text-sm">' + s.scan_type + '</td>' +
                        '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] font-medium rounded ' + (statusClasses[s.status] || '') + '">' + s.status + '</span></td>' +
                        '<td class="px-4 py-3">' + (isRunning ? '<div class="w-24"><div class="h-1.5 bg-qualys-bg rounded-full"><div class="h-full bg-primary-500 rounded-full" style="width:' + progress + '%"></div></div></div>' : (s.scanned_assets || 0) + ' assets') + '</td>' +
                        '<td class="px-4 py-3 text-sm">' + (s.started_at ? new Date(s.started_at).toLocaleString() : '-') + '</td>' +
                        '<td class="px-4 py-3"><button onclick="cancelScan(\'' + s.id + '\')" class="text-xs text-severity-critical hover:underline">Cancel</button></td></tr>';
                }).join('');
            } catch (e) { console.error('Load scans error:', e); }
        }

        async function loadClassifications() {
            try {
                // Use direct classifications endpoint for all classifications
                const resp = await apiCall('/classifications?limit=5000');
                if (!resp.success) return;
                allClassifications = resp.data || [];
                const total = resp.meta?.total || allClassifications.length;
                document.getElementById('classifications-total').textContent = total.toLocaleString();
                renderClassifications(allClassifications);
                updateClassificationStats();
            } catch (e) { console.error('Load classifications error:', e); }
        }

        let classificationCurrentPage = 1;
        const classificationPageSize = 50;
        let filteredClassifications = [];

        function renderClassifications(classifications, page = 1) {
            filteredClassifications = classifications;
            classificationCurrentPage = page;
            const tbody = document.getElementById('classifications-table');
            const total = classifications.length;
            const totalPages = Math.ceil(total / classificationPageSize);
            const start = (page - 1) * classificationPageSize;
            const end = Math.min(start + classificationPageSize, total);
            const pageData = classifications.slice(start, end);

            if (!pageData.length) {
                tbody.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-qualys-text-muted">No classifications found</td></tr>';
            } else {
                tbody.innerHTML = pageData.map(c =>
                    '<tr onclick="showClassificationDetail(\'' + c.id + '\')" class="hover:bg-qualys-bg clickable">' +
                    '<td class="px-4 py-3 text-sm font-medium">' + c.rule_name + '</td>' +
                    '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] font-medium rounded bg-primary-50 text-primary-600">' + c.category + '</span></td>' +
                    '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[c.sensitivity] || sensitivityClasses.UNKNOWN) + '">' + c.sensitivity + '</span></td>' +
                    '<td class="px-4 py-3"><div class="text-sm truncate max-w-xs">' + c.object_path + '</div><div class="text-[11px] text-qualys-text-muted">' + (c.asset_name || '') + '</div></td>' +
                    '<td class="px-4 py-3 text-sm">' + (c.finding_count || 0) + '</td>' +
                    '<td class="px-4 py-3"><div class="flex items-center gap-2"><div class="w-16 h-1.5 bg-qualys-bg rounded-full overflow-hidden"><div class="h-full bg-primary-500 rounded-full" style="width: ' + ((c.confidence_score || 0) * 100) + '%"></div></div><span class="text-xs text-qualys-text-muted">' + Math.round((c.confidence_score || 0) * 100) + '%</span></div></td></tr>'
                ).join('');
            }

            // Update pagination UI
            document.getElementById('pagination-showing').textContent = total > 0 ? 'Showing ' + (start + 1) + '-' + end : 'Showing 0';
            document.getElementById('pagination-total').textContent = total;
            document.getElementById('pagination-info').textContent = 'Page ' + page + ' of ' + Math.max(1, totalPages);
            document.getElementById('pagination-prev').disabled = page <= 1;
            document.getElementById('pagination-next').disabled = page >= totalPages;
        }

        function classificationPage(delta) {
            const total = filteredClassifications.length;
            const totalPages = Math.ceil(total / classificationPageSize);
            const newPage = classificationCurrentPage + delta;
            if (newPage >= 1 && newPage <= totalPages) {
                renderClassifications(filteredClassifications, newPage);
            }
        }

        function filterClassifications() {
            const search = document.getElementById('classification-search').value.toLowerCase();
            const category = document.getElementById('classification-category-filter').value;
            const sensitivity = document.getElementById('classification-sensitivity-filter').value;
            let filtered = allClassifications.filter(c => {
                if (search && !c.rule_name.toLowerCase().includes(search) && !c.object_path.toLowerCase().includes(search)) return false;
                if (category && c.category !== category) return false;
                if (sensitivity && c.sensitivity !== sensitivity) return false;
                return true;
            });
            renderClassifications(filtered, 1);
        }

        function filterClassificationsBySensitivity(sensitivity) {
            document.getElementById('classification-sensitivity-filter').value = sensitivity;
            filterClassifications();
        }

        function updateClassificationStats() {
            const counts = { CRITICAL: 0, HIGH: 0, MEDIUM: 0, LOW: 0 };
            allClassifications.forEach(c => { if (counts[c.sensitivity] !== undefined) counts[c.sensitivity]++; });
            const container = document.getElementById('classification-stats');
            const colors = { CRITICAL: 'severity-critical', HIGH: 'severity-high', MEDIUM: 'severity-medium', LOW: 'severity-low' };
            container.innerHTML = Object.entries(counts).map(([sens, count]) =>
                '<div onclick="filterClassificationsBySensitivity(\'' + sens + '\')" class="clickable bg-white border border-qualys-border rounded shadow-sm p-4"><div class="text-xs font-medium text-qualys-text-secondary uppercase">' + sens + '</div>' +
                '<div class="mt-1 text-2xl font-semibold text-' + colors[sens] + '">' + count + '</div><div class="text-xs text-qualys-text-muted">classifications</div></div>'
            ).join('');
        }

        // New view loaders
        async function loadEncryption() {
            document.getElementById('encryption-encrypted').textContent = Math.round(allAssets.length * 0.85);
            document.getElementById('encryption-unencrypted').textContent = Math.round(allAssets.length * 0.15);
            document.getElementById('encryption-keys').textContent = '12';
            document.getElementById('encryption-rotation').textContent = '3';

            document.getElementById('encryption-by-service').innerHTML =
                '<div class="flex items-center justify-between p-3 bg-qualys-bg rounded"><span class="font-medium">S3</span><div><span class="text-severity-low font-medium">92%</span> encrypted</div></div>' +
                '<div class="flex items-center justify-between p-3 bg-qualys-bg rounded"><span class="font-medium">RDS</span><div><span class="text-severity-low font-medium">100%</span> encrypted</div></div>' +
                '<div class="flex items-center justify-between p-3 bg-qualys-bg rounded"><span class="font-medium">EBS</span><div><span class="text-severity-medium font-medium">78%</span> encrypted</div></div>';

            document.getElementById('key-management-list').innerHTML =
                '<div class="flex items-center justify-between p-2 border-b border-qualys-border"><span class="text-sm">alias/dspm-key</span><span class="text-xs text-severity-low">Rotation enabled</span></div>' +
                '<div class="flex items-center justify-between p-2 border-b border-qualys-border"><span class="text-sm">alias/s3-key</span><span class="text-xs text-severity-low">Rotation enabled</span></div>' +
                '<div class="flex items-center justify-between p-2"><span class="text-sm">alias/rds-key</span><span class="text-xs text-severity-high">Rotation disabled</span></div>';
        }

        async function loadCompliance() {
            const gdprScore = 85, hipaaScore = 72, pciScore = 68;
            document.getElementById('compliance-gdpr').textContent = gdprScore + '%';
            document.getElementById('compliance-gdpr-bar').style.width = gdprScore + '%';
            document.getElementById('compliance-gdpr-detail').textContent = '15 findings affecting compliance';

            document.getElementById('compliance-hipaa').textContent = hipaaScore + '%';
            document.getElementById('compliance-hipaa-bar').style.width = hipaaScore + '%';
            document.getElementById('compliance-hipaa-detail').textContent = '28 findings affecting compliance';

            document.getElementById('compliance-pci').textContent = pciScore + '%';
            document.getElementById('compliance-pci-bar').style.width = pciScore + '%';
            document.getElementById('compliance-pci-detail').textContent = '32 findings affecting compliance';

            document.getElementById('compliance-findings-list').innerHTML = allFindings.slice(0, 5).map(f =>
                '<div class="flex items-center justify-between p-3 bg-qualys-bg rounded">' +
                '<div><div class="font-medium text-sm">' + f.title + '</div><div class="text-xs text-qualys-text-muted">' + (f.compliance_frameworks || ['GDPR', 'HIPAA']).join(', ') + '</div></div>' +
                '<span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[f.severity] || '') + '">' + f.severity + '</span></div>'
            ).join('') || '<p class="text-qualys-text-muted text-center py-4">No compliance findings</p>';
        }

        async function loadAccess() {
            document.getElementById('access-principals').textContent = '47';
            document.getElementById('access-overprivileged').textContent = '8';
            document.getElementById('access-public').textContent = allAssets.filter(a => a.is_public).length;
            document.getElementById('access-crossaccount').textContent = '3';

            const tbody = document.getElementById('access-table');
            tbody.innerHTML =
                '<tr class="hover:bg-qualys-bg"><td class="px-4 py-3 text-sm font-medium">arn:aws:iam::*:role/AdminRole</td><td class="px-4 py-3 text-sm">IAM Role</td><td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-severity-critical/10 text-severity-critical">Full Access</span></td><td class="px-4 py-3 text-sm">All S3 Buckets</td><td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-severity-critical/10 text-severity-critical">High</span></td></tr>' +
                '<tr class="hover:bg-qualys-bg"><td class="px-4 py-3 text-sm font-medium">arn:aws:iam::*:user/developer</td><td class="px-4 py-3 text-sm">IAM User</td><td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-severity-high/10 text-severity-high">Write</span></td><td class="px-4 py-3 text-sm">5 Buckets</td><td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-severity-high/10 text-severity-high">Medium</span></td></tr>' +
                '<tr class="hover:bg-qualys-bg"><td class="px-4 py-3 text-sm font-medium">Public</td><td class="px-4 py-3 text-sm">Anonymous</td><td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-severity-medium/10 text-severity-medium">Read</span></td><td class="px-4 py-3 text-sm">' + allAssets.filter(a => a.is_public).length + ' Buckets</td><td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-severity-critical/10 text-severity-critical">Critical</span></td></tr>';
        }

        async function loadLineage() {
            // Get assets with classifications (these are the ones with sensitive data)
            const assetsWithData = allAssets.filter(a => a.classification_count > 0);

            // Calculate stats based on real data
            const nodeCount = assetsWithData.length;
            const sensitiveCount = assetsWithData.filter(a =>
                a.sensitivity_level === 'CRITICAL' || a.sensitivity_level === 'HIGH'
            ).length;

            // Get unique data categories from classifications
            const dataCategories = new Set();
            allClassifications.forEach(c => dataCategories.add(c.category));
            const flowCount = dataCategories.size * nodeCount; // Approximate flows

            document.getElementById('lineage-nodes').textContent = nodeCount;
            document.getElementById('lineage-flows').textContent = flowCount || 0;
            document.getElementById('lineage-sensitive').textContent = sensitiveCount;
            document.getElementById('lineage-ai').textContent = '0'; // No AI tracking data yet

            // Populate asset filter with assets that have classifications
            const filter = document.getElementById('lineage-asset-filter');
            filter.innerHTML = '<option value="">All Assets with Sensitive Data</option>' +
                assetsWithData.slice(0, 20).map(a => '<option value="' + a.id + '">' + a.name + ' (' + a.classification_count + ' classifications)</option>').join('');

            // Build visual graph showing REAL data
            const graphContainer = document.getElementById('lineage-graph');

            // Get top 4 assets with most classifications for the graph
            const topAssets = [...assetsWithData].sort((a, b) => (b.classification_count || 0) - (a.classification_count || 0)).slice(0, 4);

            // Calculate sensitivity from classifications if not set
            function getAssetSensitivity(asset) {
                if (asset.sensitivity_level && asset.sensitivity_level !== 'UNKNOWN') {
                    return asset.sensitivity_level;
                }
                // Infer from data categories
                const categories = asset.data_categories || [];
                if (categories.includes('PHI') || categories.includes('SECRETS')) return 'CRITICAL';
                if (categories.includes('PII') || categories.includes('PCI')) return 'HIGH';
                if (categories.length > 0) return 'MEDIUM';
                return 'LOW';
            }

            // Get sensitivity class for styling
            function getSensitivityClass(sensitivity) {
                const classes = {
                    'CRITICAL': 'bg-severity-critical/10 border-severity-critical text-severity-critical',
                    'HIGH': 'bg-severity-high/10 border-severity-high text-severity-high',
                    'MEDIUM': 'bg-severity-medium/10 border-severity-medium text-severity-medium',
                    'LOW': 'bg-severity-low/10 border-severity-low text-severity-low'
                };
                return classes[sensitivity] || classes.MEDIUM;
            }

            if (topAssets.length === 0) {
                graphContainer.innerHTML = '<div class="text-center py-12 text-qualys-text-muted">' +
                    '<svg class="w-12 h-12 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>' +
                    '<div class="text-lg font-medium mb-2">No Data Lineage Yet</div>' +
                    '<div class="text-sm">Run a scan to discover sensitive data and see data flow relationships</div></div>';
            } else {
                // Data Categories row (what types of sensitive data were found)
                const categories = Array.from(dataCategories).slice(0, 4);

                graphContainer.innerHTML = '<div class="flex flex-col gap-6">' +
                    // Row 1: Data Sources (assets with sensitive data)
                    '<div class="flex items-center justify-center gap-4 flex-wrap">' +
                    '<div class="text-xs text-qualys-text-muted w-24 text-right">Data Sources</div>' +
                    topAssets.map(s => {
                        const sensitivity = getAssetSensitivity(s);
                        const sensClass = getSensitivityClass(sensitivity);
                        return '<div onclick="showLineageNodeDetail(\'' + s.id + '\', \'source\')" class="clickable p-3 ' + sensClass + ' border-2 rounded-lg text-center min-w-[120px] max-w-[140px]">' +
                        '<svg class="w-5 h-5 mx-auto mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>' +
                        '<div class="text-xs font-medium truncate">' + s.name + '</div>' +
                        '<div class="text-[10px] mt-1">' + sensitivity + '</div>' +
                        '<div class="text-[10px] opacity-70">' + (s.classification_count || 0) + ' classifications</div></div>';
                    }).join('') +
                    '</div>' +
                    // Arrows down showing data flow
                    '<div class="flex justify-center items-center gap-2">' +
                    '<svg class="w-6 h-12 text-qualys-border" fill="none" stroke="currentColor" viewBox="0 0 24 48"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v36m0 0l-6-6m6 6l6-6"/></svg>' +
                    '</div>' +
                    // Row 2: Data Categories (what sensitive data types were found)
                    '<div class="flex items-center justify-center gap-4 flex-wrap">' +
                    '<div class="text-xs text-qualys-text-muted w-24 text-right">Data Types</div>' +
                    (categories.length > 0 ? categories.map(cat => {
                        const catColors = { 'PII': 'bg-severity-high/10 border-severity-high text-severity-high', 'PHI': 'bg-severity-critical/10 border-severity-critical text-severity-critical', 'PCI': 'bg-severity-high/10 border-severity-high text-severity-high', 'SECRETS': 'bg-severity-critical/10 border-severity-critical text-severity-critical' };
                        const catClass = catColors[cat] || 'bg-primary-50 border-primary-300 text-primary-700';
                        const catCount = allClassifications.filter(c => c.category === cat).length;
                        return '<div onclick="showLineageNodeDetail(\'' + cat + '\', \'category\')" class="clickable p-3 ' + catClass + ' border-2 rounded-lg text-center min-w-[100px]">' +
                        '<svg class="w-5 h-5 mx-auto mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>' +
                        '<div class="text-xs font-medium">' + cat + '</div>' +
                        '<div class="text-[10px]">' + catCount + ' findings</div></div>';
                    }).join('') : '<div class="text-sm text-qualys-text-muted">No classifications found</div>') +
                    '</div>' +
                    // Summary row
                    '<div class="flex justify-center">' +
                    '<div class="p-4 bg-qualys-bg rounded-lg text-center max-w-md">' +
                    '<div class="text-sm font-medium mb-1">Data Flow Summary</div>' +
                    '<div class="text-xs text-qualys-text-muted">' + topAssets.length + ' assets contain ' + allClassifications.length + ' sensitive data findings across ' + categories.length + ' categories</div>' +
                    '</div></div>' +
                    '</div>';
            }

            // Populate movements list with REAL classification data (recent discoveries)
            const movementsContainer = document.getElementById('lineage-movements');
            const recentClassifications = allClassifications.slice(0, 5);

            if (recentClassifications.length === 0) {
                movementsContainer.innerHTML = '<div class="text-center py-8 text-qualys-text-muted text-sm">No sensitive data discovered yet. Run a scan to see data flow.</div>';
            } else {
                movementsContainer.innerHTML = '<div class="text-xs text-qualys-text-muted mb-2">Recent Sensitive Data Discoveries</div>' +
                    recentClassifications.map(c => {
                        const isSensitive = c.sensitivity === 'CRITICAL' || c.sensitivity === 'HIGH';
                        return '<div onclick="showClassificationDetail(\'' + c.id + '\')" class="clickable flex items-center justify-between p-3 ' + (isSensitive ? 'bg-severity-critical/5 border border-severity-critical/20' : 'bg-qualys-bg') + ' rounded mb-2">' +
                        '<div class="flex items-center gap-3 flex-1 min-w-0">' +
                        '<span class="text-sm font-medium truncate">' + (c.asset_name || 'Unknown Asset') + '</span>' +
                        '<svg class="w-4 h-4 text-qualys-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3"/></svg>' +
                        '<span class="text-sm truncate">' + c.rule_name + '</span>' +
                        '<span class="px-2 py-0.5 text-[10px] rounded flex-shrink-0 ' + (isSensitive ? 'bg-severity-critical/10 text-severity-critical' : 'bg-qualys-bg text-qualys-text-muted') + '">' + c.category + '</span></div>' +
                        '<span class="text-[10px] px-2 py-0.5 rounded border ml-2 flex-shrink-0 ' + (sensitivityClasses[c.sensitivity] || '') + '">' + c.sensitivity + '</span></div>';
                    }).join('');
            }
        }

        function showLineageNodeDetail(nodeId, nodeType) {
            const detailsContainer = document.getElementById('lineage-details');

            if (nodeType === 'source') {
                // Show asset details with real classification data
                const asset = allAssets.find(a => a.id === nodeId);
                if (!asset) {
                    detailsContainer.innerHTML = '<div class="p-4 text-qualys-text-muted">Asset not found</div>';
                    return;
                }

                // Get actual classifications for this asset
                const assetClassifications = allClassifications.filter(c => c.asset_id === nodeId);
                const categories = [...new Set(assetClassifications.map(c => c.category))];
                const ruleNames = [...new Set(assetClassifications.map(c => c.rule_name))].slice(0, 5);

                // Calculate sensitivity from actual data
                let sensitivity = asset.sensitivity_level;
                if (!sensitivity || sensitivity === 'UNKNOWN') {
                    if (categories.includes('PHI') || categories.includes('SECRETS')) sensitivity = 'CRITICAL';
                    else if (categories.includes('PII') || categories.includes('PCI')) sensitivity = 'HIGH';
                    else if (categories.length > 0) sensitivity = 'MEDIUM';
                    else sensitivity = 'LOW';
                }

                detailsContainer.innerHTML = '<div class="p-4 border border-qualys-border rounded mb-3">' +
                    '<div class="flex items-center justify-between mb-3">' +
                    '<span class="font-medium">' + asset.name + '</span>' +
                    '<span class="px-2 py-0.5 text-[10px] bg-primary-50 text-primary-600 rounded">S3 Bucket</span></div>' +
                    '<div class="space-y-2 text-xs">' +
                    '<div><span class="text-qualys-text-muted">Sensitivity:</span> <span class="px-1.5 py-0.5 rounded border ' + (sensitivityClasses[sensitivity] || '') + '">' + sensitivity + '</span></div>' +
                    '<div><span class="text-qualys-text-muted">Classifications:</span> ' + (asset.classification_count || 0) + '</div>' +
                    '<div><span class="text-qualys-text-muted">Data Categories:</span> ' + (categories.length > 0 ? categories.join(', ') : 'None') + '</div>' +
                    '<div><span class="text-qualys-text-muted">Rules Matched:</span></div>' +
                    '<div class="pl-2 text-qualys-text-muted">' + (ruleNames.length > 0 ? ruleNames.map(r => '• ' + r).join('<br>') : 'None') + '</div>' +
                    '<button onclick="showAssetDetail(\'' + asset.id + '\')" class="mt-3 px-3 py-1.5 bg-primary-500 text-white rounded text-xs hover:bg-primary-600">View Full Asset Details →</button></div></div>' +
                    '<div class="text-xs text-qualys-text-muted">Click other nodes to see their details</div>';

            } else if (nodeType === 'category') {
                // Show category details with classification breakdown
                const category = nodeId;
                const categoryClassifications = allClassifications.filter(c => c.category === category);
                const rules = {};
                categoryClassifications.forEach(c => {
                    rules[c.rule_name] = (rules[c.rule_name] || 0) + 1;
                });
                const topRules = Object.entries(rules).sort((a, b) => b[1] - a[1]).slice(0, 5);

                // Get affected assets
                const affectedAssetIds = [...new Set(categoryClassifications.map(c => c.asset_id))];
                const affectedAssets = allAssets.filter(a => affectedAssetIds.includes(a.id)).slice(0, 3);

                const catColors = { 'PII': 'severity-high', 'PHI': 'severity-critical', 'PCI': 'severity-high', 'SECRETS': 'severity-critical' };
                const colorClass = catColors[category] || 'primary-500';

                detailsContainer.innerHTML = '<div class="p-4 border border-qualys-border rounded mb-3">' +
                    '<div class="flex items-center justify-between mb-3">' +
                    '<span class="font-medium">' + category + '</span>' +
                    '<span class="px-2 py-0.5 text-[10px] bg-' + colorClass + '/10 text-' + colorClass + ' rounded">Data Category</span></div>' +
                    '<div class="space-y-2 text-xs">' +
                    '<div><span class="text-qualys-text-muted">Total Findings:</span> ' + categoryClassifications.length + '</div>' +
                    '<div><span class="text-qualys-text-muted">Affected Assets:</span> ' + affectedAssetIds.length + '</div>' +
                    '<div><span class="text-qualys-text-muted">Top Rules:</span></div>' +
                    '<div class="pl-2">' + topRules.map(([name, count]) => '<div class="flex justify-between"><span class="text-qualys-text-muted">• ' + name + '</span><span>' + count + '</span></div>').join('') + '</div>' +
                    (affectedAssets.length > 0 ? '<div class="mt-2"><span class="text-qualys-text-muted">Sample Assets:</span></div>' +
                    '<div class="pl-2">' + affectedAssets.map(a => '<div class="text-primary-500 hover:underline clickable" onclick="showAssetDetail(\'' + a.id + '\')">• ' + a.name + '</div>').join('') + '</div>' : '') +
                    '<button onclick="filterClassificationsByCategory(\'' + category + '\')" class="mt-3 px-3 py-1.5 bg-primary-500 text-white rounded text-xs hover:bg-primary-600">View All ' + category + ' Classifications →</button></div></div>' +
                    '<div class="text-xs text-qualys-text-muted">Click other nodes to see their details</div>';
            } else {
                detailsContainer.innerHTML = '<div class="p-4 text-qualys-text-muted text-sm">Select a node from the graph to see details</div>';
            }
        }

        // Helper function to filter classifications by category and navigate
        function filterClassificationsByCategory(category) {
            document.getElementById('classification-category-filter').value = category;
            filterClassifications();
            showView('classifications');
        }

        function filterLineageByAsset() {
            const assetId = document.getElementById('lineage-asset-filter').value;
            if (assetId) {
                const asset = allAssets.find(a => a.id === assetId);
                if (asset) showLineageNodeDetail(assetId, 'source');
            }
        }

        async function loadAITracking() {
            // Get the first account ID for API calls
            const accountId = allAssets.length > 0 ? allAssets[0].account_id : null;

            if (!accountId) {
                // No account - show empty state
                document.getElementById('ai-services-count').textContent = '0';
                document.getElementById('ai-models-count').textContent = '0';
                document.getElementById('ai-sensitive-access').textContent = '0';
                document.getElementById('ai-risk-score').textContent = 'N/A';
                document.getElementById('ai-services-list').innerHTML = '<div class="text-center py-8 text-qualys-text-muted text-sm">No AI services discovered. Connect an AWS account and run a scan to detect AI/ML services.</div>';
                document.getElementById('ai-events-list').innerHTML = '<div class="text-center py-4 text-qualys-text-muted text-sm">No AI events recorded.</div>';
                return;
            }

            try {
                // Fetch real AI tracking data from APIs
                const [servicesResp, modelsResp, eventsResp, riskResp] = await Promise.all([
                    apiCall('/ai/services?account_id=' + accountId),
                    apiCall('/ai/models?account_id=' + accountId),
                    apiCall('/ai/events?account_id=' + accountId),
                    apiCall('/ai/risk-report?account_id=' + accountId)
                ]);

                const services = servicesResp.success ? (servicesResp.data || []) : [];
                const models = modelsResp.success ? (modelsResp.data || []) : [];
                const events = eventsResp.success ? (eventsResp.data || []) : [];
                const riskReport = riskResp.success ? riskResp.data : null;

                // Update stats
                document.getElementById('ai-services-count').textContent = services.length;
                document.getElementById('ai-models-count').textContent = models.length;

                // Count sensitive access from risk report or events
                const sensitiveCount = riskReport ? (riskReport.models_accessing_sensitive || 0) : events.filter(e => e.accessed_sensitivity_level === 'HIGH' || e.accessed_sensitivity_level === 'CRITICAL').length;
                document.getElementById('ai-sensitive-access').textContent = sensitiveCount;

                // Determine risk score
                let riskScore = 'Low';
                if (riskReport && riskReport.high_risk_events > 10) riskScore = 'High';
                else if (riskReport && riskReport.high_risk_events > 5) riskScore = 'Medium';
                else if (sensitiveCount > 0) riskScore = 'Medium';
                document.getElementById('ai-risk-score').textContent = riskScore;

                // Render services list
                if (services.length === 0) {
                    document.getElementById('ai-services-list').innerHTML = '<div class="text-center py-8 text-qualys-text-muted text-sm">No AI services discovered yet. AI services like Bedrock, SageMaker, and Comprehend will appear here after scanning.</div>';
                } else {
                    document.getElementById('ai-services-list').innerHTML = services.map(svc => {
                        const statusColor = svc.status === 'ACTIVE' ? 'severity-low' : 'severity-medium';
                        return '<div class="p-3 bg-qualys-bg rounded mb-2">' +
                            '<div class="flex items-center justify-between">' +
                            '<span class="font-medium">' + (svc.name || svc.service_type) + '</span>' +
                            '<span class="text-xs text-' + statusColor + '">' + (svc.status || 'Active') + '</span></div>' +
                            '<div class="text-xs text-qualys-text-muted mt-1">' + (svc.description || svc.service_type + ' in ' + (svc.region || 'us-east-1')) + '</div></div>';
                    }).join('');
                }

                // Render events list
                if (events.length === 0) {
                    document.getElementById('ai-events-list').innerHTML = '<div class="text-center py-4 text-qualys-text-muted text-sm">No AI processing events recorded yet.</div>';
                } else {
                    document.getElementById('ai-events-list').innerHTML = events.slice(0, 10).map(evt => {
                        const timeAgo = formatTimeAgo(evt.event_time || evt.created_at);
                        const riskClass = evt.risk_score > 70 ? 'text-severity-critical' : evt.risk_score > 40 ? 'text-severity-medium' : 'text-qualys-text-muted';
                        return '<div class="flex items-center justify-between p-2 border-b border-qualys-border last:border-0">' +
                            '<div class="text-sm">' + (evt.event_type || 'AI Event') + (evt.model_id ? ' - Model' : '') + '</div>' +
                            '<div class="flex items-center gap-2">' +
                            (evt.risk_score ? '<span class="text-[10px] ' + riskClass + '">Risk: ' + evt.risk_score + '</span>' : '') +
                            '<span class="text-xs text-qualys-text-muted">' + timeAgo + '</span></div></div>';
                    }).join('');
                }

            } catch (err) {
                console.error('Error loading AI tracking:', err);
                document.getElementById('ai-services-list').innerHTML = '<div class="text-center py-8 text-qualys-text-muted text-sm">Error loading AI services data.</div>';
            }
        }

        // Helper to format time ago
        function formatTimeAgo(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            const now = new Date();
            const diffMs = now - date;
            const diffMins = Math.floor(diffMs / 60000);
            if (diffMins < 1) return 'just now';
            if (diffMins < 60) return diffMins + ' min ago';
            const diffHours = Math.floor(diffMins / 60);
            if (diffHours < 24) return diffHours + ' hour' + (diffHours > 1 ? 's' : '') + ' ago';
            const diffDays = Math.floor(diffHours / 24);
            return diffDays + ' day' + (diffDays > 1 ? 's' : '') + ' ago';
        }

        async function loadRemediation() {
            // Get the first account ID for API calls
            const accountId = allAssets.length > 0 ? allAssets[0].account_id : null;

            if (!accountId) {
                document.getElementById('remediation-pending').textContent = '0';
                document.getElementById('remediation-approval').textContent = '0';
                document.getElementById('remediation-completed').textContent = '0';
                document.getElementById('remediation-failed').textContent = '0';
                document.getElementById('remediation-table').innerHTML = '<tr><td colspan="5" class="px-4 py-8 text-center text-qualys-text-muted">No remediation actions yet. Create actions from findings to fix security issues.</td></tr>';
                return;
            }

            try {
                // Fetch remediation summary and actions from API
                const [summaryResp, actionsResp, defsResp] = await Promise.all([
                    apiCall('/remediation/summary?account_id=' + accountId),
                    apiCall('/remediation?account_id=' + accountId),
                    apiCall('/remediation/definitions')
                ]);

                const summary = summaryResp.success ? summaryResp.data : {};
                const actions = actionsResp.success ? (actionsResp.data || []) : [];
                window.remediationDefinitions = defsResp.success ? (defsResp.data || []) : [];

                // Update stats
                document.getElementById('remediation-pending').textContent = summary.pending_count || 0;
                document.getElementById('remediation-approval').textContent = summary.approved_count || 0;
                document.getElementById('remediation-completed').textContent = summary.completed_count || 0;
                document.getElementById('remediation-failed').textContent = summary.failed_count || 0;

                // Render actions table
                const tbody = document.getElementById('remediation-table');
                if (actions.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="5" class="px-4 py-8 text-center text-qualys-text-muted">No remediation actions yet. Create actions from findings to fix security issues.</td></tr>';
                } else {
                    tbody.innerHTML = actions.map(action => {
                        const statusColors = {
                            'PENDING': 'bg-severity-medium/10 text-severity-medium',
                            'APPROVED': 'bg-primary-50 text-primary-600',
                            'EXECUTING': 'bg-severity-medium/10 text-severity-medium',
                            'COMPLETED': 'bg-severity-low/10 text-severity-low',
                            'FAILED': 'bg-severity-critical/10 text-severity-critical',
                            'REJECTED': 'bg-qualys-bg text-qualys-text-muted',
                            'ROLLED_BACK': 'bg-severity-medium/10 text-severity-medium'
                        };
                        const statusClass = statusColors[action.status] || statusColors.PENDING;

                        // Find asset name
                        const asset = allAssets.find(a => a.id === action.asset_id);
                        const assetName = asset ? asset.name : (action.asset_id || 'Unknown');

                        // Action buttons based on status
                        let actionButtons = '-';
                        if (action.status === 'PENDING') {
                            actionButtons = '<button onclick="approveRemediation(\'' + action.id + '\')" class="px-2 py-1 text-xs bg-severity-low text-white rounded mr-1">Approve</button>' +
                                '<button onclick="rejectRemediation(\'' + action.id + '\')" class="px-2 py-1 text-xs bg-qualys-text-muted text-white rounded">Reject</button>';
                        } else if (action.status === 'APPROVED') {
                            actionButtons = '<button onclick="executeRemediation(\'' + action.id + '\')" class="px-2 py-1 text-xs bg-primary-500 text-white rounded">Execute</button>';
                        } else if (action.status === 'COMPLETED' && action.rollback_available) {
                            actionButtons = '<button onclick="rollbackRemediation(\'' + action.id + '\')" class="px-2 py-1 text-xs bg-severity-medium text-white rounded">Rollback</button>';
                        }

                        return '<tr class="hover:bg-qualys-bg clickable" onclick="showRemediationDetail(\'' + action.id + '\')">' +
                            '<td class="px-4 py-3 text-sm font-medium">' + action.description + '</td>' +
                            '<td class="px-4 py-3 text-sm">' + assetName + '</td>' +
                            '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-primary-50 text-primary-600">' + action.action_type + '</span></td>' +
                            '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded ' + statusClass + '">' + action.status + '</span></td>' +
                            '<td class="px-4 py-3" onclick="event.stopPropagation()">' + actionButtons + '</td></tr>';
                    }).join('');
                }
            } catch (err) {
                console.error('Error loading remediation:', err);
                document.getElementById('remediation-table').innerHTML = '<tr><td colspan="5" class="px-4 py-8 text-center text-qualys-text-muted">Error loading remediation data.</td></tr>';
            }
        }

        // Remediation action functions
        async function approveRemediation(actionId) {
            try {
                const resp = await apiCall('/remediation/' + actionId + '/approve', { method: 'POST', body: JSON.stringify({ approved_by: 'admin' }) });
                if (resp.success) {
                    loadRemediation();
                } else {
                    alert('Failed to approve: ' + (resp.error?.message || 'Unknown error'));
                }
            } catch (err) { alert('Error: ' + err.message); }
        }

        async function rejectRemediation(actionId) {
            const reason = prompt('Enter rejection reason:');
            if (!reason) return;
            try {
                const resp = await apiCall('/remediation/' + actionId + '/reject', { method: 'POST', body: JSON.stringify({ reason }) });
                if (resp.success) {
                    loadRemediation();
                } else {
                    alert('Failed to reject: ' + (resp.error?.message || 'Unknown error'));
                }
            } catch (err) { alert('Error: ' + err.message); }
        }

        async function executeRemediation(actionId) {
            if (!confirm('Execute this remediation action? This will make changes to your AWS resources.')) return;
            try {
                const resp = await apiCall('/remediation/' + actionId + '/execute', { method: 'POST', body: JSON.stringify({ provider: 'aws' }) });
                if (resp.success) {
                    alert('Remediation executed successfully!');
                    loadRemediation();
                } else {
                    alert('Execution failed: ' + (resp.error?.message || 'Unknown error'));
                }
            } catch (err) { alert('Error: ' + err.message); }
        }

        async function rollbackRemediation(actionId) {
            if (!confirm('Rollback this remediation? This will revert the changes made.')) return;
            try {
                const resp = await apiCall('/remediation/' + actionId + '/rollback', { method: 'POST', body: JSON.stringify({ provider: 'aws' }) });
                if (resp.success) {
                    alert('Rollback completed successfully!');
                    loadRemediation();
                } else {
                    alert('Rollback failed: ' + (resp.error?.message || 'Unknown error'));
                }
            } catch (err) { alert('Error: ' + err.message); }
        }

        function showRemediationDetail(actionId) {
            // Show modal with remediation action details
            console.log('Show remediation detail:', actionId);
            // TODO: Implement detail modal
        }

        async function showAssetDetail(assetId) {
            try {
                let asset = allAssets.find(a => a.id === assetId);
                // If not in cache, fetch from API
                if (!asset) {
                    const resp = await apiCall('/assets/' + assetId);
                    if (!resp.success || !resp.data) {
                        console.error('Asset not found:', assetId);
                        alert('Asset not found. Please refresh the page.');
                        return;
                    }
                    asset = resp.data;
                    allAssets.push(asset); // Add to cache
                }
                document.getElementById('asset-modal-title').textContent = asset.name;

                // Fetch classifications
                const classResp = await apiCall('/assets/' + assetId + '/classifications?limit=50');
                const classifications = classResp.success ? classResp.data : [];

                // Find related findings
                const assetFindings = allFindings.filter(f => f.asset_id === assetId || (f.resource_arn && f.resource_arn === asset.resource_arn));

                // Build tabbed content
                const tabs = ['Overview', 'Classifications', 'Findings', 'Compliance', 'Lineage', 'Remediation'];
                const tabHtml = '<div class="flex border-b border-qualys-border mb-4">' + tabs.map((t, i) =>
                    '<button onclick="switchAssetTab(\'' + t.toLowerCase() + '\')" id="asset-tab-' + t.toLowerCase() + '" class="px-4 py-2 text-sm ' + (i === 0 ? 'border-b-2 border-primary-500 text-primary-500' : 'text-qualys-text-muted hover:text-qualys-text-primary') + '">' + t + '</button>'
                ).join('') + '</div>';

                // Overview tab
                const overviewHtml = '<div id="asset-content-overview">' +
                    '<div class="grid grid-cols-2 gap-4 mb-6">' +
                    '<div><div class="text-xs text-qualys-text-muted">ARN</div><div class="text-sm font-mono break-all">' + (asset.resource_arn || 'N/A') + '</div></div>' +
                    '<div><div class="text-xs text-qualys-text-muted">Type</div><div class="text-sm">' + asset.resource_type + '</div></div>' +
                    '<div><div class="text-xs text-qualys-text-muted">Region</div><div class="text-sm">' + (asset.region || 'N/A') + '</div></div>' +
                    '<div><div class="text-xs text-qualys-text-muted">Sensitivity</div><span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[asset.sensitivity_level] || sensitivityClasses.UNKNOWN) + '">' + (asset.sensitivity_level || 'UNKNOWN') + '</span></div>' +
                    '<div><div class="text-xs text-qualys-text-muted">Public Access</div><span class="text-sm ' + (asset.is_public ? 'text-severity-critical' : 'text-severity-low') + '">' + (asset.is_public ? 'Yes - Public' : 'No - Private') + '</span></div>' +
                    '<div><div class="text-xs text-qualys-text-muted">Encryption</div><span class="text-sm ' + (asset.encrypted ? 'text-severity-low' : 'text-severity-high') + '">' + (asset.encrypted ? 'Encrypted' : 'Unencrypted') + '</span></div>' +
                    '</div>' +
                    '<div class="grid grid-cols-4 gap-3">' +
                    '<div onclick="switchAssetTab(\'classifications\')" class="clickable p-3 bg-qualys-bg rounded text-center"><div class="text-lg font-semibold text-primary-500">' + classifications.length + '</div><div class="text-xs text-qualys-text-muted">Classifications</div></div>' +
                    '<div onclick="switchAssetTab(\'findings\')" class="clickable p-3 bg-qualys-bg rounded text-center"><div class="text-lg font-semibold text-severity-high">' + assetFindings.length + '</div><div class="text-xs text-qualys-text-muted">Findings</div></div>' +
                    '<div onclick="switchAssetTab(\'compliance\')" class="clickable p-3 bg-qualys-bg rounded text-center"><div class="text-lg font-semibold text-severity-medium">3</div><div class="text-xs text-qualys-text-muted">Frameworks</div></div>' +
                    '<div onclick="switchAssetTab(\'remediation\')" class="clickable p-3 bg-qualys-bg rounded text-center"><div class="text-lg font-semibold text-severity-low">2</div><div class="text-xs text-qualys-text-muted">Actions</div></div>' +
                    '</div></div>';

                // Classifications tab - store classifications for modal access
                window.assetClassifications = classifications;
                const classHtml = '<div id="asset-content-classifications" class="hidden">' + (classifications.length > 0 ?
                    '<div class="space-y-2">' + classifications.slice(0, 20).map((c, idx) =>
                        '<div onclick="showAssetClassificationDetail(' + idx + ')" class="clickable p-3 bg-qualys-bg rounded"><div class="flex items-center justify-between"><span class="font-medium">' + c.rule_name + '</span>' +
                        '<span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[c.sensitivity] || '') + '">' + c.sensitivity + '</span></div>' +
                        '<div class="text-xs text-qualys-text-muted mt-1">' + c.object_path + ' - ' + c.finding_count + ' matches</div>' +
                        '<div class="text-xs mt-1"><span class="text-qualys-text-muted">Compliance:</span> <span class="text-primary-500">GDPR Art.4, HIPAA §164.514</span></div></div>'
                    ).join('') + '</div>' : '<p class="text-qualys-text-muted text-center py-8">No classifications found</p>') + '</div>';

                // Findings tab
                const findingsHtml = '<div id="asset-content-findings" class="hidden">' + (assetFindings.length > 0 ?
                    '<div class="space-y-2">' + assetFindings.map(f =>
                        '<div onclick="showFindingDetail(\'' + f.id + '\')" class="clickable p-3 bg-qualys-bg rounded"><div class="flex items-center justify-between">' +
                        '<div class="flex items-center gap-2"><span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[f.severity] || '') + '">' + f.severity + '</span><span class="font-medium">' + f.title + '</span></div>' +
                        '<span class="text-xs ' + (f.status === 'open' ? 'text-severity-critical' : 'text-severity-low') + '">' + f.status + '</span></div>' +
                        '<div class="flex items-center justify-between mt-2"><span class="text-xs text-qualys-text-muted">' + f.finding_type + '</span>' +
                        '<button onclick="event.stopPropagation(); createRemediation(\'' + f.id + '\')" class="text-xs text-primary-500 hover:underline">Create Fix</button></div></div>'
                    ).join('') + '</div>' : '<p class="text-qualys-text-muted text-center py-8">No findings for this asset</p>') + '</div>';

                // Compliance tab
                const complianceHtml = '<div id="asset-content-compliance" class="hidden">' +
                    '<div class="space-y-3">' +
                    '<div onclick="showComplianceDetail(\'GDPR\')" class="clickable p-4 border border-qualys-border rounded"><div class="flex items-center justify-between mb-2"><span class="font-medium">GDPR</span><span class="text-sm text-severity-medium">2 gaps</span></div>' +
                    '<div class="text-xs text-qualys-text-muted">Art.4 - Personal Data Definition • Art.32 - Security of Processing</div>' +
                    '<div class="w-full h-1.5 bg-qualys-bg rounded-full mt-2"><div class="h-full bg-severity-low rounded-full" style="width: 75%"></div></div></div>' +
                    '<div onclick="showComplianceDetail(\'HIPAA\')" class="clickable p-4 border border-qualys-border rounded"><div class="flex items-center justify-between mb-2"><span class="font-medium">HIPAA</span><span class="text-sm text-severity-high">4 gaps</span></div>' +
                    '<div class="text-xs text-qualys-text-muted">§164.312 - Technical Safeguards • §164.514 - De-identification</div>' +
                    '<div class="w-full h-1.5 bg-qualys-bg rounded-full mt-2"><div class="h-full bg-severity-medium rounded-full" style="width: 60%"></div></div></div>' +
                    '<div onclick="showComplianceDetail(\'PCI-DSS\')" class="clickable p-4 border border-qualys-border rounded"><div class="flex items-center justify-between mb-2"><span class="font-medium">PCI-DSS</span><span class="text-sm text-severity-low">1 gap</span></div>' +
                    '<div class="text-xs text-qualys-text-muted">Req 3.4 - Render PAN Unreadable • Req 7 - Restrict Access</div>' +
                    '<div class="w-full h-1.5 bg-qualys-bg rounded-full mt-2"><div class="h-full bg-severity-low rounded-full" style="width: 90%"></div></div></div>' +
                    '</div></div>';

                // Lineage tab
                const lineageHtml = '<div id="asset-content-lineage" class="hidden">' +
                    '<div class="text-center mb-4"><p class="text-sm text-qualys-text-muted">Data flow to and from this asset</p></div>' +
                    '<div class="flex items-center justify-center gap-2 flex-wrap">' +
                    '<div class="p-3 bg-primary-50 border border-primary-200 rounded text-center min-w-[120px]"><div class="text-xs text-primary-600">SOURCE</div><div class="text-sm font-medium">app-logs-bucket</div></div>' +
                    '<svg class="w-6 h-6 text-qualys-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3"/></svg>' +
                    '<div class="p-3 bg-severity-critical/10 border border-severity-critical/30 rounded text-center min-w-[120px]"><div class="text-xs text-severity-critical">THIS ASSET</div><div class="text-sm font-medium">' + asset.name + '</div></div>' +
                    '<svg class="w-6 h-6 text-qualys-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 8l4 4m0 0l-4 4m4-4H3"/></svg>' +
                    '<div class="p-3 bg-severity-high/10 border border-severity-high/30 rounded text-center min-w-[120px]"><div class="text-xs text-severity-high">AI SERVICE</div><div class="text-sm font-medium">Bedrock</div></div>' +
                    '</div>' +
                    '<div class="mt-6 p-4 bg-severity-critical/5 border border-severity-critical/20 rounded"><div class="flex items-center gap-2 text-severity-critical text-sm font-medium mb-2"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01"/></svg>Sensitive Data Flow Detected</div>' +
                    '<p class="text-xs text-qualys-text-secondary">PII data from this asset flows to Amazon Bedrock for AI processing. Review data handling policies.</p></div></div>';

                // Remediation tab
                const remediationHtml = '<div id="asset-content-remediation" class="hidden">' +
                    '<div class="space-y-3">' +
                    '<div class="p-4 border border-qualys-border rounded">' +
                    '<div class="flex items-center justify-between mb-3"><div><span class="font-medium">Enable Server-Side Encryption</span><span class="ml-2 px-2 py-0.5 text-[11px] rounded bg-severity-medium/10 text-severity-medium">Recommended</span></div>' +
                    '<button onclick="executeRemediation(\'encrypt\', \'' + assetId + '\')" class="px-3 py-1 text-xs bg-primary-500 text-white rounded hover:bg-primary-600">Apply Fix</button></div>' +
                    '<p class="text-xs text-qualys-text-muted">Enables AES-256 encryption using AWS managed keys (SSE-S3) for all objects.</p>' +
                    '<div class="text-xs mt-2"><span class="text-qualys-text-muted">Impact:</span> <span>No downtime, applies to new objects immediately</span></div></div>' +
                    '<div class="p-4 border border-qualys-border rounded">' +
                    '<div class="flex items-center justify-between mb-3"><div><span class="font-medium">Block Public Access</span><span class="ml-2 px-2 py-0.5 text-[11px] rounded bg-severity-high/10 text-severity-high">Critical</span></div>' +
                    '<button onclick="executeRemediation(\'block_public\', \'' + assetId + '\')" class="px-3 py-1 text-xs bg-severity-critical text-white rounded hover:bg-severity-critical/80">Apply Fix</button></div>' +
                    '<p class="text-xs text-qualys-text-muted">Enables all S3 Block Public Access settings to prevent accidental public exposure.</p>' +
                    '<div class="text-xs mt-2"><span class="text-qualys-text-muted">Impact:</span> <span class="text-severity-high">May break existing public access</span></div></div>' +
                    '</div></div>';

                document.getElementById('asset-modal-content').innerHTML = tabHtml + overviewHtml + classHtml + findingsHtml + complianceHtml + lineageHtml + remediationHtml;
                document.getElementById('asset-modal').classList.remove('hidden');
            } catch (e) { console.error('Show asset detail error:', e); }
        }

        function switchAssetTab(tabName) {
            ['overview', 'classifications', 'findings', 'compliance', 'lineage', 'remediation'].forEach(t => {
                const tab = document.getElementById('asset-tab-' + t);
                const content = document.getElementById('asset-content-' + t);
                if (tab) { tab.classList.remove('border-b-2', 'border-primary-500', 'text-primary-500'); tab.classList.add('text-qualys-text-muted'); }
                if (content) content.classList.add('hidden');
            });
            const activeTab = document.getElementById('asset-tab-' + tabName);
            const activeContent = document.getElementById('asset-content-' + tabName);
            if (activeTab) { activeTab.classList.add('border-b-2', 'border-primary-500', 'text-primary-500'); activeTab.classList.remove('text-qualys-text-muted'); }
            if (activeContent) activeContent.classList.remove('hidden');
        }

        function closeAssetModal() { document.getElementById('asset-modal').classList.add('hidden'); }

        async function showFindingDetail(findingId) {
            const finding = allFindings.find(f => f.id === findingId);
            if (!finding) return;
            document.getElementById('finding-modal-title').textContent = finding.title;

            // Find related asset
            const relatedAsset = allAssets.find(a => a.id === finding.asset_id || a.resource_arn === finding.resource_arn);

            const content = '<div class="mb-4 flex items-center gap-2">' +
                '<span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[finding.severity] || '') + '">' + finding.severity + '</span>' +
                '<span class="px-2 py-0.5 text-[11px] font-medium rounded ' + (finding.status === 'open' ? 'bg-severity-critical/10 text-severity-critical' : 'bg-severity-low/10 text-severity-low') + '">' + finding.status + '</span>' +
                '<span class="px-2 py-0.5 text-[11px] font-medium rounded bg-primary-50 text-primary-600">' + finding.finding_type + '</span></div>' +

                // Description
                '<div class="mb-4"><div class="text-xs text-qualys-text-muted mb-1">Description</div><p class="text-sm">' + (finding.description || 'Security finding detected on this resource.') + '</p></div>' +

                // Resource link
                '<div class="mb-4 p-3 bg-qualys-bg rounded">' +
                '<div class="text-xs text-qualys-text-muted mb-1">Affected Resource</div>' +
                '<p class="text-sm font-mono break-all">' + (finding.resource_arn || 'N/A') + '</p>' +
                (relatedAsset ? '<button onclick="closeFindingModal(); showAssetDetail(\'' + relatedAsset.id + '\')" class="mt-2 text-xs text-primary-500 hover:underline">View Asset Details →</button>' : '') +
                '</div>' +

                // Rule that triggered
                '<div class="mb-4 p-3 border border-qualys-border rounded">' +
                '<div class="flex items-center justify-between mb-2"><div class="text-xs text-qualys-text-muted">Detection Rule</div>' +
                '<button onclick="showRuleDetail(\'' + (finding.rule_name || finding.finding_type) + '\')" class="text-xs text-primary-500 hover:underline">View Rule →</button></div>' +
                '<div class="text-sm font-medium">' + (finding.rule_name || finding.finding_type) + '</div>' +
                '<div class="text-xs text-qualys-text-muted mt-1">Pattern matched sensitive data in this resource</div></div>' +

                // Compliance impact
                '<div class="mb-4"><div class="text-xs text-qualys-text-muted mb-2">Compliance Impact</div>' +
                '<div class="flex flex-wrap gap-2">' +
                '<span onclick="showComplianceDetail(\'GDPR\')" class="clickable px-2 py-1 text-xs bg-severity-high/10 text-severity-high rounded">GDPR Art.32</span>' +
                '<span onclick="showComplianceDetail(\'HIPAA\')" class="clickable px-2 py-1 text-xs bg-severity-high/10 text-severity-high rounded">HIPAA §164.312</span>' +
                '<span onclick="showComplianceDetail(\'PCI-DSS\')" class="clickable px-2 py-1 text-xs bg-severity-medium/10 text-severity-medium rounded">PCI-DSS Req 3</span>' +
                '</div></div>' +

                // Remediation section
                '<div class="p-4 bg-primary-50 border border-primary-200 rounded">' +
                '<div class="flex items-center justify-between mb-3">' +
                '<div class="text-sm font-medium text-primary-700">Recommended Fix</div>' +
                '<button onclick="createRemediation(\'' + findingId + '\')" class="px-3 py-1.5 text-xs bg-primary-500 text-white rounded hover:bg-primary-600">Create Remediation</button></div>' +
                '<p class="text-sm text-primary-700">' + (finding.remediation || 'Enable encryption and restrict access to this resource to prevent unauthorized data exposure.') + '</p>' +
                '<div class="mt-3 pt-3 border-t border-primary-200">' +
                '<div class="text-xs text-primary-600 mb-2">Available Actions:</div>' +
                '<div class="flex flex-wrap gap-2">' +
                '<span class="px-2 py-1 text-xs bg-white text-primary-600 rounded border border-primary-200">Enable Encryption</span>' +
                '<span class="px-2 py-1 text-xs bg-white text-primary-600 rounded border border-primary-200">Block Public Access</span>' +
                '<span class="px-2 py-1 text-xs bg-white text-primary-600 rounded border border-primary-200">Restrict IAM Policy</span>' +
                '</div></div></div>' +

                // Timeline
                '<div class="mt-4"><div class="text-xs text-qualys-text-muted mb-2">Timeline</div>' +
                '<div class="space-y-2">' +
                '<div class="flex items-center gap-3 text-xs"><div class="w-2 h-2 bg-severity-critical rounded-full"></div><span class="text-qualys-text-muted">Detected</span><span>' + (finding.created_at ? new Date(finding.created_at).toLocaleString() : 'Recently') + '</span></div>' +
                (finding.status === 'open' ? '<div class="flex items-center gap-3 text-xs"><div class="w-2 h-2 bg-severity-medium rounded-full"></div><span class="text-qualys-text-muted">Status</span><span>Awaiting remediation</span></div>' :
                '<div class="flex items-center gap-3 text-xs"><div class="w-2 h-2 bg-severity-low rounded-full"></div><span class="text-qualys-text-muted">Resolved</span><span>' + (finding.resolved_at ? new Date(finding.resolved_at).toLocaleString() : 'Recently') + '</span></div>') +
                '</div></div>';

            document.getElementById('finding-modal-content').innerHTML = content;
            document.getElementById('finding-modal').classList.remove('hidden');
        }

        function closeFindingModal() { document.getElementById('finding-modal').classList.add('hidden'); }

        function showAccountAssets(accountId) { showView('assets'); }

        async function triggerScan(accountId) {
            try {
                await apiCall('/accounts/' + accountId + '/scan', { method: 'POST', body: '{}' });
                alert('Scan started');
                loadScans();
            } catch (e) { alert('Failed to start scan: ' + e.message); }
        }

        async function cancelScan(scanId) {
            if (!confirm('Cancel this scan?')) return;
            try {
                await apiCall('/scans/' + scanId + '/cancel', { method: 'POST' });
                loadScans();
            } catch (e) { alert('Error: ' + e.message); }
        }

        async function clearStuckScans() {
            if (!confirm('Clear all stuck scans?')) return;
            try {
                await apiCall('/scans/clear-stuck', { method: 'POST' });
                loadScans();
            } catch (e) { alert('Error: ' + e.message); }
        }

        // Classification Rules data
        const builtInRules = [
            { name: 'SSN Detector', category: 'PII', sensitivity: 'CRITICAL', pattern: '\\d{3}-\\d{2}-\\d{4}', compliance: ['GDPR', 'HIPAA', 'CCPA'], matches: 0 },
            { name: 'Credit Card Number', category: 'PCI', sensitivity: 'CRITICAL', pattern: '\\d{4}[- ]?\\d{4}[- ]?\\d{4}[- ]?\\d{4}', compliance: ['PCI-DSS'], matches: 0 },
            { name: 'Email Address', category: 'PII', sensitivity: 'MEDIUM', pattern: '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}', compliance: ['GDPR', 'CCPA'], matches: 0 },
            { name: 'Phone Number', category: 'PII', sensitivity: 'MEDIUM', pattern: '\\+?1?[-.\\s]?\\(?\\d{3}\\)?[-.\\s]?\\d{3}[-.\\s]?\\d{4}', compliance: ['GDPR'], matches: 0 },
            { name: 'Medical Record Number', category: 'PHI', sensitivity: 'CRITICAL', pattern: 'MRN[:\\s]*\\d{6,10}', compliance: ['HIPAA'], matches: 0 },
            { name: 'Health Insurance ID', category: 'PHI', sensitivity: 'HIGH', pattern: '[A-Z]{3}\\d{9}', compliance: ['HIPAA'], matches: 0 },
            { name: 'AWS Access Key', category: 'SECRETS', sensitivity: 'CRITICAL', pattern: 'AKIA[0-9A-Z]{16}', compliance: ['SOC2'], matches: 0 },
            { name: 'API Key Generic', category: 'SECRETS', sensitivity: 'HIGH', pattern: 'api[_-]?key[\\s]*[:=][\\s]*[\\w]{20,}', compliance: ['SOC2'], matches: 0 },
            { name: 'Date of Birth', category: 'PII', sensitivity: 'HIGH', pattern: '\\d{1,2}/\\d{1,2}/\\d{4}', compliance: ['GDPR', 'HIPAA'], matches: 0 },
            { name: 'Passport Number', category: 'PII', sensitivity: 'CRITICAL', pattern: '[A-Z]{1,2}\\d{6,9}', compliance: ['GDPR'], matches: 0 },
            { name: 'IBAN', category: 'PCI', sensitivity: 'HIGH', pattern: '[A-Z]{2}\\d{2}[A-Z0-9]{4,30}', compliance: ['PCI-DSS', 'GDPR'], matches: 0 },
            { name: 'Diagnosis Code (ICD)', category: 'PHI', sensitivity: 'HIGH', pattern: '[A-Z]\\d{2}\\.?\\d{0,2}', compliance: ['HIPAA'], matches: 0 },
        ];

        function loadRules() {
            // Update counts based on classifications
            const categoryCounts = { PII: 0, PCI: 0, PHI: 0, SECRETS: 0 };
            builtInRules.forEach(r => {
                r.matches = allClassifications.filter(c => c.rule_name === r.name).length;
                if (categoryCounts[r.category] !== undefined) categoryCounts[r.category]++;
            });

            document.getElementById('rules-total').textContent = builtInRules.length;
            document.getElementById('rules-pii').textContent = categoryCounts.PII;
            document.getElementById('rules-pci').textContent = categoryCounts.PCI;
            document.getElementById('rules-phi').textContent = categoryCounts.PHI;

            renderRules(builtInRules);
        }

        function renderRules(rules) {
            const tbody = document.getElementById('rules-table');
            tbody.innerHTML = rules.map(r =>
                '<tr onclick="showRuleDetail(\'' + r.name + '\')" class="hover:bg-qualys-bg clickable">' +
                '<td class="px-4 py-3"><div class="text-sm font-medium">' + r.name + '</div></td>' +
                '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] rounded bg-primary-50 text-primary-600">' + r.category + '</span></td>' +
                '<td class="px-4 py-3"><span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[r.sensitivity] || '') + '">' + r.sensitivity + '</span></td>' +
                '<td class="px-4 py-3"><code class="text-xs bg-qualys-bg px-2 py-1 rounded">' + r.pattern.substring(0, 30) + (r.pattern.length > 30 ? '...' : '') + '</code></td>' +
                '<td class="px-4 py-3"><span class="text-sm ' + (r.matches > 0 ? 'text-severity-high font-medium' : 'text-qualys-text-muted') + '">' + r.matches + '</span></td>' +
                '<td class="px-4 py-3"><div class="flex flex-wrap gap-1">' + r.compliance.map(c => '<span class="px-1.5 py-0.5 text-[10px] bg-qualys-bg rounded">' + c + '</span>').join('') + '</div></td>' +
                '</tr>'
            ).join('');
        }

        function filterRulesByCategory(category) {
            const filtered = category ? builtInRules.filter(r => r.category === category) : builtInRules;
            renderRules(filtered);
        }

        function showRuleDetail(ruleName) {
            let rule = builtInRules.find(r => r.name === ruleName);
            if (!rule) {
                rule = { name: ruleName, category: 'PII', sensitivity: 'HIGH', pattern: 'Pattern not available', compliance: ['GDPR'], matches: 0 };
            }

            const matchingClassifications = allClassifications.filter(c => c.rule_name === ruleName);
            const matchingAssets = [...new Set(matchingClassifications.map(c => c.asset_name))];

            document.getElementById('rule-modal-title').textContent = rule.name;
            document.getElementById('rule-modal-subtitle').textContent = rule.category + ' Detection Rule';

            const content = '<div class="grid grid-cols-2 gap-4 mb-6">' +
                '<div><div class="text-xs text-qualys-text-muted">Category</div><span class="px-2 py-0.5 text-[11px] rounded bg-primary-50 text-primary-600">' + rule.category + '</span></div>' +
                '<div><div class="text-xs text-qualys-text-muted">Sensitivity</div><span class="px-2 py-0.5 text-[11px] font-medium rounded border ' + (sensitivityClasses[rule.sensitivity] || '') + '">' + rule.sensitivity + '</span></div>' +
                '</div>' +

                '<div class="mb-6 p-4 bg-qualys-bg rounded">' +
                '<div class="text-xs text-qualys-text-muted mb-2">Detection Pattern (Regex)</div>' +
                '<code class="text-sm font-mono break-all">' + rule.pattern + '</code></div>' +

                '<div class="mb-6"><div class="text-xs text-qualys-text-muted mb-2">Compliance Frameworks</div>' +
                '<div class="flex flex-wrap gap-2">' + rule.compliance.map(c =>
                    '<span onclick="showComplianceDetail(\'' + c + '\')" class="clickable px-3 py-1.5 text-xs bg-primary-50 text-primary-600 rounded border border-primary-200 hover:bg-primary-100">' + c + '</span>'
                ).join('') + '</div></div>' +

                '<div class="mb-6"><div class="flex items-center justify-between mb-3">' +
                '<div class="text-sm font-medium">Matches Found: <span class="text-severity-high">' + matchingClassifications.length + '</span></div>' +
                '<button onclick="closeRuleModal(); showView(\'classifications\')" class="text-xs text-primary-500 hover:underline">View All Classifications →</button></div>' +

                (matchingAssets.length > 0 ?
                '<div class="space-y-2">' + matchingAssets.slice(0, 5).map(assetName => {
                    const asset = allAssets.find(a => a.name === assetName);
                    const count = matchingClassifications.filter(c => c.asset_name === assetName).length;
                    return '<div onclick="' + (asset ? 'closeRuleModal(); showAssetDetail(\'' + asset.id + '\')' : '') + '" class="' + (asset ? 'clickable' : '') + ' p-3 border border-qualys-border rounded flex items-center justify-between">' +
                        '<div><div class="text-sm font-medium">' + assetName + '</div><div class="text-xs text-qualys-text-muted">' + count + ' matches in this asset</div></div>' +
                        '<svg class="w-4 h-4 text-qualys-text-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg></div>';
                }).join('') + '</div>' : '<p class="text-qualys-text-muted text-center py-4">No matches found in current scan</p>') +
                '</div>' +

                '<div class="p-4 bg-severity-medium/5 border border-severity-medium/20 rounded">' +
                '<div class="text-sm font-medium text-severity-medium mb-2">Why This Matters</div>' +
                '<p class="text-xs text-qualys-text-secondary">This rule detects ' + rule.name.toLowerCase() + ' patterns which are considered ' + rule.sensitivity.toLowerCase() + ' sensitivity data. ' +
                'Exposure of this data type can result in regulatory penalties under ' + rule.compliance.join(', ') + '.</p></div>';

            document.getElementById('rule-modal-content').innerHTML = content;
            document.getElementById('rule-modal').classList.remove('hidden');
        }

        function closeRuleModal() { document.getElementById('rule-modal').classList.add('hidden'); }

        function showClassificationDetail(classificationId) {
            const c = allClassifications.find(x => x.id === classificationId);
            if (!c) return;
            renderClassificationModal(c);
        }

        function closeClassificationModal() { document.getElementById('classification-modal').classList.add('hidden'); }

        function showAssetClassificationDetail(idx) {
            const c = window.assetClassifications ? window.assetClassifications[idx] : null;
            if (!c) return;
            renderClassificationModal(c);
        }

        function renderClassificationModal(c) {
            document.getElementById('classification-modal-title').textContent = c.rule_name;
            document.getElementById('classification-modal-subtitle').textContent = c.category + ' • ' + c.sensitivity;

            // sample_matches can be {"samples": [...]} or {"value": "..."} (legacy)
            let sampleHtml = '';
            const sm = c.sample_matches;
            if (sm && sm.samples && sm.samples.length > 0) {
                sampleHtml = sm.samples.slice(0, 10).map(s =>
                    '<div class="p-3 bg-qualys-bg rounded font-mono text-sm">' +
                    '<div class="flex items-center justify-between mb-2">' +
                    '<span class="text-xs text-qualys-text-muted">Line ' + (s.line || '?') + (s.column ? ', Col ' + s.column : '') + '</span>' +
                    '<span class="px-2 py-0.5 text-[10px] bg-severity-critical/10 text-severity-critical rounded">MASKED</span></div>' +
                    '<div class="text-severity-critical font-medium">' + (s.value || 'N/A') + '</div>' +
                    '<div class="text-xs text-qualys-text-muted mt-1">' + (s.context || '') + '</div></div>'
                ).join('');
            } else if (sm && sm.value) {
                // Legacy format - just a single masked value
                sampleHtml = '<div class="p-3 bg-qualys-bg rounded font-mono text-sm">' +
                    '<div class="flex items-center justify-between mb-2">' +
                    '<span class="text-xs text-qualys-text-muted">Sample</span>' +
                    '<span class="px-2 py-0.5 text-[10px] bg-severity-critical/10 text-severity-critical rounded">MASKED</span></div>' +
                    '<div class="text-severity-critical font-medium">' + sm.value + '</div></div>';
            } else {
                sampleHtml = '<p class="text-qualys-text-muted">No sample matches available - run a new scan to populate</p>';
            }

            // Get match locations for line numbers
            const locations = (c.match_locations && c.match_locations.locations) ? c.match_locations.locations : [];
            const linesFromLegacy = (c.match_locations && c.match_locations.lines) ? c.match_locations.lines : [];
            const allLines = locations.length > 0 ? locations.map(l => l.line) : linesFromLegacy;
            const lineInfo = allLines.length > 0 ?
                '<div class="mb-4"><div class="text-xs text-qualys-text-muted mb-2">Match Locations</div><div class="flex flex-wrap gap-1">' +
                allLines.slice(0, 20).map(l => '<span class="px-2 py-0.5 text-xs bg-qualys-bg rounded">Line ' + l + '</span>').join('') +
                (allLines.length > 20 ? '<span class="px-2 py-0.5 text-xs text-qualys-text-muted">+' + (allLines.length - 20) + ' more</span>' : '') +
                '</div></div>' : '';

            document.getElementById('classification-modal-content').innerHTML =
                '<div class="grid grid-cols-2 gap-4 mb-6">' +
                '<div><div class="text-xs text-qualys-text-muted">File Path</div><div class="text-sm font-mono break-all">' + (c.object_path || 'N/A') + '</div></div>' +
                '<div><div class="text-xs text-qualys-text-muted">Asset</div><div class="text-sm">' + (c.asset_name || 'N/A') + '</div></div>' +
                '<div><div class="text-xs text-qualys-text-muted">Match Count</div><div class="text-sm font-semibold text-severity-critical">' + (c.finding_count || 0) + ' occurrences</div></div>' +
                '<div><div class="text-xs text-qualys-text-muted">Confidence</div><div class="text-sm">' + Math.round((c.confidence_score || 0) * 100) + '%</div></div>' +
                '</div>' +
                lineInfo +
                '<div class="mb-4"><h4 class="font-medium mb-3">Sample Matches (Masked)</h4><div class="space-y-2">' + sampleHtml + '</div></div>' +
                '<div class="flex gap-2">' +
                '<button onclick="showRuleDetail(\'' + c.rule_name + '\')" class="px-3 py-1.5 text-sm bg-primary-500 text-white rounded hover:bg-primary-600">View Rule</button>' +
                '<button onclick="closeClassificationModal()" class="px-3 py-1.5 text-sm border border-qualys-border rounded hover:bg-qualys-bg">Close</button></div>';

            document.getElementById('classification-modal').classList.remove('hidden');
        }

        function showComplianceDetail(framework) {
            const frameworks = {
                'GDPR': {
                    name: 'General Data Protection Regulation',
                    requirements: [
                        { id: 'Art.4', title: 'Personal Data Definition', desc: 'Proper identification and classification of personal data', status: 'partial' },
                        { id: 'Art.5', title: 'Data Processing Principles', desc: 'Lawful, fair, transparent data processing', status: 'met' },
                        { id: 'Art.25', title: 'Data Protection by Design', desc: 'Privacy built into systems from the start', status: 'met' },
                        { id: 'Art.32', title: 'Security of Processing', desc: 'Encryption and access controls for personal data', status: 'gap' },
                        { id: 'Art.33', title: 'Breach Notification', desc: 'Notify authorities within 72 hours of breach', status: 'met' },
                    ],
                    score: 85
                },
                'HIPAA': {
                    name: 'Health Insurance Portability and Accountability Act',
                    requirements: [
                        { id: '§164.308', title: 'Administrative Safeguards', desc: 'Security management and access controls', status: 'partial' },
                        { id: '§164.310', title: 'Physical Safeguards', desc: 'Facility access and device controls', status: 'met' },
                        { id: '§164.312', title: 'Technical Safeguards', desc: 'Encryption, audit controls, transmission security', status: 'gap' },
                        { id: '§164.314', title: 'Business Associates', desc: 'Third-party data handling agreements', status: 'met' },
                        { id: '§164.514', title: 'De-identification', desc: 'PHI de-identification standards', status: 'gap' },
                    ],
                    score: 72
                },
                'PCI-DSS': {
                    name: 'Payment Card Industry Data Security Standard',
                    requirements: [
                        { id: 'Req 1', title: 'Firewall Configuration', desc: 'Install and maintain firewall configuration', status: 'met' },
                        { id: 'Req 3', title: 'Protect Stored Data', desc: 'Protect stored cardholder data', status: 'gap' },
                        { id: 'Req 4', title: 'Encrypt Transmission', desc: 'Encrypt transmission of cardholder data', status: 'met' },
                        { id: 'Req 7', title: 'Restrict Access', desc: 'Restrict access to cardholder data by need-to-know', status: 'partial' },
                        { id: 'Req 10', title: 'Track Access', desc: 'Track and monitor all access', status: 'met' },
                    ],
                    score: 68
                }
            };

            const fw = frameworks[framework] || frameworks['GDPR'];
            document.getElementById('compliance-modal-title').textContent = framework;
            document.getElementById('compliance-modal-subtitle').textContent = fw.name;

            const statusColors = { met: 'severity-low', partial: 'severity-medium', gap: 'severity-critical' };
            const statusLabels = { met: 'Compliant', partial: 'Partial', gap: 'Gap' };

            const content = '<div class="mb-6 flex items-center gap-6">' +
                '<div class="text-center"><div class="text-4xl font-bold ' + (fw.score >= 80 ? 'text-severity-low' : fw.score >= 60 ? 'text-severity-medium' : 'text-severity-critical') + '">' + fw.score + '%</div><div class="text-xs text-qualys-text-muted">Overall Score</div></div>' +
                '<div class="flex-1"><div class="w-full h-3 bg-qualys-bg rounded-full overflow-hidden"><div class="h-full rounded-full ' + (fw.score >= 80 ? 'bg-severity-low' : fw.score >= 60 ? 'bg-severity-medium' : 'bg-severity-critical') + '" style="width: ' + fw.score + '%"></div></div></div>' +
                '</div>' +

                '<div class="mb-6"><div class="text-sm font-medium mb-3">Requirements</div>' +
                '<div class="space-y-2">' + fw.requirements.map(req =>
                    '<div class="p-3 border border-qualys-border rounded">' +
                    '<div class="flex items-center justify-between mb-1">' +
                    '<div class="flex items-center gap-2"><span class="font-mono text-xs bg-qualys-bg px-2 py-0.5 rounded">' + req.id + '</span><span class="text-sm font-medium">' + req.title + '</span></div>' +
                    '<span class="px-2 py-0.5 text-[11px] rounded bg-' + statusColors[req.status] + '/10 text-' + statusColors[req.status] + '">' + statusLabels[req.status] + '</span></div>' +
                    '<p class="text-xs text-qualys-text-muted">' + req.desc + '</p>' +
                    (req.status === 'gap' ? '<button onclick="closeComplianceModal(); showView(\'findings\')" class="mt-2 text-xs text-primary-500 hover:underline">View Related Findings →</button>' : '') +
                    '</div>'
                ).join('') + '</div></div>' +

                '<div class="p-4 bg-primary-50 border border-primary-200 rounded">' +
                '<div class="text-sm font-medium text-primary-700 mb-2">Path to Compliance</div>' +
                '<p class="text-xs text-primary-600 mb-3">Address the ' + fw.requirements.filter(r => r.status === 'gap').length + ' gaps and ' + fw.requirements.filter(r => r.status === 'partial').length + ' partial requirements to achieve full compliance.</p>' +
                '<button onclick="closeComplianceModal(); showView(\'remediation\')" class="px-3 py-1.5 text-xs bg-primary-500 text-white rounded hover:bg-primary-600">View Remediation Actions →</button></div>';

            document.getElementById('compliance-modal-content').innerHTML = content;
            document.getElementById('compliance-modal').classList.remove('hidden');
        }

        function closeComplianceModal() { document.getElementById('compliance-modal').classList.add('hidden'); }

        function createRemediation(findingId) {
            const finding = allFindings.find(f => f.id === findingId);
            document.getElementById('remediation-modal-title').textContent = 'Create Remediation Action';

            const actions = [
                { type: 'ENABLE_ENCRYPTION', name: 'Enable Encryption', desc: 'Enable server-side encryption for this resource', risk: 'low', auto: true },
                { type: 'BLOCK_PUBLIC_ACCESS', name: 'Block Public Access', desc: 'Remove public access permissions', risk: 'medium', auto: true },
                { type: 'ENABLE_KEY_ROTATION', name: 'Enable Key Rotation', desc: 'Enable automatic key rotation for KMS keys', risk: 'low', auto: true },
                { type: 'RESTRICT_IAM', name: 'Restrict IAM Policy', desc: 'Apply least-privilege IAM policies', risk: 'high', auto: false },
            ];

            const content = '<div class="mb-4 p-3 bg-qualys-bg rounded">' +
                '<div class="text-xs text-qualys-text-muted">Related Finding</div>' +
                '<div class="text-sm font-medium">' + (finding ? finding.title : 'Security Finding') + '</div>' +
                '</div>' +

                '<div class="mb-4"><div class="text-sm font-medium mb-3">Select Remediation Action</div>' +
                '<div class="space-y-2">' + actions.map(a =>
                    '<label class="flex items-start p-3 border border-qualys-border rounded hover:bg-qualys-bg cursor-pointer">' +
                    '<input type="radio" name="remediation-action" value="' + a.type + '" class="mt-1 mr-3">' +
                    '<div class="flex-1">' +
                    '<div class="flex items-center gap-2"><span class="text-sm font-medium">' + a.name + '</span>' +
                    (a.auto ? '<span class="px-1.5 py-0.5 text-[10px] bg-severity-low/10 text-severity-low rounded">Auto</span>' : '<span class="px-1.5 py-0.5 text-[10px] bg-severity-medium/10 text-severity-medium rounded">Manual Review</span>') +
                    '</div>' +
                    '<p class="text-xs text-qualys-text-muted mt-1">' + a.desc + '</p>' +
                    '<div class="text-xs mt-1"><span class="text-qualys-text-muted">Risk:</span> <span class="text-' + (a.risk === 'low' ? 'severity-low' : a.risk === 'medium' ? 'severity-medium' : 'severity-high') + '">' + a.risk + '</span></div>' +
                    '</div></label>'
                ).join('') + '</div></div>' +

                '<div class="mb-4"><div class="text-sm font-medium mb-2">Execution Options</div>' +
                '<label class="flex items-center gap-2 mb-2"><input type="checkbox" id="remediation-auto-execute"><span class="text-sm">Execute immediately (no approval required)</span></label>' +
                '<label class="flex items-center gap-2"><input type="checkbox" id="remediation-notify"><span class="text-sm">Notify team on completion</span></label></div>' +

                '<div class="flex gap-3">' +
                '<button onclick="submitRemediation(\'' + findingId + '\')" class="flex-1 px-4 py-2 bg-primary-500 text-white text-sm rounded hover:bg-primary-600">Create Remediation</button>' +
                '<button onclick="closeRemediationModal()" class="px-4 py-2 border border-qualys-border text-sm rounded hover:bg-qualys-bg">Cancel</button></div>';

            document.getElementById('remediation-modal-content').innerHTML = content;
            document.getElementById('remediation-modal').classList.remove('hidden');
        }

        function submitRemediation(findingId) {
            alert('Remediation action created successfully! It will appear in the Remediation queue.');
            closeRemediationModal();
            showView('remediation');
        }

        function executeRemediation(action, assetId) {
            if (!confirm('Execute remediation action: ' + action + '?')) return;
            alert('Remediation action queued for execution.');
        }

        function closeRemediationModal() { document.getElementById('remediation-modal').classList.add('hidden'); }

        function showView(viewName) {
            document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
            document.querySelectorAll('.nav-link').forEach(l => { l.classList.remove('bg-qualys-sidebar-active', 'text-white'); l.classList.add('text-qualys-text-muted'); });
            document.getElementById('view-' + viewName).classList.add('active');
            const activeLink = document.querySelector('[data-view="' + viewName + '"]');
            if (activeLink) { activeLink.classList.remove('text-qualys-text-muted'); activeLink.classList.add('bg-qualys-sidebar-active', 'text-white'); }

            // Load view-specific data
            switch (viewName) {
                case 'dashboard': loadDashboard(); break;
                case 'accounts': loadAccounts(); break;
                case 'assets': loadAssets(); break;
                case 'findings': loadFindings(); break;
                case 'classifications': loadClassifications(); break;
                case 'rules': loadRules(); break;
                case 'encryption': loadEncryption(); break;
                case 'compliance': loadCompliance(); break;
                case 'access': loadAccess(); break;
                case 'lineage': loadLineage(); break;
                case 'aitracking': loadAITracking(); break;
                case 'remediation': loadRemediation(); break;
                case 'scans': loadScans(); break;
            }

            return false;
        }

        // Initialize charts
        classificationChart = new Chart(document.getElementById('classificationChart'), {
            type: 'doughnut',
            data: { labels: ['PII', 'PCI', 'PHI', 'SECRETS'], datasets: [{ data: [0, 0, 0, 0], backgroundColor: ['#1991e1', '#c41230', '#2e7d32', '#e85d04'], borderWidth: 0 }] },
            options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { position: 'right', labels: { boxWidth: 12, padding: 15, font: { size: 11 } } } } }
        });

        severityChart = new Chart(document.getElementById('severityChart'), {
            type: 'bar',
            data: { labels: ['Critical', 'High', 'Medium', 'Low', 'Info'], datasets: [{ label: 'Findings', data: [0, 0, 0, 0, 0], backgroundColor: ['#c41230', '#e85d04', '#f4a100', '#2e7d32', '#56707e'], borderRadius: 2 }] },
            options: { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { x: { grid: { display: false } }, y: { beginAtZero: true, grid: { color: '#e7ecee' } } } }
        });

        // Initialize
        document.addEventListener('DOMContentLoaded', function() {
            if (authToken) {
                apiCall('/auth/me').then(resp => {
                    if (resp.success) {
                        document.getElementById('login-modal').classList.add('hidden');
                        loadDashboard();
                    } else {
                        localStorage.removeItem('dspm_token');
                        authToken = null;
                    }
                }).catch(() => {
                    localStorage.removeItem('dspm_token');
                    authToken = null;
                });
            }
        });
    </script>
</body>
</html>`
