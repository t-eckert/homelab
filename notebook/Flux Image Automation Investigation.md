# Flux Image Automation Investigation

## Investigation Date
December 3, 2025

## Problem Statement
Flux ImageRepository is consistently not detecting the latest versions of the Field Theories container image. Manual updates are required despite having image automation configured.

---

## Environment

**Flux Version:**
- Image Reflector Controller: `v0.35.2`
- Image Update Automation: Enabled

**Image Repository:**
- Registry: `ghcr.io/t-eckert/field-theories`
- Total Tags: 80
- Tag Format: `YYYYMMDD-HHMM-sha-xxxxxxx`

**Current State:**
- ImagePolicy reports: `20251111-0358-sha-4c091d3` (November 11, 2025)
- Actually deployed: `20251203-0536-sha-1972e38` (December 3, 2025 - manually updated)
- Latest available: `20251203-0536-sha-1972e38`

---

## Investigation Findings

### 1. ImageRepository Behavior

**Status:**
```yaml
status:
  lastScanResult:
    latestTags:
    - sha-f03fe0f
    - sha-e70e789
    - sha-e3a65ab
    # ... 7 more sha-only tags
    scanTime: "2025-12-03T13:56:47Z"
    tagCount: 80
```

**Key Finding:** The `latestTags` field only shows 10 tags, all in `sha-xxxxx` format. None match the timestamped pattern `YYYYMMDD-HHMM-sha-xxxxxxx`.

**Controller Logs:**
```
"no new tags found, next scan in 5m0s"
"Latest image tag resolved to 20251111-0358-sha-4c091d3"
```

The controller scans every 5 minutes but consistently resolves to the same November 11 tag.

---

### 2. Root Cause Analysis

#### Issue #1: `latestTags` is a Sample, Not "Latest"

