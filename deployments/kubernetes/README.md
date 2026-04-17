# Telecom Platform Kubernetes Deployment

This directory contains the complete Kubernetes deployment configuration for the Telecom Platform, including all applications and monitoring stack.

## Architecture Overview

```
telecom-platform Namespace
  api-server (3 replicas)          - Main GraphQL API server
  charging-engine (2 replicas)     - Rust charging engine
  packet-gateway (2 replicas)      - eBPF packet gateway
  web-dashboard (2 replicas)       - Next.js admin dashboard
  postgres (1 replica)              - PostgreSQL database
  redis (1 replica)                 - Redis cache
  postgres-exporter (1 replica)     - PostgreSQL metrics
  redis-exporter (1 replica)        - Redis metrics

telecom-monitoring Namespace
  prometheus (1 replica)           - Metrics collection
  grafana (1 replica)              - Visualization dashboards
  influxdb (1 replica)             - Time-series storage
  alertmanager (1 replica)         - Alert management
  node-exporter (1 replica)        - System metrics
```

## Prerequisites

### Required Tools
- **kubectl** - Kubernetes CLI
- **Docker** - Container runtime
- **Helm** (optional) - For advanced deployments

### Kubernetes Requirements
- **Kubernetes 1.25+** - For modern API support
- **Ingress Controller** - nginx-ingress recommended
- **Persistent Storage** - 50GB+ total for databases and monitoring
- **LoadBalancer** - For external service access (or NodePort)

### Resource Requirements
- **CPU**: 4+ cores minimum
- **Memory**: 8GB+ minimum
- **Storage**: 50GB+ persistent storage

## Quick Start

### 1. Deploy Everything
```bash
cd deployments/kubernetes
./deploy.sh
```

### 2. Access Services
After deployment, access services via:
- **Web Dashboard**: http://localhost:3000
- **API Server**: http://localhost:8000
- **Grafana**: http://localhost:3001 (admin/admin123)
- **Prometheus**: http://localhost:9090

### 3. Port Forwarding (if needed)
```bash
# Applications
kubectl port-forward svc/api-server-service 8000:8000 -n telecom-platform
kubectl port-forward svc/web-dashboard-service 3000:3000 -n telecom-platform

# Monitoring
kubectl port-forward svc/grafana 3001:3000 -n telecom-monitoring
kubectl port-forward svc/prometheus 9090:9090 -n telecom-monitoring
```

## Deployment Components

### Namespaces
- **telecom-platform** - Main application services
- **telecom-monitoring** - Monitoring and observability

### Applications

#### API Server
- **Replicas**: 3 (auto-scalable to 10)
- **Ports**: 8000 (HTTP), 9090 (Metrics)
- **Resources**: 256Mi-512Mi memory, 250m-500m CPU
- **Features**: GraphQL API, REST endpoints, metrics export

#### Charging Engine
- **Replicas**: 2 (auto-scalable to 6)
- **Port**: 3001 (HTTP + Metrics)
- **Resources**: 128Mi-256Mi memory, 100m-200m CPU
- **Features**: Real-time credit control, usage tracking

#### Packet Gateway
- **Replicas**: 2 (auto-scalable to 4)
- **Port**: 9000 (Metrics)
- **Resources**: 256Mi-512Mi memory, 250m-500m CPU
- **Features**: eBPF packet processing, network isolation
- **Privileges**: NET_ADMIN, SYS_ADMIN for eBPF

#### Web Dashboard
- **Replicas**: 2 (auto-scalable to 6)
- **Port**: 3000 (HTTP)
- **Resources**: 128Mi-256Mi memory, 100m-200m CPU
- **Features**: React admin interface, real-time updates

### Databases

#### PostgreSQL
- **Version**: 15
- **Storage**: 10Gi persistent volume
- **Features**: Primary database, metrics export
- **Exporter**: postgres-exporter on port 9187

