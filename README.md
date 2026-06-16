A distributed log aggregation and streaming platform.

3 layers:
Ingestion -> Redis Stream -> Egress

Layer 1 (Ingestion):
A node-level log agent that mounts container runtime log directories, dynamically discovers and tails log files
(like /var/log/pods for Kubernetes or /var/lib/docker/containers for Docker), tracks read offsets for reliability,
enriches logs with container metadata, and streams them to Redis.