According to [Flux documentation](https://fluxcd.io/flux/components/image/imagerepositories/) and [GitHub Issue #443](https://github.com/fluxcd/image-reflector-controller/issues/443):

> The `latestTags` field shows a **sample** of the latest tags that were read in the last scan, not necessarily the most recent by date or semantic version. The sorting is **alphabetical**.

**Implication:**
- ImageRepository found 80 tags total
- Only 10 are displayed in status (`latestTags`)
- These 10 happen to be `sha-*` format tags (alphabetically after `202512...`)
- The timestamped tags exist but aren't in the sample shown

#### Issue #2: ImagePolicy Should See All Tags (But Doesn't)

The ImagePolicy is **supposed to** evaluate against **all 80 tags** in the ImageRepository's internal database, not just the 10 in `latestTags`.

However, logs show:
```
Latest image tag resolved to 20251111-0358-sha-4c091d3
```

This tag is from **November 11**, despite newer tags from December 2 and December 3 existing.

**Hypothesis:** The ImageRepository's internal tag database may not be properly updating, or there's a filtering/sorting issue preventing the ImagePolicy from seeing newer tags.

---

### 3. Pattern Validation

**Pattern:** `^[0-9]{8}-[0-9]{4}-sha-[a-f0-9]+$`

**Test Results:**
```bash
# Matches timestamped tags ✅
echo "20251203-0536-sha-1972e38" | grep -E '^[0-9]{8}-[0-9]{4}-sha-[a-f0-9]+$'
✅ Match

# Rejects sha-only tags ✅
echo "sha-f03fe0f" | grep -E '^[0-9]{8}-[0-9]{4}-sha-[a-f0-9]+$'
❌ No match
```

**Conclusion:** The regex pattern is correct and should match the timestamped tags.

---

### 4. Alphabetical Sorting Analysis

**Current Policy:**
```yaml
policy:
  alphabetical:
    order: desc  # Newest = highest
filterTags:
  pattern: ^[0-9]{8}-[0-9]{4}-sha-[a-f0-9]+$
```

**Tag Sort Order (descending):**
```
20251203-0536-sha-1972e38  ← Should be selected (Dec 3)
20251202-1650-sha-f9b7c89  ← (Dec 2)
20251128-2006-sha-fe3d065  ← (Nov 28)
20251111-0358-sha-4c091d3  ← Currently selected (Nov 11)
```

**Finding:** Alphabetical descending order is correct for this timestamp format. The ImagePolicy **should** be selecting the Dec 3 tag, but it's not.

---

### 5. Known Flux Limitations

From research ([Issue #159](https://github.com/fluxcd/image-reflector-controller/issues/159)):

> **1000 Tag Limit:** ImageRepository will not pick up any more tags after it finds 1000 tags. If a new image is pushed after hitting this limit, Flux won't see it.

**Current Status:** Only 80 tags exist (well under 1000 limit). This is not the issue.

---

## Possible Causes

### Most Likely Cause: ImageRepository Database Staleness

The ImageRepository may have scanned and stored all 80 tags initially, but newer tags added after that initial scan are not being picked up. Possible reasons:

1. **Registry API pagination issue**: GHCR may be paginating results, and Flux only reads the first page
2. **Tag timestamp caching**: The controller may be using cached tag metadata
3. **Scan optimization**: Flux may skip re-scanning tags it thinks haven't changed

### Secondary Cause: Dual Tagging Strategy Confusion

The repository has **two tag formats:**
- `YYYYMMDD-HHMM-sha-xxxxxxx` (timestamped)
- `sha-xxxxxxx` (SHA-only)

When the same commit gets both tags, the registry may be presenting them in an order that confuses Flux's internal processing.

---

## Web Research Summary

### Key Resources:
1. [Flux Image Policies Documentation](https://fluxcd.io/flux/components/image/imagepolicies/)
2. [How to make sortable image tags](https://fluxcd.io/flux/guides/sortable-image-tags/)
3. [GitHub Issue #443: ImageRepository latestTags sorting](https://github.com/fluxcd/image-reflector-controller/issues/443)
4. [GitHub Issue #159: 1000 tag limit](https://github.com/fluxcd/image-reflector-controller/issues/159)
5. [GitHub Issue #2081: Image policy not displaying latest tag](https://github.com/fluxcd/flux2/issues/2081)

### Best Practices Found:

**For Timestamp-Based Tags:**
- Use RFC3339 timestamps: `2025-12-03T05:36:00Z`
- Or use `YYYYMMDD.HHMMSS` format
- With alphabetical ordering, order: `desc`

**Alternative: Numerical Ordering**
```yaml
filterTags:
  pattern: '^(?P<ts>[0-9]{8}-[0-9]{4})-sha-[a-f0-9]+$'
  extract: '$ts'
policy:
  numerical:
    order: asc
```

This extracts just the timestamp portion and sorts numerically.

---

## Recommended Solutions

### Solution 1: Switch to Numerical Ordering with Extract (Recommended)

Update `cluster/apps/field-theories/fieldtheories-image-policy.yaml`:

```yaml
apiVersion: image.toolkit.fluxcd.io/v1beta2
kind: ImagePolicy
metadata:
  name: field-theories
  namespace: field-theories
spec:
  imageRepositoryRef:
    name: field-theories
  filterTags:
    pattern: '^(?P<ts>[0-9]{8}-[0-9]{4})-sha-[a-f0-9]+$'
    extract: '$ts'
  policy:
    numerical:
      order: asc
```

**Why this helps:**
- Extracts timestamp: `20251203-0536` from tag `20251203-0536-sha-1972e38`
- Compares numerically: `202512030536` > `202512021650` > `202511110358`
- More explicit about what we're comparing

### Solution 2: Force ImageRepository Rescan

Delete and recreate the ImageRepository to force a fresh scan:

```bash
# Backup current config
kubectl get imagerepository field-theories -n field-theories -o yaml > /tmp/imagerepo-backup.yaml

# Delete (forces fresh scan on recreate)
kubectl delete imagerepository field-theories -n field-theories

# Recreate
kubectl apply -f cluster/apps/field-theories/fieldtheories-image-repository.yaml

# Wait for scan
kubectl wait --for=condition=Ready imagerepository/field-theories -n field-theories --timeout=2m

# Check policy
kubectl get imagepolicy field-theories -n field-theories -o jsonpath='{.status.latestImage}'
```

### Solution 3: Add Exclusion for SHA-Only Tags

Update `cluster/apps/field-theories/fieldtheories-image-repository.yaml`:

```yaml
spec:
  exclusionList:
  - ^.*\.sig$
  - ^sha-[a-f0-9]+$  # Exclude SHA-only tags
```

**Why this helps:**
- Reduces noise by excluding the `sha-*` format tags
- Ensures only timestamped tags are considered
- May help with internal tag processing

### Solution 4: Increase Scan Frequency (Temporary)

```yaml
spec:
  interval: 1m  # Down from 5m
```

While this doesn't fix the root cause, it ensures scans happen more frequently so manual reconciliation can pick up new tags sooner.

---

## Testing Plan

1. **Apply Solution 1** (numerical ordering)
2. **Wait 2 minutes** for ImagePolicy reconciliation
3. **Check status:**
   ```bash
   kubectl get imagepolicy field-theories -n field-theories \
     -o jsonpath='{.status.latestImage}'
   ```
4. **Expected:** Should show `20251203-0536-sha-1972e38`
5. **If still wrong:** Apply Solution 2 (force rescan)
6. **If still wrong:** Apply Solution 3 (exclude sha tags)

---

## Immediate Workaround

Until the automation is fixed, manually update images:

```bash
# Check for new tags
curl -s "https://ghcr.io/v2/t-eckert/field-theories/tags/list" | jq -r '.tags[]' | grep -E '^[0-9]{8}' | sort -r | head -5

# Update deployment
kubectl set image deployment/field-theories field-theories=ghcr.io/t-eckert/field-theories:<NEW_TAG> -n field-theories

# Or edit the deployment YAML and git commit
```

---

## Long-Term Recommendations

1. **Simplify tagging strategy:**
   - Remove dual-tagging (`sha-*` + `YYYYMMDD-HHMM-sha-*`)
   - Use only timestamped format
   - Or use semantic versioning (e.g., `v1.2.3`)

2. **Consider semantic versioning:**
   ```yaml
   policy:
     semver:
       range: '>=0.0.0'
   ```
   Benefits: Industry standard, well-supported by Flux

3. **Monitor Flux controller health:**
   - Set up alerts for ImagePolicy stuck on old versions
   - Track `imagepolicy_reconcile_duration_seconds` metrics

4. **Upgrade Flux regularly:**
   - Current: v0.35.2
   - Check for updates with bug fixes for tag scanning

---

## Questions for Further Investigation

1. Why did manual updates work (Dec 2, Dec 3) but automation stopped at Nov 11?
2. Was there a change in tagging strategy around Nov 11 that confused Flux?
3. Are there any Flux admission webhook or policies preventing updates?
4. Is there a Git repository state issue (ImageUpdateAutomation writing back)?

---

## Related Issues

- [fluxcd/image-reflector-controller#443](https://github.com/fluxcd/image-reflector-controller/issues/443) - latestTags not sorted by semver
- [fluxcd/image-reflector-controller#159](https://github.com/fluxcd/image-reflector-controller/issues/159) - 1000 tag limit
- [fluxcd/flux2#2081](https://github.com/fluxcd/flux2/issues/2081) - Policy not displaying latest tag
- [fluxcd/flux2#3244](https://github.com/fluxcd/flux2/issues/3244) - Cannot determine latest tag

---

## Conclusion

The Flux image automation is not seeing newer tags due to either:
1. ImageRepository database not updating with new tags (most likely)
2. ImagePolicy filter/sort logic not properly evaluating all available tags

**Recommended Action:** Apply Solution 1 (numerical ordering with extract) first, as it's the most robust approach and aligns with Flux best practices. If that doesn't resolve it, proceed with Solution 2 (force rescan).

The current pattern is valid, and alphabetical ordering should work, but switching to numerical ordering makes the intent clearer and may work around internal Flux sorting issues.
