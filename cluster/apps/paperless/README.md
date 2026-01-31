# Paperless NGX

Document management system for digitizing and organizing physical documents.

## Access

Paperless is available via Tailscale at: `http://paperless.feist-gondola.ts.net:8000`

## Storage

- **Data**: `/usr/src/paperless/data` (5Gi) - Application data and configuration
- **Media**: `/usr/src/paperless/media` (50Gi) - Document storage
- **Consume**: `/usr/src/paperless/consume` (5Gi) - Document intake folder
- **Export**: `/usr/src/paperless/export` (5Gi) - Document export folder

## Services

- **Paperless Web**: Main application interface
- **Consumer**: Background task processor for document OCR and processing
- **Redis**: Task queue and caching
- **Gotenberg**: PDF conversion service
- **Tika**: Document parsing and metadata extraction

## Configuration

- **Database**: PostgreSQL (shared infrastructure)
- **Timezone**: America/New_York
- **OCR Language**: English (eng)
- **User/Group**: 1000:1000 (for proper volume permissions)

## First Setup

1. Access the web interface via Tailscale URL
2. Create initial admin user when prompted
3. Configure document consumption settings
4. Set up any additional users or groups as needed

## Database

Uses shared PostgreSQL instance in `postgres` namespace with dedicated `paperless` database.

## Resources

- CPU: 100m request / 1000m limit
- Memory: 512Mi request / 2Gi limit (per pod)