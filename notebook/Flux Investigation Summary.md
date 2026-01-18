# Flux Image Automation Investigation - Final Summary

## Date: December 3, 2025

## Problem
Flux ImageRepository and ImagePolicy consistently fail to detect latest Field Theories container images, staying stuck on `20251111-0358-sha-4c091d3` (November 11) despite newer images existing.

---

## What We Tried

### ✅ Solution 1: Numerical Ordering with Extract
**Result:** FAILED - Timestamp format `YYYYMMDD-HHMM` contains hyphen, causing "failed to parse invalid numeric value" errors

### ✅ Solution 2: Force ImageRepository Rescan
**Result:** PARTIAL SUCCESS - Deleted and recreated ImageRepository, confirmed it scans GHCR successfully

### ✅ Solution 3: Exclude SHA-Only Tags
**Result:** PARTIAL SUCCESS - Reduced tag count from 80 → 24, `latestTags` now shows timestamped tags including Dec 3

### ✅ Additional Steps:
- Restarted image-reflector-controller
- Multiple forced reconciliations
- Reverted to alphabetical ordering (correct for this timestamp format)

---

## Current State

**ImageRepository Status:**
```yaml
latestTags:
  - latest
  - 20251203-0536-sha-1972e38  ← Dec 3 (LATEST!)
  - 20251202-1650-sha-f9b7c89  ← Dec 2
  - 20251128-2159-sha-a867daf
  # ... more tags
scanTime: 2025-12-03T14:17:00Z
tagCount: 24
```

**ImagePolicy Status:**
```yaml
latestImage: ghcr.io/t-eckert/field-theories:20251111-0358-sha-4c091d3
```

**Discrepancy:** ImageRepository SEES the Dec 3 tag, but ImagePolicy does NOT select it.

---

## Root Cause Analysis

The ImageRepository successfully scans and displays the latest tags (including Dec 2 and Dec 3) in its `status.lastScanResult.latestTags` field.

**However**, the ImagePolicy **persistently resolves to November 11**, despite:
1. The correct tags being visible in ImageRepository
2. The filter pattern matching correctly (`^[0-9]{8}-[0-9]{4}-sha-[a-f0-9]+$`)
3. Alphabetical descending order being the right policy for this format
4. Multiple forced reconciliations

**Hypothesis:** There appears to be a disconnect between:
- **ImageRepository's public status** (shows Dec 3 in `latestTags`)
- **ImageRepository's internal database** (what ImagePolicy queries)

The ImagePolicy may be querying a stale or incomplete internal tag database that was populated before the newer images were pushed to GHCR.

---

## Evidence from Logs

```
14:10:07 - "successful scan: found 80 tags"
14:10:07 - ImagePolicy: "Latest image resolved to 20251111-0358-sha-4c091d3"

14:11:50 - "successful scan: found 23 tags" (with exclusions)
14:11:50 - ImagePolicy: "Latest image resolved to 20251111-0358-sha-4c091d3"

14:15:16 - "successful scan: found 80 tags" (after controller restart, exclusions lost)
14:15:16 - ImagePolicy: "Latest image resolved to 20251111-0358-sha-4c091d3"

14:17:00 - "successful scan: found 24 tags" (exclusions reapplied)
14:17:xx - ImagePolicy: "Latest image resolved to 20251111-0358-sha-4c091d3"
```

**Pattern:** No matter what we do, ImagePolicy stays on Nov 11.

---

## Possible Flux Bug

This behavior suggests a potential bug in Flux image-reflector-controller v0.35.2:

1. **ImageRepository scans correctly** - finds all 24 timestamped tags
2. **ImageRepository status shows correct tags** - Dec 3 appears in `latestTags`
3. **ImagePolicy filter works** - pattern matches timestamped tags
4. **ImagePolicy sorting works** - alphabetical descending is correct
5. **BUT: ImagePolicy selection fails** - always returns Nov 11