#### Redis
- **Version**: 7-alpine
- **Storage**: 256Mi memory limit
- **Features**: Caching, session storage, real-time data
- **Exporter**: redis-exporter on port 9121

### Monitoring Stack

#### Prometheus
- **Version**: v2.45.0
- **Storage**: 20Gi persistent volume
- **Port**: 9090
- **Features**: Metrics collection, alerting rules
- **Targets**: Auto-discovery of all services

#### Grafana
- **Version**: 10.2.0
- **Storage**: 10Gi persistent volume
- **Port**: 3000
- **Features**: Dashboards, alerting, user management
- **Data Sources**: Prometheus, InfluxDB

#### InfluxDB
- **Version**: 2.7
- **Storage**: 20Gi persistent volume
- **Port**: 8086
- **Features**: Time-series data, long-term storage
- **Organization**: telecom, Bucket: telecom

#### Alertmanager
- **Version**: v0.25.0
- **Storage**: 5Gi persistent volume
- **Port**: 9093
- **Features**: Alert routing, notifications, silencing

## Configuration

### Environment Variables
Key configuration in `configmaps.yaml`:
- Database connections
- Redis configuration
- Monitoring endpoints
- Service URLs

### Secrets
Sensitive data in `secrets.yaml`:
- Database passwords
- API keys
- Authentication tokens
- TLS certificates

### Customization
To customize the deployment:

1. **Modify ConfigMaps**: Update `configmaps.yaml`
2. **Update Secrets**: Modify `secrets.yaml` with real values
3. **Adjust Resources**: Update resource limits in deployment files
4. **Change Replicas**: Modify replica counts
5. **Update Ingress**: Configure hostnames and TLS

## Auto-scaling

### Horizontal Pod Autoscalers (HPA)
All applications have HPA configured:
- **API Server**: 3-10 replicas, 70% CPU, 80% memory
- **Charging Engine**: 2-6 replicas, 70% CPU, 80% memory
- **Packet Gateway**: 2-4 replicas, 75% CPU, 85% memory
- **Web Dashboard**: 2-6 replicas, 70% CPU, 80% memory

### Scaling Behavior
- **Scale Up**: 50% increase, 60s stabilization
- **Scale Down**: 10% decrease, 300s stabilization

## Ingress Configuration

### Hostnames
Default ingress configuration:
- **API**: api.telecom.example.com
- **Dashboard**: dashboard.telecom.example.com
- **Grafana**: grafana.telecom.example.com
- **Prometheus**: prometheus.telecom.example.com

### TLS
- Automatic TLS with cert-manager
- Custom certificates supported
- HTTP to HTTPS redirection

### Rate Limiting
- 100 requests per minute per IP
- Configurable per service

## Monitoring and Alerting

### Prometheus Metrics
All services export metrics on dedicated endpoints:
- HTTP request metrics
- Business metrics (subscribers, usage)
- System metrics (CPU, memory)
- Custom application metrics

### Grafana Dashboards
Pre-configured dashboards:
- **Telecom Platform Overview** - System-wide metrics
- **Subscriber Metrics** - Detailed subscriber analytics
- **Infrastructure** - Resource utilization
- **Applications** - Service-specific metrics

### Alerting Rules
Critical alerts configured:
- High error rates (>10%)
- Service down events
- Resource exhaustion
- Database connection failures

### Alert Routing
- **Critical**: Email + webhook notifications
- **Warning**: Webhook notifications only
- **Silencing**: Configurable maintenance windows

## Storage

### Persistent Volumes
- **PostgreSQL**: 10Gi, ReadWriteOnce
- **Redis**: emptyDir (in-memory)
- **Prometheus**: 20Gi, ReadWriteOnce
- **Grafana**: 10Gi, ReadWriteOnce
- **InfluxDB**: 20Gi, ReadWriteOnce
- **Alertmanager**: 5Gi, ReadWriteOnce

### Storage Classes
Uses default storage class. Customize for your environment:
- AWS: gp2/gp3
- GCP: standard-ssd
- Azure: Premium SSD
- On-prem: Local SSD/NVMe

