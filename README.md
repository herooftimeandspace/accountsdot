# Go Employee Provisioner

Go Employee Provisioner is a self-hosted, mission-critical employee provisioning service designed around PostgreSQL-backed orchestration, resilient provider integrations, and real-time operational visibility.

## LLM Usage Disclaimer
This repository is an LLM-driven project and was written with heavy use of LLMs. This disclaimer is required project policy and must remain present in the repository.

## Local Development
Local testing is supported through either `docker compose` or the VS Code Dev Containers extension.

### Docker Compose
1. Copy `.env.example` to `.env` and adjust values if needed.
2. Start the local stack:
   ```bash
   make up
   ```
3. Run tests inside the app container:
   ```bash
   make test
   ```
4. Stop the stack:
   ```bash
   make down
   ```

### VS Code Dev Containers
1. Install the Dev Containers extension.
2. Open this folder in VS Code.
3. Run `Dev Containers: Reopen in Container`.
4. Inside the container, run:
   ```bash
   make test
   ```

## Test Commands
- `make test-unit`
- `make test-contract`
- `make test-integration`
- `make test`

## Environment Variables
Required or commonly used local variables:

- `APP_ENV`
- `APP_PORT`
- `DATABASE_URL`
- `SESSION_SECRET`
- `ENCRYPTION_KEY`
- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REDIRECT_URL`
- `GOOGLE_ALLOWED_GROUPS_CONFIG`
- `GOOGLE_SHEETS_SPREADSHEET_ID`
- `GOOGLE_SERVICE_ACCOUNT_JSON`
- `ZOOM_ACCOUNT_ID`
- `ZOOM_CLIENT_ID`
- `ZOOM_CLIENT_SECRET`
- `ZOOM_BASE_URL`
- `ZOOM_OAUTH_URL`
- `AERIES_BASE_URL`
- `AERIES_CLIENT_ID`
- `AERIES_CLIENT_SECRET`
- `SFTP_HOST`
- `SFTP_PORT`
- `SFTP_USERNAME`
- `SFTP_PRIVATE_KEY`
- `SFTP_REMOTE_PATH`
- `USE_MOCK_ZOOM`
- `USE_MOCK_GOOGLE`
- `USE_MOCK_AERIES`
- `USE_MOCK_SFTP`
- `ZOOM_SLG_MAX_MEMBERS`

## Local Testing Notes
- Local mode defaults all `USE_MOCK_*` flags to `true`.
- Real third-party integrations are opt-in and should remain disabled for normal local TDD work.
- The local stack is intentionally lean: app, worker, and postgres are enough for baseline development.