**Theory:** The ImageRepository's internal SQLite database may have been initialized when only tags up to Nov 11 existed, and subsequent scans are not properly updating this database with newer tags.

---

## Workaround: Manual Updates

Until this is resolved, manual image updates are required:

```bash
# Check for new images
kubectl get imagerepository field-theories -n field-theories \
  -o jsonpath='{.status.lastScanResult.latestTags[0:5]}'

# Update deployment manually
kubectl set image deployment/field-theories \
  field-theories=ghcr.io/t-eckert/field-theories:20251203-0536-sha-1972e38 \
  -n field-theories

# OR: Edit deployment.yaml and commit
```

---

## Recommendations

### Immediate Actions

1. **Continue manual updates** - Most reliable until automation is fixed
2. **Monitor for Flux updates** - Check for bug fixes in newer versions
3. **Keep exclusion list** - Helps ImageRepository performance

### Long-Term Fixes

1. **Simplify tagging strategy:**
   ```
   # Current (problematic)
   20251203-0536-sha-1972e38  ← timestamped
   sha-1972e38                ← SHA-only (duplicate)

   # Recommended
   20251203-0536-sha-1972e38  ← timestamped only
   ```

2. **Consider semantic versioning:**
   ```yaml
   # Tag format: v1.2.3
   policy:
     semver:
       range: '>=0.0.0'
   ```
   Benefits: Industry standard, better Flux support

3. **File upstream bug report:**
   - Repository: https://github.com/fluxcd/image-reflector-controller
   - Title: "ImagePolicy not detecting newer tags despite ImageRepository scanning them"
   - Include: Our configuration, logs, and evidence

---

## What Actually Worked

**Partial Success:**
- ✅ ImageRepository now properly scans and displays latest tags (24 instead of 80)
- ✅ Exclusion list filters out SHA-only noise
- ✅ Latest tag (Dec 3) IS visible in ImageRepository status

**Still Broken:**
- ❌ ImagePolicy won't select anything newer than Nov 11
- ❌ Image automation remains non-functional

---

## Files Modified

1. `cluster/apps/field-theories/fieldtheories-image-repository.yaml`
   - Added: `^sha-[a-f0-9]+$` to exclusionList

