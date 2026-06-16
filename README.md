## Parsel DLASS

### Proposed System Architecture:

<img width="1657" height="742" alt="image" src="https://github.com/user-attachments/assets/f1bf992e-7f36-4edb-8694-d12c40c080ef" />

3 layers:
Ingestion -> Redis Stream -> Egress

Layer 1 (Ingestion):
A node-level log agent that mounts container runtime log directories, dynamically discovers and tails log files
(like /var/log/pods for Kubernetes or /var/lib/docker/containers for Docker), tracks read offsets for reliability,
enriches logs with container metadata, and streams them to Redis.
