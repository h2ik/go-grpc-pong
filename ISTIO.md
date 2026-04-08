# Istio Configuration Guide for go-grpc-pong

This guide provides examples for deploying `go-grpc-pong` with Istio service mesh, including Gateway and VirtualService configurations for various scenarios.

## Prerequisites

- Kubernetes cluster with Istio installed
- Istio injection enabled in target namespaces
- `istioctl` CLI (optional, for debugging)
- Container image available at `ghcr.io/h2ik/go-grpc-pong:latest` (or your custom registry)

**Note**: All examples use `ghcr.io/h2ik/go-grpc-pong:latest`. Replace with your registry if using a fork.

## Table of Contents

- [Basic Deployment with Istio](#basic-deployment-with-istio)
- [Ingress Gateway Configuration](#ingress-gateway-configuration)
- [VirtualService Configurations](#virtualservice-configurations)
- [Multi-Cluster Setup](#multi-cluster-setup)
- [Testing Cross-Cluster Connectivity](#testing-cross-cluster-connectivity)
- [Advanced Scenarios](#advanced-scenarios)

## Basic Deployment with Istio

### 1. Create Namespace with Istio Injection

```bash
kubectl create namespace pong-system
kubectl label namespace pong-system istio-injection=enabled
```

### 2. Deploy Pong Server

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pong-server
  namespace: pong-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pong-server
      version: v1
  template:
    metadata:
      labels:
        app: pong-server
        version: v1
    spec:
      containers:
      - name: pong
        image: ghcr.io/h2ik/go-grpc-pong:latest
        args: ["pong", "--addr", ":50051"]
        ports:
        - containerPort: 50051
          name: grpc
          protocol: TCP
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: pong-service
  namespace: pong-system
  labels:
    app: pong-server
spec:
  selector:
    app: pong-server
  ports:
  - port: 50051
    targetPort: 50051
    protocol: TCP
    name: grpc
  type: ClusterIP
```

### 3. Deploy Ping Client

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ping-client
  namespace: pong-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ping-client
      version: v1
  template:
    metadata:
      labels:
        app: ping-client
        version: v1
    spec:
      containers:
      - name: ping
        image: ghcr.io/h2ik/go-grpc-pong:latest
        args:
        - "ping"
        - "--addr"
        - "pong-service.pong-system.svc.cluster.local:50051"
        - "--interval"
        - "2s"
        resources:
          requests:
            cpu: 50m
            memory: 32Mi
          limits:
            cpu: 100m
            memory: 64Mi
```

## Ingress Gateway Configuration

### Expose Pong Service via Istio Ingress Gateway

This allows external clients to reach the pong service through the Istio ingress gateway.

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: pong-gateway
  namespace: pong-system
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: grpc
      protocol: GRPC
    hosts:
    - "pong.example.com"
  - port:
      number: 443
      name: grpc-tls
      protocol: GRPC
    tls:
      mode: SIMPLE
      credentialName: pong-tls-cert
    hosts:
    - "pong.example.com"
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: pong-ingress
  namespace: pong-system
spec:
  hosts:
  - "pong.example.com"
  gateways:
  - pong-gateway
  http:
  - match:
    - uri:
        prefix: "/pong.PongService"
    route:
    - destination:
        host: pong-service
        port:
          number: 50051
```

### Create TLS Certificate (for HTTPS/port 443)

To use port 443 with TLS, create a Kubernetes secret with your certificate:

```bash
# Create TLS secret with your certificate and key
kubectl create -n istio-system secret tls pong-tls-cert \
  --key=tls.key \
  --cert=tls.crt

# Or use cert-manager to automatically provision certificates
# (see Istio documentation for cert-manager integration)
```

### Test External Access

```bash
# Get the ingress gateway address
export INGRESS_HOST=$(kubectl -n istio-system get service istio-ingressgateway \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test with grpcurl (HTTP/port 80 - plaintext)
grpcurl -plaintext -authority pong.example.com \
  -d '{"message": "ping", "timestamp": 1234567890}' \
  $INGRESS_HOST:80 pong.PongService/Ping

# Test with grpcurl (HTTPS/port 443 - requires TLS certificate)
grpcurl -authority pong.example.com \
  -d '{"message": "ping", "timestamp": 1234567890}' \
  $INGRESS_HOST:443 pong.PongService/Ping

# Or use the ping client (port 80)
./go-grpc-pong ping --addr $INGRESS_HOST:80
```

## VirtualService Configurations

### Traffic Splitting (Canary Deployment)

Split traffic between two versions of the pong service:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: pong-service
  namespace: pong-system
spec:
  hosts:
  - pong-service
  http:
  - route:
    - destination:
        host: pong-service
        subset: v1
      weight: 90
    - destination:
        host: pong-service
        subset: v2
      weight: 10
---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: pong-service
  namespace: pong-system
spec:
  host: pong-service
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
```

### Header-Based Routing

Route based on request headers (useful for testing specific versions):

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: pong-service
  namespace: pong-system
spec:
  hosts:
  - pong-service
  http:
  - match:
    - headers:
        x-version:
          exact: "v2"
    route:
    - destination:
        host: pong-service
        subset: v2
  - route:
    - destination:
        host: pong-service
        subset: v1
```

### Timeout and Retry Configuration

Add resilience with timeouts and retries:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: pong-service
  namespace: pong-system
spec:
  hosts:
  - pong-service
  http:
  - route:
    - destination:
        host: pong-service
    timeout: 5s
    retries:
      attempts: 3
      perTryTimeout: 2s
      retryOn: "5xx,reset,connect-failure,refused-stream"
```

### Fault Injection (Testing)

Inject delays or failures to test resilience:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: pong-service
  namespace: pong-system
spec:
  hosts:
  - pong-service
  http:
  - fault:
      delay:
        percentage:
          value: 10.0
        fixedDelay: 500ms
      abort:
        percentage:
          value: 5.0
        httpStatus: 503
    route:
    - destination:
        host: pong-service
```

## Multi-Cluster Setup

### Scenario: Cross-Cluster Communication

Deploy pong server in **Cluster A** and ping client in **Cluster B**.

#### Cluster A: Deploy Pong Server with ServiceEntry

```yaml
# On Cluster A
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pong-server
  namespace: pong-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: pong-server
  template:
    metadata:
      labels:
        app: pong-server
    spec:
      containers:
      - name: pong
        image: ghcr.io/h2ik/go-grpc-pong:latest
        args: ["pong", "--addr", ":50051"]
        ports:
        - containerPort: 50051
          name: grpc
---
apiVersion: v1
kind: Service
metadata:
  name: pong-service
  namespace: pong-system
  labels:
    app: pong-server
spec:
  selector:
    app: pong-server
  ports:
  - port: 50051
    targetPort: 50051
    name: grpc
```

#### Cluster A: Expose via East-West Gateway

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: cross-cluster-gateway
  namespace: istio-system
spec:
  selector:
    istio: eastwestgateway
  servers:
  - port:
      number: 15443
      name: tls
      protocol: TLS
    tls:
      mode: AUTO_PASSTHROUGH
    hosts:
    - "*.pong-system.svc.cluster.local"
```

#### Cluster B: ServiceEntry for Remote Pong Service

```yaml
# On Cluster B
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: pong-service-remote
  namespace: pong-system
spec:
  hosts:
  - pong-service.pong-system.svc.cluster.local
  location: MESH_INTERNAL
  ports:
  - number: 50051
    name: grpc
    protocol: GRPC
  resolution: DNS
  addresses:
  - 240.0.0.1  # VIP for remote service
  endpoints:
  - address: <CLUSTER_A_EASTWEST_GATEWAY_IP>
    ports:
      grpc: 15443
    labels:
      cluster: cluster-a
```

#### Cluster B: Deploy Ping Client

```yaml
# On Cluster B
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ping-client
  namespace: pong-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ping-client
  template:
    metadata:
      labels:
        app: ping-client
    spec:
      containers:
      - name: ping
        image: ghcr.io/h2ik/go-grpc-pong:latest
        args:
        - "ping"
        - "--addr"
        - "pong-service.pong-system.svc.cluster.local:50051"
        - "--interval"
        - "2s"
```

## Testing Cross-Cluster Connectivity

### Verify Istio Configuration

```bash
# Check if services are recognized
istioctl proxy-status

# Verify sidecar configuration for ping client
istioctl proxy-config endpoint <ping-client-pod> --namespace pong-system

# Check if remote service is reachable
kubectl exec -it <ping-client-pod> -n pong-system -c ping -- /bin/sh
# Inside the pod:
# nslookup pong-service.pong-system.svc.cluster.local
```

### Monitor Traffic

```bash
# Watch ping client logs
kubectl logs -f <ping-client-pod> -n pong-system -c ping

# Watch pong server logs
kubectl logs -f <pong-server-pod> -n pong-system -c pong

# View Istio proxy logs
kubectl logs -f <ping-client-pod> -n pong-system -c istio-proxy
```

### Check Metrics in Kiali

```bash
# Port-forward to Kiali (if installed)
istioctl dashboard kiali

# Navigate to Graph view
# Select namespace: pong-system
# Look for traffic flow: ping-client -> pong-service
```

## Advanced Scenarios

### mTLS Policy

Enforce mutual TLS between ping and pong services:

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: pong-mtls
  namespace: pong-system
spec:
  selector:
    matchLabels:
      app: pong-server
  mtls:
    mode: STRICT
```

### Authorization Policy

Restrict access to pong service:

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: pong-authz
  namespace: pong-system
spec:
  selector:
    matchLabels:
      app: pong-server
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/pong-system/sa/ping-client"]
    to:
    - operation:
        methods: ["POST"]
        paths: ["/pong.PongService/Ping"]
```

### Circuit Breaker

Protect pong service from overload:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: pong-circuit-breaker
  namespace: pong-system
spec:
  host: pong-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        http2MaxRequests: 100
        maxRequestsPerConnection: 2
    outlierDetection:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
      minHealthPercent: 40
```

### Egress Gateway (For External Pong Service)

Route ping client traffic through egress gateway:

```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: pong-egress-gateway
  namespace: istio-system
spec:
  selector:
    istio: egressgateway
  servers:
  - port:
      number: 50051
      name: grpc
      protocol: GRPC
    hosts:
    - external-pong.example.com
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: pong-external
  namespace: pong-system
spec:
  hosts:
  - external-pong.example.com
  gateways:
  - mesh
  - istio-system/pong-egress-gateway
  http:
  - match:
    - gateways:
      - mesh
      port: 50051
    route:
    - destination:
        host: istio-egressgateway.istio-system.svc.cluster.local
        port:
          number: 50051
  - match:
    - gateways:
      - istio-system/pong-egress-gateway
      port: 50051
    route:
    - destination:
        host: external-pong.example.com
        port:
          number: 50051
```

## Troubleshooting

### Common Issues

1. **Sidecar not injected**: Verify namespace has `istio-injection=enabled` label
   ```bash
   kubectl get namespace pong-system --show-labels
   ```

2. **Connection refused**: Check if services are properly registered
   ```bash
   istioctl proxy-config endpoints <pod-name> -n pong-system
   ```

3. **mTLS errors**: Verify PeerAuthentication policies
   ```bash
   istioctl authn tls-check <pod-name>.pong-system pong-service.pong-system.svc.cluster.local
   ```

4. **Cross-cluster connectivity fails**: Ensure east-west gateway is accessible
   ```bash
   kubectl get svc -n istio-system istio-eastwestgateway
   ```

### Debug Commands

```bash
# Analyze Istio configuration
istioctl analyze -n pong-system

# Check proxy configuration sync status
istioctl proxy-status

# View complete Envoy configuration
istioctl proxy-config all <pod-name> -n pong-system -o json

# Test connectivity from ping pod
kubectl exec -it <ping-pod> -n pong-system -c ping -- \
  /app/go-grpc-pong ping --addr pong-service:50051 --interval 1s
```

## Performance Tuning

### Optimize for Low Latency

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: pong-performance
  namespace: pong-system
spec:
  host: pong-service
  trafficPolicy:
    connectionPool:
      http:
        h2UpgradePolicy: UPGRADE
        useClientProtocol: true
    loadBalancer:
      simple: LEAST_REQUEST
      localityLbSetting:
        enabled: true
        distribute:
        - from: us-west-1/*
          to:
            "us-west-1/*": 80
            "us-east-1/*": 20
```

## References

- [Istio Documentation](https://istio.io/latest/docs/)
- [Istio Traffic Management](https://istio.io/latest/docs/concepts/traffic-management/)
- [Istio Multi-Cluster](https://istio.io/latest/docs/setup/install/multicluster/)
- [Istio Security](https://istio.io/latest/docs/concepts/security/)