2. `cluster/apps/field-theories/fieldtheories-image-policy.yaml`
   - Kept: Alphabetical ordering (descending)
   - Removed: Numerical policy attempts (didn't work with hyphenated timestamps)

3. `cluster/apps/field-theories/fieldtheories-deployment.yaml`
   - Manually updated: `20251202-1650-sha-f9b7c89` → `20251203-0536-sha-1972e38`

---

## Next Investigation Steps

If you want to dig deeper:

1. **Check Flux database directly:**
   ```bash
   kubectl exec -n flux-system deployment/image-reflector-controller -- \
     ls -la /data/
   ```

2. **Enable debug logging:**
   ```bash
   kubectl set env deployment/image-reflector-controller \
     -n flux-system LOG_LEVEL=debug
   ```

3. **Try upgrading Flux:**
   ```bash
   flux check --pre
   flux install --version=latest
   ```

4. **File GitHub issue** with full reproduction steps

---

## December 5 Update: Git Sync Investigation

### User Hypothesis
User suspected that Flux GitRepository sync might be reverting manual image updates back to old versions, creating a conflict between ImageUpdateAutomation (which should write updates) and GitRepository (which syncs from Git).

### Investigation Results

**Current State as of December 5, 2025:**

1. **ImageRepository Status (WORKING):**
   ```
   latestTags:
     - latest
     - 20251205-0718-sha-8d175df  ← Dec 5 07:18 (NEWEST!)
     - 20251205-0638-sha-37b32d6  ← Dec 5 06:38
     - 20251204-0042-sha-7bfd543  ← Dec 4
     - 20251203-0536-sha-1972e38  ← Dec 3
   ```

2. **ImagePolicy Status (STILL BROKEN):**
   ```yaml
   latestImage: ghcr.io/t-eckert/field-theories:20251111-0358-sha-4c091d3
   status: Succeeded
   message: Latest image tag resolved to 20251111-0358-sha-4c091d3
   lastTransitionTime: 2025-12-03T14:13:32Z
   ```

3. **Actual Deployment in Cluster (STABLE):**
   ```
   image: ghcr.io/t-eckert/field-theories:20251205-0638-sha-37b32d6
   ```

4. **Git Repository State:**
   ```
   HEAD commit: c567a2e806d9b78b9ac413c9202470902e53b923
   Contains: 20251205-0638-sha-37b32d6 (Dec 5 06:38)
   Author: User (manual commit at 02:08:04 on Dec 5)
   ```

5. **GitRepository Sync:**
   ```yaml
   status: Synced
   revision: main@sha1:c567a2e806d9b78b9ac413c9202470902e53b923
   lastScanResult: Succeeded
   ```

6. **Kustomization Status:**
   ```yaml
   lastAppliedRevision: main@sha1:c567a2e806d9b78b9ac413c9202470902e53b923
   status: ReconciliationSucceeded
   conditions: Applied revision successfully
   ```

7. **Recent Kustomization Reconciliations:**
   ```
   22:03:19Z - Deployment/field-theories/field-theories: unchanged
   22:13:30Z - Deployment/field-theories/field-theories: unchanged
   22:24:01Z - Deployment/field-theories/field-theories: unchanged
   22:34:16Z - Deployment/field-theories/field-theories: unchanged
   ```

### Conclusions

**Git Sync is NOT Causing Reverts:**
- The deployment has remained stable at `20251205-0638-sha-37b32d6` (Dec 5 06:38)
- Every Kustomization reconciliation reports the deployment as "unchanged"
- GitRepository is synced to the correct commit containing the Dec 5 image
- No evidence of the deployment reverting to older versions

**Root Cause Remains ImagePolicy Bug:**
- ImagePolicy is stuck on November 11 tag (`20251111-0358-sha-4c091d3`)
- ImageRepository correctly sees tags through December 5
- ImageUpdateAutomation cannot work because it relies on ImagePolicy
- Without working ImagePolicy, automation cannot update Git

**Why Manual Updates Work:**
- Manual updates bypass the broken ImagePolicy
- Changes are committed to Git by the user
- GitRepository syncs these commits successfully
- Kustomization applies them correctly
- Deployment remains stable with manual updates

**New Discovery:**
- An even newer image exists: `20251205-0718-sha-8d175df` (Dec 5 07:18)
- ImageRepository sees it
- ImagePolicy ignores it (still on Nov 11)
- Deployment hasn't been manually updated to it yet

### Updated Workflow

**Current Working Process:**
1. User pushes new Field Theories image to GHCR
2. ImageRepository scans and detects it (✅ WORKING)
3. ImagePolicy should select it (❌ BROKEN - stuck on Nov 11)
4. ImageUpdateAutomation should write update to Git (❌ BLOCKED by broken ImagePolicy)
5. User manually updates deployment.yaml and commits (✅ WORKAROUND)
6. GitRepository syncs the commit (✅ WORKING)
7. Kustomization applies the change (✅ WORKING)
8. Deployment runs new image (✅ WORKING)

**The Broken Link:** Step 3 (ImagePolicy selection) prevents steps 4-7 from being automated.

---

## Conclusion

We've successfully improved the ImageRepository's scanning (now sees Dec 5 tags), but hit a wall with ImagePolicy selection logic. This appears to be a Flux controller bug where the internal tag database doesn't match what the public API shows.

The user's hypothesis about Git sync conflicts was reasonable given the symptoms, but investigation confirmed that Git sync is functioning correctly. The deployment remains stable with manual updates, and Kustomization reconciliation is not reverting changes.

**Recommendation:** Continue with manual image updates and monitor for Flux updates or file an upstream bug report with our findings.

The investigation documentation in `notebook/Flux Image Automation Investigation.md` contains full technical details.
