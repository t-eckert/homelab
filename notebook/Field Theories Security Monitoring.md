# Field Theories Security Monitoring

## Overview

This document outlines recommended monitoring and alerting for the Field Theories Cloudflare Tunnel deployment to detect potential security incidents.

## Implementation Date

December 2, 2025

## Security Improvements Implemented

1. ✅ Network policies isolating Field Theories pods from other cluster services
2. ✅ Security contexts enforcing non-root execution
3. ✅ Pinned container image versions (cloudflared:2025.11.1)
4. ✅ Separated BlueSky poster into isolated namespace
5. ✅ Downgraded namespace pod security from privileged to baseline

## Recommended Monitoring

### 1. Network Policy Violations

**What to Monitor:**
- Blocked connections from field-theories pods to other namespaces
- Unexpected ingress attempts to field-theories pods

**Implementation:**
```bash
# Check network policy counters (if using Calico or Cilium)
kubectl get networkpolicies -n field-theories
```

**Alert Threshold:** Any blocked connection attempts

---

### 2. Pod Security Context Changes

**What to Monitor:**
- Pods running as root (uid=0)
- Privilege escalation attempts
- New capabilities added to containers

**Implementation:**
```bash
# Audit running pods
kubectl get pods -n field-theories -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.securityContext.runAsUser}{"\n"}{end}'
```

**Alert Threshold:** Any pod running with uid != expected values (1000 for field-theories, 65532 for cloudflared)

---

### 3. Cloudflared Tunnel Status

**What to Monitor:**
- Tunnel connection failures
- Certificate/credential errors
- Unusual connection churn

**Implementation:**
```bash
# Check tunnel connections
kubectl logs -n field-theories deployment/cloudflared --tail=100 | grep -E "Registered|error|Error"
```

**Alert Threshold:**
- More than 5 connection failures in 10 minutes
- Any "credential" or "authentication" errors

---

### 4. Unexpected Network Egress

**What to Monitor:**
- Field Theories pods connecting to external IPs (should only need DNS)
- Cloudflared connecting to non-Cloudflare IPs

**Implementation:**
```bash
# Review network connections (requires network monitoring tool)
# Example with Cilium Hubble:
hubble observe --namespace field-theories --verdict DROPPED
```

**Alert Threshold:** Any dropped egress traffic from field-theories pods

---

### 5. Resource Consumption Anomalies

**What to Monitor:**
- CPU/memory spikes indicating cryptomining or DDoS abuse
- Unusual request patterns to Field Theories

**Current Limits:**
```yaml
field-theories:
  memory: 128Mi
  cpu: 100m

cloudflared:
  # No limits set (should add)
```

**Implementation:**
```bash
# Check resource usage
kubectl top pods -n field-theories
```

**Alert Threshold:**
- Field Theories consistently hitting memory/CPU limits
- Sustained high CPU usage (>80% for >5 minutes)

---

### 6. Image Version Changes

**What to Monitor:**
- Unexpected image tag changes
- Use of :latest tag being introduced

**Implementation:**
```bash
# Audit current images
kubectl get pods -n field-theories -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].image}{"\n"}{end}'
```

**Expected Values:**
- `cloudflared:2025.11.1` (pinned)
- `field-theories:20251128-2006-sha-fe3d065` (Flux-managed)

**Alert Threshold:** Any deviation from pinned versions

---

### 7. Secret Access Auditing

**What to Monitor:**
- Access to cloudflared-credentials secret
- Cross-namespace secret access attempts

**Implementation:**
```bash
# Enable Kubernetes audit logging and monitor for secret access
# Check audit logs for events like:
# - get secrets/cloudflared-credentials
# - list secrets in field-theories namespace
```

**Alert Threshold:** Any secret access from unexpected service accounts

---

### 8. Cloudflared Version Monitoring

**What to Monitor:**
- New Cloudflare security advisories
- Available updates to cloudflared

**Implementation:**
- Subscribe to: https://github.com/cloudflare/cloudflared/releases
- Check monthly: `curl -s https://api.github.com/repos/cloudflare/cloudflared/releases/latest`

**Update Schedule:** Review and update quarterly, or immediately for security fixes

---

## Quick Security Check Commands

```bash
# 1. Verify network policies are active
kubectl get networkpolicies -n field-theories

# 2. Check all pods are running with security contexts
kubectl get pods -n field-theories -o json | jq '.items[] | {name: .metadata.name, user: .spec.securityContext.runAsUser}'

# 3. Verify Cloudflared tunnel health
kubectl logs -n field-theories deployment/cloudflared --tail=20 | grep Registered

# 4. Check for failed pods
kubectl get pods -n field-theories --field-selector=status.phase!=Running

# 5. Verify public site accessibility
curl -I https://fieldtheories.blog

# 6. Check resource usage
kubectl top pods -n field-theories
```

---

## Incident Response Procedures

### If Field Theories Pod is Compromised:

1. **Isolate:**
   ```bash
   kubectl scale deployment field-theories -n field-theories --replicas=0
   ```

2. **Investigate:**
   ```bash
   kubectl logs -n field-theories deployment/field-theories --previous
   ```

3. **Check lateral movement attempts:**
   ```bash
   # Review blocked connections in network policy logs
   kubectl describe networkpolicy field-theories-isolation -n field-theories
   ```

4. **Rebuild:**
   ```bash
   # Force new deployment with fresh image
   kubectl rollout restart deployment/field-theories -n field-theories
   ```

### If Cloudflared Credentials are Compromised:

1. **Revoke tunnel credentials in Cloudflare dashboard**
2. **Generate new tunnel token**
3. **Update secret:**
   ```bash
   kubectl delete secret cloudflared-credentials -n field-theories
   kubectl create secret generic cloudflared-credentials \
     --from-file=credentials.json=./new-credentials.json \
     -n field-theories
   ```
4. **Restart cloudflared:**
   ```bash
   kubectl rollout restart deployment/cloudflared -n field-theories
   ```

---

## Next Steps

1. **Implement log aggregation** (Loki is already deployed in monitoring namespace)
2. **Set up Prometheus alerts** for the metrics above
3. **Enable Kubernetes audit logging** for secret access monitoring
4. **Consider adding WAF rules** in Cloudflare dashboard
5. **Implement rate limiting** in Cloudflare for DDoS protection
6. **Schedule quarterly security reviews** of Field Theories deployment

---

## References

- [Security Threat Analysis](./Field Theories Security Analysis.md) (if created)
- [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Cloudflare Tunnel Security](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/get-started/tunnel-permissions/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
