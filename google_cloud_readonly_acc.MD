⏺ Via gcloud CLI

  # 1. Create the service account
  gcloud iam service-accounts create claude-readonly \
    --display-name="Claude Code Read-Only" \
    --project=YOUR_PROJECT_ID

  # 2. Grant viewer role (read-only access to all resources)
  gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:claude-readonly@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/viewer"

  # 3. Create and download a key file
  gcloud iam service-accounts keys create ~/claude-gcp-key.json \
    --iam-account="claude-readonly@YOUR_PROJECT_ID.iam.gserviceaccount.com"

  Then point the MCP server at the key:
  export GOOGLE_APPLICATION_CREDENTIALS=~/claude-gcp-key.json
  
  Via Web Console

  1. Go to IAM & Admin → Service Accounts
  2. Click Create Service Account
  3. Name it (e.g. claude-readonly), click Create and Continue
  4. In "Grant access" step, add role Viewer (roles/viewer), click Done
  5. Click the service account → Keys tab → Add Key → Create new key → JSON
  6. Download the JSON key file