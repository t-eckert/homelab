# S3 Storage and Backup Solutions

Overview of options for integrating S3-compatible storage (like Backblaze B2) with the Kubernetes homelab cluster.

## S3-Compatible Storage for Kubernetes

### Option 1: S3 as Primary Storage (via CSI)

Use a Container Storage Interface (CSI) driver to mount S3 buckets directly as PersistentVolumes:

**Available CSI Drivers:**
- [CSI-S3](https://github.com/yandoo/csi-s3) - General purpose S3 CSI driver
- [Mountpoint for S3](https://github.com/awslabs/mountpoint-s3-csi-driver) - AWS's official S3 CSI driver

**Limitations:**
- S3 is object storage, not block storage
- Poor performance for databases and applications requiring POSIX filesystem features
- Higher latency than local disk
- Not suitable for applications with frequent random writes

**Best Use Cases:**
- Static files and media
- Application logs and archives
- Infrequently accessed data

### Option 2: Hybrid Storage (Recommended)

Combine local and S3 storage:
- Keep `local-path-provisioner` for primary storage (databases, application state)
- Add S3-backed StorageClass for specific workloads (media, backups, archives)
- Best of both worlds: performance where needed, cost-effective archival storage

## Backup Services (Recommended Approach)

Rather than using S3 as primary storage, implement backup solutions that copy data from local volumes to remote S3-compatible storage.

### 1. Velero (Kubernetes-Native Backups)

**Features:**
- Kubernetes-native backup and restore
- Supports Backblaze B2 and any S3-compatible storage
- Backs up entire namespaces, PersistentVolumes, or specific resources
- Scheduled backups via CronSchedules
- Disaster recovery and cluster migration capabilities

**Best For:**
- Cluster-wide backup strategy
- Disaster recovery
- Namespace/resource-level backups

**Installation:**
```bash
# Install Velero CLI
brew install velero

# Install Velero in cluster with Backblaze B2
velero install \
  --provider aws \
  --plugins velero/velero-plugin-for-aws:v1.8.0 \
  --bucket my-velero-backups \
  --backup-location-config region=us-west-004,s3Url=https://s3.us-west-004.backblazeb2.com \
  --secret-file ./credentials-velero
```

### 2. Restic + CronJob

**Features:**
- Lightweight incremental backups
- Deduplication and encryption
- Direct S3 support
- Can run as Kubernetes CronJobs

**Best For:**
- Specific PersistentVolume backups
- Custom backup schedules
- File-level granularity

**Example CronJob:**
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-to-s3
  namespace: backups
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: restic-backup
            image: restic/restic:latest
            env:
            - name: RESTIC_REPOSITORY
              value: "s3:https://s3.us-west-004.backblazeb2.com/my-backup-bucket"
            - name: RESTIC_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: restic-secrets
                  key: password
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: b2-credentials
                  key: keyId
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: b2-credentials
                  key: applicationKey
            volumeMounts:
            - name: data
              mountPath: /data
              readOnly: true
            command:
            - /bin/sh
            - -c
            - |
              restic backup /data
              restic forget --keep-daily 7 --keep-weekly 4 --keep-monthly 6
              restic prune
          volumes:
          - name: data
            persistentVolumeClaim:
              claimName: service-data-pvc
          restartPolicy: OnFailure
```

### 3. Longhorn (Distributed Storage)

**Features:**
- Distributed block storage for Kubernetes
- Built-in S3 backups
- Snapshots and disaster recovery
- Volume replication

**Considerations:**
- Designed for multi-node clusters
- Overkill for single-node setup
- Worth considering if cluster expands

### 4. Service-Specific Backups

Custom backup solutions per service type:

**PostgreSQL:**
```yaml
# CronJob for pg_dump to S3
command:
- /bin/sh
- -c
- |
  pg_dump -h postgres -U admin dbname | \
  gzip | \
  aws s3 cp - s3://bucket/backups/postgres-$(date +%Y%m%d).sql.gz
```

**MySQL:**
```yaml
# CronJob for mysqldump to S3
command:
- /bin/sh
- -c
- |
  mysqldump -h mysql -u root -p$MYSQL_ROOT_PASSWORD dbname | \
  gzip | \
  aws s3 cp - s3://bucket/backups/mysql-$(date +%Y%m%d).sql.gz
```

**Files (rclone):**
```yaml
# CronJob for rclone sync to S3
containers:
- name: rclone-backup
  image: rclone/rclone:latest
  command:
  - rclone
  - sync
  - /data
  - b2:my-bucket/backups/
  volumeMounts:
  - name: data
    mountPath: /data
    readOnly: true
```

## Recommended Architecture

```
┌─────────────────────────────────────┐
│  Kubernetes Cluster                 │
│  ┌───────────────────────────────┐  │
│  │ Apps with local PVs           │  │
│  │ (fast, local-path-provisioner)│  │
│  └───────────────────────────────┘  │
│           │                          │
│           │ backup                   │
│           ▼                          │
│  ┌───────────────────────────────┐  │
│  │ Velero or Restic CronJobs     │  │
│  └───────────────────────────────┘  │
└─────────────┬───────────────────────┘
              │
              ▼
     ┌─────────────────┐
     │  Backblaze B2   │
     │  (S3-compatible)│
     └─────────────────┘
```

## Implementation Recommendations

For this single-node homelab cluster:

1. **Primary Storage**: Continue using `local-path-provisioner` for performance-critical workloads
2. **Backups**: Implement Velero for cluster-wide backups to Backblaze B2
3. **Service-Specific**: Add custom CronJobs for critical services (databases, important state)
4. **Future**: Consider S3 CSI for media/archival workloads if needed

## Next Steps

1. Set up Backblaze B2 bucket
2. Install and configure Velero with Backblaze backend
3. Create backup schedules for critical namespaces
4. Test restore procedures
5. Document backup and restore runbooks

## Resources

- [Velero Documentation](https://velero.io/docs/)
- [Restic Documentation](https://restic.readthedocs.io/)
- [Backblaze B2 with Velero](https://help.backblaze.com/hc/en-us/articles/115003105754-Using-Backblaze-B2-with-Velero)
- [CSI-S3 GitHub](https://github.com/yandoo/csi-s3)
