# Architecture

## Overview

```
┌─────────────────────────────────────────────────┐
│                   Browser                        │
│  React + TypeScript + Tailwind + Zustand         │
│  WebSocket ←──────────────────────── REST API    │
└──────┬──────────────────────────────────┬────────┘
       │ ws://                            │ http://
┌──────▼──────────────────────────────────▼────────┐
│              Go Binary (single process)           │
│                                                   │
│  ┌─────────┐  ┌──────────┐  ┌─────────────────┐  │
│  │ WS Hub  │  │ REST API │  │ Embedded SPA    │  │
│  │ + Watch │  │ Handlers │  │ (go:embed)      │  │
│  └────┬────┘  └────┬─────┘  └─────────────────┘  │
│       │             │                             │
│  ┌────▼─────────────▼─────┐  ┌─────────────────┐ │
│  │   Hub Manager          │  │  Ollama Client  │ │
│  │   (K8s dynamic client) │  │  (SSE streaming)│ │
│  └────────────┬───────────┘  └────────┬────────┘ │
└───────────────┼───────────────────────┼──────────┘
                │                       │
        ┌───────▼───────┐       ┌───────▼───────┐
        │  Hub Cluster  │       │    Ollama     │
        │  (ACM + ZTP)  │       │  (localhost)  │
        └───────────────┘       └───────────────┘
```

## Watched Resources

The dashboard watches these Kubernetes resources via the dynamic client:

| Resource | API Group | Scope |
|----------|-----------|-------|
| ManagedCluster | `cluster.open-cluster-management.io/v1` | Cluster |
| ClusterDeployment | `hive.openshift.io/v1` | Namespaced |
| ClusterInstance | `siteconfig.open-cluster-management.io/v1alpha1` | Namespaced |
| InfraEnv | `agent-install.openshift.io/v1beta1` | Namespaced |
| BareMetalHost | `metal3.io/v1alpha1` | Namespaced |
| Agent | `agent-install.openshift.io/v1beta1` | Namespaced |
| AgentClusterInstall | `extensions.hive.openshift.io/v1beta1` | Namespaced |
| Policy | `policy.open-cluster-management.io/v1` | Namespaced |
| Application | `argoproj.io/v1alpha1` | Namespaced |

## Event Classification

Every watch event is classified server-side with:

- **Severity** — `good`, `bad`, `warning`, `info`, or `neutral` based on resource state analysis (conditions, compliance, power state, ArgoCD health)
- **Insight** — A one-line contextual description like "Policy violations detected — remediation needed" or "Cluster is healthy and reporting"

## AI Diagnostics

The AI integration uses domain-specific prompt templates for:

- **Provisioning errors** — ClusterDeployment/AgentClusterInstall failure conditions
- **Cluster health** — Degraded operators, NotReady nodes
- **Policy compliance** — Non-compliant policies with remediation guidance
- **BMC errors** — BareMetalHost conditions and hardware issues
- **General diagnostics** — Any ZTP resource context

Responses are streamed via SSE (Server-Sent Events) for real-time display with a typing animation.
