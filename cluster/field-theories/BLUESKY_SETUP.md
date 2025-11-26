# BlueSky Poster Setup Guide

This guide walks you through deploying the BlueSky auto-poster to your homelab.

## Prerequisites

- Field Theories blog is published and accessible at https://fieldtheories.blog
- RSS feed is available at https://fieldtheories.blog/rss.xml
- BlueSky account created

## Step 1: Create BlueSky App Password

1. Log in to BlueSky at https://bsky.app
2. Go to Settings → App Passwords: https://bsky.app/settings/app-passwords
3. Click "Add App Password"
4. Name it something like "Field Theories Auto-Poster"
5. Copy the generated password (format: `xxxx-xxxx-xxxx-xxxx`)
6. **Save this password securely** - you won't be able to see it again!

## Step 2: Update the Secret

Edit `bluesky-poster-secret.yaml` and replace the placeholder values:

```yaml
stringData:
  handle: "fieldtheories.bsky.social"  # Your actual BlueSky handle
  password: "abcd-efgh-ijkl-mnop"      # The app password you just created
```

**Important**: This file contains sensitive credentials. Make sure it's in `.gitignore` or encrypt it before committing!

## Step 3: Push Code to Trigger Container Build

The BlueSky poster container will be automatically built when you push to the field-theories repository:

```bash
cd /path/to/field-theories
git add bluesky-poster/
git commit -m "Add BlueSky auto-poster"
git push
```

Monitor the GitHub Actions workflow at:
https://github.com/t-eckert/field-theories/actions

Wait for the build to complete and the container to be pushed to GHCR.

## Step 4: Configure Date Filter (Optional but Recommended)

To prevent posting your entire back catalog on first run, set a cutoff date in `bluesky-poster-cronjob.yaml`:

```yaml
- name: POST_AFTER_DATE
  value: "2025-11-24T00:00:00Z"  # Only post items published after this date
```

**How to choose the date**:
- Use today's date if you only want to share new posts going forward
- Use the date of your most recent post if you want to share just that one
- Format must be RFC 3339: `YYYY-MM-DDTHH:MM:SSZ`

**Example**: To only post content published from November 24, 2025 onwards:
```
POST_AFTER_DATE=2025-11-24T00:00:00Z
```

**Note**: If you omit this variable, ALL posts in your RSS feed will be posted when the job first runs!

## Step 5: Apply Kubernetes Manifests

Once the container is built and you've configured the date filter, apply the manifests in order:

```bash
# Apply PersistentVolumeClaim
kubectl apply -f bluesky-poster-pvc.yaml

# Apply Secret (make sure you updated the credentials first!)
kubectl apply -f bluesky-poster-secret.yaml

# Apply CronJob (make sure you set POST_AFTER_DATE!)
kubectl apply -f bluesky-poster-cronjob.yaml
```

## Step 6: Test with a Manual Job Run

Trigger a manual job to test before waiting for the hourly schedule:

```bash
kubectl create job --from=cronjob/bluesky-poster bluesky-poster-test -n field-theories
```

Watch the job:

```bash
kubectl get jobs -n field-theories -w
```

Check the logs:

```bash
kubectl logs -n field-theories -l job-name=bluesky-poster-test
```

Expected output:
```
INFO Starting BlueSky poster
INFO State store initialized at /data/posted_items.db
INFO State store has 0 previously posted items
INFO Fetching RSS feed from https://fieldtheories.blog/rss.xml
INFO Found XX items in RSS feed
INFO Found X new items to post
INFO Authenticating with BlueSky as fieldtheories.bsky.social
INFO Successfully authenticated with BlueSky
INFO Posting to BlueSky: Post Title
INFO Successfully posted to BlueSky: Post Title
INFO Posting complete. Posted: X, Failed: 0
```

## Step 7: Verify on BlueSky

1. Go to your BlueSky profile
2. Check that your latest blog posts have been shared (only those after the cutoff date)
3. Verify the format looks correct: `Title: Subtitle\n\nLink`

## Step 8: Monitor

The CronJob will now run every hour automatically. You can monitor it with:

```bash
# List recent jobs
kubectl get jobs -n field-theories

# Get CronJob status
kubectl get cronjob bluesky-poster -n field-theories

# View logs from latest job
kubectl logs -n field-theories -l app=bluesky-poster --tail=100
```

## Troubleshooting

### Job fails with authentication error

- Double-check the BlueSky handle and app password in the Secret
- Verify the Secret is correctly mounted: `kubectl describe pod -n field-theories -l app=bluesky-poster`
- Try generating a new app password

### Job fails to fetch RSS feed

- Verify the blog is accessible: `curl https://fieldtheories.blog/rss.xml`
- Check network connectivity from the cluster
- Review pod logs for specific error messages

### Posts not appearing on BlueSky

- Check the job logs for posting errors
- Verify you're looking at the correct BlueSky account
- Check if BlueSky's API is experiencing issues

### Database issues

- Check if the PVC is correctly mounted: `kubectl describe pvc bluesky-poster-data -n field-theories`
- Verify the `/data` directory has correct permissions
- Check pod logs for SQLite errors

### No posts being shared despite new content

- Check the `POST_AFTER_DATE` setting - it might be filtering out your posts
- View logs to see if posts are being skipped: `kubectl logs -n field-theories -l app=bluesky-poster --tail=50`
- Look for messages like "Skipping item published before cutoff date"
- If needed, update the date or remove the variable entirely

## Updating

To update the poster with new code:

1. Make changes in the `field-theories/bluesky-poster/` directory
2. Commit and push to trigger a new container build
3. Kubernetes will automatically use the new `:latest` tag on the next job run
4. Or manually trigger: `kubectl create job --from=cronjob/bluesky-poster bluesky-poster-manual -n field-theories`

## Uninstalling

To remove the BlueSky poster:

```bash
kubectl delete cronjob bluesky-poster -n field-theories
kubectl delete secret bluesky-poster-secret -n field-theories
kubectl delete pvc bluesky-poster-data -n field-theories
```

## Schedule Customization

To change the posting frequency, edit `bluesky-poster-cronjob.yaml`:

```yaml
spec:
  schedule: "0 * * * *"  # Current: Every hour
  # schedule: "0 */6 * * *"  # Every 6 hours
  # schedule: "0 9 * * *"    # Daily at 9 AM
  # schedule: "*/30 * * * *" # Every 30 minutes
```

Then reapply: `kubectl apply -f bluesky-poster-cronjob.yaml`