## Security

### Network Policies
Default policies allow:
- Intra-namespace communication
- Cross-namespace for monitoring
- External access via Ingress only

### RBAC
Service accounts with minimal permissions:
- Database access only for applications
- Monitoring access for Prometheus
- ConfigMap/Secret access as needed

### Secrets Management
- Kubernetes native secrets
- Encrypted at rest
- Environment variable injection
- Rotation policies recommended

## Troubleshooting

### Common Issues

#### Pods Not Starting
```bash
# Check pod status
kubectl get pods -n telecom-platform -o wide

# Check pod logs
kubectl logs -f deployment/api-server -n telecom-platform

# Describe pod for detailed info
kubectl describe pod <pod-name> -n telecom-platform
```

#### Service Not Accessible
```bash
# Check service endpoints
kubectl get endpoints -n telecom-platform

# Test service connectivity
kubectl exec -it <pod-name> -n telecom-platform -- curl http://api-server-service:8000/health
```

#### Monitoring Issues
```bash
# Check Prometheus targets
kubectl port-forward svc/prometheus 9090:9090 -n telecom-monitoring
# Visit http://localhost:9090/targets

# Check Grafana data sources
kubectl port-forward svc/grafana 3001:3000 -n telecom-monitoring
# Visit http://localhost:3001/datasources
```

#### Storage Issues
```bash
# Check PVC status
kubectl get pvc -n telecom-platform
kubectl get pvc -n telecom-monitoring

# Check storage class
kubectl get storageclass
```

### Debug Commands
```bash
# Full cluster status
kubectl get all -n telecom-platform
kubectl get all -n telecom-monitoring

# Resource usage
kubectl top pods -n telecom-platform
kubectl top nodes

# Events
kubectl get events -n telecom-platform --sort-by='.lastTimestamp'
kubectl get events -n telecom-monitoring --sort-by='.lastTimestamp'
```

## Maintenance

### Updates
```bash
# Update images
kubectl set image deployment/api-server api-server=telecom/api-server:v2.0.0 -n telecom-platform

# Rolling restart
kubectl rollout restart deployment/api-server -n telecom-platform

# Check rollout status
kubectl rollout status deployment/api-server -n telecom-platform
```

### Backup
```bash
# Backup databases
kubectl exec -it postgres-0 -n telecom-platform -- pg_dump telecom_platform > backup.sql

# Backup configurations
kubectl get all -n telecom-platform -o yaml > platform-backup.yaml
kubectl get all -n telecom-monitoring -o yaml > monitoring-backup.yaml
```

### Cleanup
```bash
# Remove everything
./deploy.sh cleanup

# Remove specific components
kubectl delete -f manifests/applications.yaml
kubectl delete -f manifests/monitoring.yaml
```

## Production Considerations

### High Availability
- Use multiple replicas for stateless services
- Configure database replication
- Implement backup and disaster recovery
- Use anti-affinity rules for pod distribution

### Performance
- Enable resource limits and requests
- Use appropriate storage classes
- Configure network policies
- Monitor and optimize based on metrics

### Security
- Enable Pod Security Policies
- Use network segmentation
- Implement secret rotation
- Regular security scanning

### Compliance
- Enable audit logging
- Implement data retention policies
- Use encrypted storage
- Regular compliance checks

## Next Steps

1. **Customize Configuration**: Update for your environment
2. **Set Up DNS**: Configure hostnames in your DNS
3. **Configure TLS**: Update certificates for production
4. **Set Up Monitoring**: Configure alert recipients
5. **Implement Backup**: Set up automated backups
6. **Performance Tuning**: Optimize based on actual usage

## Support

For deployment issues:
1. Check the troubleshooting section
2. Review pod logs and events
3. Verify resource availability
4. Test with smaller deployments first

For application issues:
1. Check the application documentation
2. Review service health endpoints
3. Verify database connectivity
4. Check monitoring dashboards
