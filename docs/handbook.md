# NicheCP Engineering Handbook

Welcome to the NicheCP Engineering Handbook. This repository contains the complete, production-grade documentation for the architecture, deployment, and operational procedures of the NicheCP competitive programming platform.

Whether you are a new contributor attempting to understand the Execution Engine, or a DevOps engineer tasked with deploying the platform to Oracle Cloud, this handbook is your definitive guide.

## Documentation Index

### 1. Product & Vision
* **[00 - Executive Summary](./00-executive-summary.md)**: Vision, features, tech stack, and roadmap.
* **[01 - Product Requirements](./01-product-requirements.md)**: Target audience, functional requirements, and hardware constraints.

### 2. Architecture & Design
* **[02 - System Architecture](./02-system-architecture.md)**: High-level component interactions, request lifecycles, and sequence diagrams.
* **[03 - Backend Architecture](./03-backend-architecture.md)**: Go packages, Gin routing, and RBAC middleware.
* **[04 - Frontend Architecture](./04-frontend-architecture.md)**: Vanilla JS, Glassmorphism CSS, and GSAP animations.
* **[05 - Database Design](./05-database-design.md)**: PostgreSQL schemas, ER diagrams, and normalization strategies.
* **[06 - Judge Architecture](./06-judge-architecture.md)**: (Critical) Execution engine, LRU caching, Singleflight, Docker sandboxing, and Semaphore limits.

### 3. Security, Performance & APIs
* **[07 - Security Audit](./07-security-audit.md)**: Threat modeling, sandbox security, and vulnerability assessments.
* **[08 - Performance Analysis](./08-performance-analysis.md)**: Bottleneck identification and tmpfs/LRU optimizations.
* **[09 - API Reference](./09-api-reference.md)**: Endpoints, JWT formats, and payload structures.

### 4. Operations & Deployment
* **[10 - Deployment Handbook](./10-deployment-handbook.md)**: Oracle Cloud Free Tier provisioning, Nginx, and Systemd configs.
* **[11 - Monitoring & Observability](./10-deployment-handbook.md#11---monitoring--observability)**: Future integration with Prometheus and Grafana.
* **[12 - Engineering Decisions](./10-deployment-handbook.md#12---engineering-decisions)**: The "Why" behind Go, Docker, Redis, and custom Worker Pools.

### 5. Production Readiness
* **[13 - Production Readiness Audit](./13-production-readiness-audit.md)**: Health scores, critical blockers, and technical debt.
* **[14 - Testing Strategy](./13-production-readiness-audit.md#14---testing-strategy)**: CI/CD integration, unit testing, and load testing roadmaps.

---

> "The true cost of software is not in its creation, but in its maintenance." 
> *Use this handbook to maintain the integrity of NicheCP.*
