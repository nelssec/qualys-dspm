targetScope = 'subscription'

@description('Display name for the DSPM application registration')
param appDisplayName string = 'Qualys DSPM Scanner'

@description('Resource group for DSPM connector resources')
param resourceGroupName string = 'qualys-dspm-connector-rg'

@description('Location for resources')
param location string = 'eastus'

@description('Tags to apply to resources')
param tags object = {
  Purpose: 'Qualys DSPM'
  ManagedBy: 'Bicep'
}

resource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: resourceGroupName
  location: location
  tags: tags
}

module connector 'modules/connector.bicep' = {
  scope: rg
  name: 'dspm-connector'
  params: {
    location: location
    tags: tags
  }
}

output instructions string = '''

Azure Connector Created Successfully

Next Steps:
1. Create an App Registration in Azure AD:
   - Go to Azure Portal, Azure Active Directory, App registrations
   - Click "New registration"
   - Name: "Qualys DSPM Scanner"
   - Create a client secret

2. Assign Reader role at subscription level:
   az role assignment create \
     --assignee <app-id> \
     --role "Reader" \
     --scope /subscriptions/<subscription-id>

3. Assign Storage Blob Data Reader for storage access:
   az role assignment create \
     --assignee <app-id> \
     --role "Storage Blob Data Reader" \
     --scope /subscriptions/<subscription-id>

4. Add to DSPM Dashboard, Accounts, Add Account:
   - Select "Azure"
   - Enter Subscription ID, Tenant ID, Client ID, Client Secret

'''
