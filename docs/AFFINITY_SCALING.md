# Node Affinity/Selector Scaling

## Overview

Kproximate can now detect and respond to pods that are unschedulable due to node affinity or node selector mismatches. This feature is **disabled by default** and must be explicitly enabled.

## How It Works

When enabled, kproximate:
1. Monitors for pods with scheduling failures containing:
   - `didn't match Pod's node affinity/selector`
   - `didn't match node selector`
2. **Extracts required labels** from the pod's `nodeSelector` and `affinity.nodeAffinity` specifications
3. **Compares** extracted labels against configured `kpNodeLabels`
4. **Provisions one new node** only if at least one label matches

## Configuration

### Enable the Feature

Set `enableAffinityScaling: true` in your Helm values:

```yaml
kproximate:
  config:
    enableAffinityScaling: true
    kpNodeLabels: "workload-type=general,zone={{ .TargetHost }}"
```

Or via environment variable:
```bash
export enableAffinityScaling=true
```

### Configure Node Labels

**Critical**: You must configure `kpNodeLabels` to include at least one label that matches your pod's affinity/selector requirements.

Example pod with node selector:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  nodeSelector:
    workload-type: general
  containers:
  - name: app
    image: nginx
```

Matching kproximate configuration:
```yaml
kproximate:
  config:
    enableAffinityScaling: true
    kpNodeLabels: "workload-type=general"
```

### Dynamic Labels

Use Go templating for dynamic values:

```yaml
kpNodeLabels: "topology.kubernetes.io/zone={{ .TargetHost }},workload-type=general"
```

Available template variables:
- `{{ .TargetHost }}` - The Proxmox host name where the node is provisioned

**Note**: Templated labels (containing `{{ }}`) are skipped during matching validation.

## Behavior

### Scaling Trigger
- Only triggers when **no resource-based scaling** is needed (no CPU/memory shortage)
- Only triggers when **no scaling events are in progress**
- **Validates label match** before provisioning
- Provisions **exactly one node** per detection

### Label Matching Logic

Kproximate extracts labels from:
- `spec.nodeSelector` - all key-value pairs
- `spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution` - only `In` operator with values

If **at least one** configured label matches a required label, scaling proceeds.

### Limitations

1. **Partial matching**: Only checks if at least one label matches, not all required labels
2. **First value only**: For affinity with multiple values, only the first value is extracted
3. **Templated labels ignored**: Labels with `{{ }}` are not validated during matching
4. **Single node only**: Only provisions one node per poll cycle

## Example Scenarios

### Scenario 1: Node Selector Match

**Pod:**
```yaml
nodeSelector:
  gpu: "true"
```

**Configuration:**
```yaml
kpNodeLabels: "gpu=true"
enableAffinityScaling: true
```

**Result:** ✅ Label matches → Node provisioned with `gpu=true` label → Pod schedules

### Scenario 2: Node Affinity with Custom Domain

**Pod:**
```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: api.hostoflow.internal/dedicated
              operator: In
              values:
                - tenant-deployment
```

**Configuration:**
```yaml
kpNodeLabels: "api.hostoflow.internal/dedicated=tenant-deployment"
enableAffinityScaling: true
```

**Result:** ✅ Label matches → Node provisioned → Pod schedules

### Scenario 3: Multiple Labels (Partial Match)

**Pod:**
```yaml
nodeSelector:
  disk-type: ssd
  workload-type: database
```

**Configuration:**
```yaml
kpNodeLabels: "workload-type=database,zone={{ .TargetHost }}"
enableAffinityScaling: true
```

**Result:** ✅ One label matches (`workload-type`) → Node provisioned with both labels → Pod schedules

### Scenario 4: No Match

**Pod:**
```yaml
nodeSelector:
  disk-type: ssd
```

**Configuration:**
```yaml
kpNodeLabels: "workload-type=general"
enableAffinityScaling: true
```

**Result:** ❌ No matching labels → Warning logged → No scaling → Pod remains unschedulable

### Scenario 5: Zone Affinity (Templated)

**Pod:**
```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: topology.kubernetes.io/zone
          operator: In
          values:
          - pve-host-1
```

**Configuration:**
```yaml
kpNodeLabels: "topology.kubernetes.io/zone={{ .TargetHost }}"
enableAffinityScaling: true
```

**Result:** ⚠️ Templated label skipped during validation → No scaling (unless other labels match)

**Workaround**: Add a static label that matches:
```yaml
kpNodeLabels: "topology.kubernetes.io/zone={{ .TargetHost }},workload-type=general"
# And add workload-type to pod selector
```

## Monitoring

Check logs for affinity-related scaling:
```bash
kubectl logs -n kproximate deployment/kproximate-controller | grep affinity
```

Expected log messages:
```
Pod (my-app) unschedulable due to node affinity/selector mismatch
Triggering scale up due to node affinity/selector mismatch with matching labels
```

Warning when labels don't match:
```
Pod affinity/selector mismatch detected but configured kpNodeLabels do not match required labels required=map[disk-type:ssd]
```

## Recommendations

1. **Start disabled**: Test your `kpNodeLabels` configuration first
2. **Include static labels**: Ensure at least one non-templated label matches pod requirements
3. **Use specific labels**: Avoid generic labels that might match unintended pods
4. **Monitor logs**: Watch for "do not match required labels" warnings
5. **Combine with resource scaling**: This feature complements, not replaces, resource-based scaling

## Disabling

Set to `false` or omit from configuration:
```yaml
kproximate:
  config:
    enableAffinityScaling: false  # or omit entirely
```
