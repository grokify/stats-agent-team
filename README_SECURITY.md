# Security Guide for Statistics Agent Team

This document provides comprehensive security recommendations for deploying the Statistics Agent Team system, covering credential management, cloud deployment (AWS/Azure), and local development scenarios.

## Table of Contents

- [Overview](#overview)
- [Threat Model](#threat-model)
- [Credential Management](#credential-management)
  - [Current State: Environment Variables](#current-state-environment-variables)
  - [Cloud Secrets Managers](#cloud-secrets-managers)
  - [Workload Identity with SPIFFE/SPIRE](#workload-identity-with-spiffespire)
  - [Enterprise Identity: Okta Integration](#enterprise-identity-okta-integration)
- [Cloud Deployment Security](#cloud-deployment-security)
  - [AWS Security Configuration](#aws-security-configuration)
  - [Azure Security Configuration](#azure-security-configuration)
- [Local Development Security](#local-development-security)
- [Network Security](#network-security)
- [API Security](#api-security)
- [Container Security](#container-security)
- [Monitoring and Auditing](#monitoring-and-auditing)
- [Incident Response](#incident-response)
- [Compliance Considerations](#compliance-considerations)

---

## Overview

The Statistics Agent Team is a multi-agent system that requires several types of credentials:

| Credential Type | Purpose | Sensitivity |
|----------------|---------|-------------|
| **LLM API Keys** | Gemini, Claude, OpenAI, xAI | High - billing, rate limits |
| **Search API Keys** | Serper, SerpAPI | Medium - billing |
| **Inter-Agent Auth** | A2A protocol tokens | Medium - internal access |
| **Database Credentials** | Future: result caching | High - data access |

**Key Principle**: Never store secrets in code, configuration files committed to version control, or container images.

---

## Threat Model

### Attack Vectors

1. **Credential Exposure**
   - Secrets in source code or logs
   - Environment variable leakage
   - Container image inspection

2. **Network Attacks**
   - Man-in-the-middle on inter-agent communication
   - Unauthorized API access
   - DNS spoofing

3. **Application Vulnerabilities**
   - Injection attacks via user topics
   - Server-Side Request Forgery (SSRF) via URL fetching
   - Denial of Service through resource exhaustion

4. **Supply Chain**
   - Compromised dependencies
   - Malicious container base images

### Security Objectives

- **Confidentiality**: Protect API keys and user queries
- **Integrity**: Ensure verified statistics are not tampered with
- **Availability**: Maintain service uptime and rate limit management

---

## Credential Management

### Current State: Environment Variables

The current implementation uses environment variables for credential storage:

```bash
# Current approach (basic security)
export GOOGLE_API_KEY="your-key"
export SERPER_API_KEY="your-key"
```

**Limitations**:
- Credentials visible in process listings (`ps aux`)
- Persisted in shell history
- Visible in Docker inspect
- No rotation mechanism
- No audit trail

### Cloud Secrets Managers

#### AWS Secrets Manager

**Recommended for AWS deployments.** Provides automatic rotation, fine-grained IAM policies, and audit logging.

**Setup**:

1. **Create secrets**:
```bash
aws secretsmanager create-secret \
    --name "stats-agent/llm-keys" \
    --description "LLM API keys for Statistics Agent" \
    --secret-string '{
        "GEMINI_API_KEY": "your-gemini-key",
        "ANTHROPIC_API_KEY": "your-anthropic-key",
        "OPENAI_API_KEY": "your-openai-key"
    }'

aws secretsmanager create-secret \
    --name "stats-agent/search-keys" \
    --description "Search API keys" \
    --secret-string '{
        "SERPER_API_KEY": "your-serper-key"
    }'
```

2. **IAM Policy for ECS/EKS**:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue"
            ],
            "Resource": [
                "arn:aws:secretsmanager:*:*:secret:stats-agent/*"
            ]
        }
    ]
}
```

3. **Integration in Go**:
```go
import (
    "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func LoadSecretsFromAWS(ctx context.Context) (map[string]string, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, err
    }

    client := secretsmanager.NewFromConfig(cfg)
    result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String("stats-agent/llm-keys"),
    })
    if err != nil {
        return nil, err
    }

    var secrets map[string]string
    json.Unmarshal([]byte(*result.SecretString), &secrets)
    return secrets, nil
}
```

4. **ECS Task Definition with Secrets**:
```json
{
    "containerDefinitions": [{
        "name": "stats-agent-eino",
        "secrets": [
            {
                "name": "GEMINI_API_KEY",
                "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:stats-agent/llm-keys:GEMINI_API_KEY::"
            },
            {
                "name": "SERPER_API_KEY",
                "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:stats-agent/search-keys:SERPER_API_KEY::"
            }
        ]
    }]
}
```

#### AWS Systems Manager Parameter Store

**Lower cost alternative** for simpler use cases without automatic rotation needs.

```bash
# Create parameters (use SecureString for secrets)
aws ssm put-parameter \
    --name "/stats-agent/gemini-api-key" \
    --type "SecureString" \
    --value "your-gemini-key"
```

#### Azure Key Vault

**Recommended for Azure deployments.**

1. **Create Key Vault**:
```bash
az keyvault create \
    --name stats-agent-vault \
    --resource-group stats-agent-rg \
    --location eastus

az keyvault secret set \
    --vault-name stats-agent-vault \
    --name "gemini-api-key" \
    --value "your-gemini-key"
```

2. **Managed Identity for AKS/Container Apps**:
```bash
# Enable managed identity
az aks update \
    --resource-group stats-agent-rg \
    --name stats-agent-aks \
    --enable-managed-identity

# Grant access to Key Vault
az keyvault set-policy \
    --name stats-agent-vault \
    --object-id <managed-identity-object-id> \
    --secret-permissions get list
```

3. **Integration in Go**:
```go
import (
    "github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
    "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func LoadSecretsFromAzure(ctx context.Context) (map[string]string, error) {
    cred, err := azidentity.NewDefaultAzureCredential(nil)
    if err != nil {
        return nil, err
    }

    client, err := azsecrets.NewClient(
        "https://stats-agent-vault.vault.azure.net/",
        cred,
        nil,
    )
    if err != nil {
        return nil, err
    }

    resp, err := client.GetSecret(ctx, "gemini-api-key", "", nil)
    if err != nil {
        return nil, err
    }

    return map[string]string{
        "GEMINI_API_KEY": *resp.Value,
    }, nil
}
```

### Workload Identity with SPIFFE/SPIRE

**SPIFFE** (Secure Production Identity Framework for Everyone) provides cryptographic identities to workloads, eliminating static credentials for service-to-service authentication.

#### Use Cases for Stats Agent

1. **Inter-agent authentication** (Research ↔ Synthesis ↔ Verification)
2. **Zero-trust network communication**
3. **Short-lived, automatically rotated credentials**

#### SPIRE Deployment

1. **Install SPIRE Server**:
```yaml
# spire-server.yaml for Kubernetes
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: spire-server
  namespace: spire
spec:
  selector:
    matchLabels:
      app: spire-server
  template:
    spec:
      containers:
      - name: spire-server
        image: ghcr.io/spiffe/spire-server:1.9.0
        args:
          - -config
          - /run/spire/config/server.conf
        volumeMounts:
        - name: spire-config
          mountPath: /run/spire/config
```

2. **Configure SPIRE Agent**:
```hcl
# agent.conf
agent {
    data_dir = "/opt/spire/data"
    log_level = "INFO"
    server_address = "spire-server"
    server_port = "8081"
    socket_path = "/run/spire/sockets/agent.sock"
    trust_bundle_path = "/opt/spire/conf/bootstrap.crt"
    trust_domain = "stats-agent.example.com"
}

plugins {
    NodeAttestor "k8s_psat" {
        plugin_data {
            cluster = "stats-agent-cluster"
        }
    }

    WorkloadAttestor "k8s" {
        plugin_data {
            skip_kubelet_verification = true
        }
    }
}
```

3. **Register Workloads** (assign SPIFFE IDs):
```bash
# Register Research Agent
spire-server entry create \
    -spiffeID spiffe://stats-agent.example.com/research-agent \
    -parentID spiffe://stats-agent.example.com/k8s-node \
    -selector k8s:ns:stats-agent \
    -selector k8s:sa:research-agent

# Register Synthesis Agent
spire-server entry create \
    -spiffeID spiffe://stats-agent.example.com/synthesis-agent \
    -parentID spiffe://stats-agent.example.com/k8s-node \
    -selector k8s:ns:stats-agent \
    -selector k8s:sa:synthesis-agent
```

4. **Go Integration with go-spiffe**:
```go
import (
    "github.com/spiffe/go-spiffe/v2/spiffeid"
    "github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
    "github.com/spiffe/go-spiffe/v2/workloadapi"
)

func NewSecureHTTPClient(ctx context.Context) (*http.Client, error) {
    // Create X509Source from Workload API
    source, err := workloadapi.NewX509Source(ctx)
    if err != nil {
        return nil, fmt.Errorf("unable to create X509Source: %w", err)
    }

    // Authorize connections to specific SPIFFE IDs
    authorizedIDs := []spiffeid.ID{
        spiffeid.RequireFromString("spiffe://stats-agent.example.com/synthesis-agent"),
        spiffeid.RequireFromString("spiffe://stats-agent.example.com/verification-agent"),
    }

    tlsConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeOneOf(authorizedIDs...))

    return &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: tlsConfig,
        },
    }, nil
}
```

### Enterprise Identity: Okta Integration

For enterprise deployments requiring centralized identity management, integrate with **Okta** for:
- User authentication for API access
- Service account management
- Fine-grained authorization policies

#### Okta API Access Management (XAA)

1. **Create API Authorization Server**:
```bash
# Via Okta Admin Console or API
curl -X POST "https://${OKTA_DOMAIN}/api/v1/authorizationServers" \
  -H "Authorization: SSWS ${OKTA_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Stats Agent API",
    "description": "Authorization server for Statistics Agent",
    "audiences": ["api://stats-agent"]
  }'
```

2. **Define Scopes**:
```json
{
    "scopes": [
        {"name": "stats:read", "description": "Read statistics"},
        {"name": "stats:search", "description": "Search for statistics"},
        {"name": "stats:verify", "description": "Verify statistics"},
        {"name": "admin:agents", "description": "Manage agent configuration"}
    ]
}
```

3. **Create Access Policies**:
```json
{
    "name": "Stats Agent Default Policy",
    "conditions": {
        "clients": {"include": ["ALL_CLIENTS"]}
    },
    "rules": [{
        "name": "Standard Access",
        "conditions": {
            "grantTypes": {"include": ["client_credentials"]},
            "scopes": {"include": ["stats:read", "stats:search"]}
        },
        "actions": {
            "token": {
                "accessTokenLifetimeMinutes": 60,
                "refreshTokenLifetimeMinutes": 0
            }
        }
    }]
}
```

4. **Go Middleware for JWT Validation**:
```go
import (
    "github.com/okta/okta-jwt-verifier-golang"
)

func OktaAuthMiddleware(next http.Handler) http.Handler {
    verifier := jwtverifier.JwtVerifier{
        Issuer:           "https://your-okta-domain.okta.com/oauth2/default",
        ClaimsToValidate: map[string]string{
            "aud": "api://stats-agent",
        },
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearerToken(r)
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        _, err := verifier.VerifyAccessToken(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

#### Okta Identity Governance (OIG) / Just-in-Time Access

For sensitive operations, implement **Just-in-Time (JIT) access** using Okta Identity Governance:

1. **Define Access Bundles**:
```yaml
# Access bundle for admin operations
name: stats-agent-admin
description: "Admin access to Statistics Agent"
resources:
  - type: api_scope
    name: admin:agents
  - type: api_scope
    name: stats:verify
max_duration: 4h
approval_required: true
approvers:
  - group: security-team
```

2. **Request Access Flow**:
```go
// Example: Request elevated access for admin operations
func RequestElevatedAccess(ctx context.Context, userID string) error {
    client := okta.NewClient(ctx, oktaConfig)

    request := &okta.AccessRequest{
        ResourceSetID: "stats-agent-admin",
        Justification: "Need to modify agent configuration for maintenance",
        Duration:      "4h",
    }

    _, err := client.AccessRequests.Create(ctx, userID, request)
    return err
}
```

---

## Cloud Deployment Security

### AWS Security Configuration

#### VPC Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         VPC                                  │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │   Public Subnet     │    │     Private Subnet          │ │
│  │  ┌───────────────┐  │    │  ┌─────────────────────┐   │ │
│  │  │      ALB      │  │    │  │   ECS/EKS Cluster   │   │ │
│  │  │  (HTTPS:443)  │──┼────┼─>│   - Research Agent  │   │ │
│  │  └───────────────┘  │    │  │   - Synthesis Agent │   │ │
│  │                     │    │  │   - Verification    │   │ │
│  │  ┌───────────────┐  │    │  │   - Orchestration   │   │ │
│  │  │   NAT GW      │<─┼────┼──│                     │   │ │
│  │  └───────────────┘  │    │  └─────────────────────┘   │ │
│  └─────────────────────┘    │                             │ │
│                             │  ┌─────────────────────┐   │ │
│                             │  │    VPC Endpoints    │   │ │
│                             │  │  - Secrets Manager  │   │ │
│                             │  │  - ECR              │   │ │
│                             │  │  - CloudWatch       │   │ │
│                             │  └─────────────────────┘   │ │
│                             └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

#### Terraform Configuration

```hcl
# vpc.tf
resource "aws_vpc" "stats_agent" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "stats-agent-vpc"
  }
}

# Security Group for Agents
resource "aws_security_group" "agents" {
  name_prefix = "stats-agent-"
  vpc_id      = aws_vpc.stats_agent.id

  # Internal agent communication
  ingress {
    from_port = 8000
    to_port   = 8005
    protocol  = "tcp"
    self      = true
  }

  # ALB health checks
  ingress {
    from_port       = 8000
    to_port         = 8005
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  # Outbound for LLM/Search APIs
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# VPC Endpoint for Secrets Manager (avoid NAT costs, stay private)
resource "aws_vpc_endpoint" "secrets_manager" {
  vpc_id              = aws_vpc.stats_agent.id
  service_name        = "com.amazonaws.${var.region}.secretsmanager"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = aws_subnet.private[*].id
  security_group_ids  = [aws_security_group.vpc_endpoints.id]
  private_dns_enabled = true
}
```

#### IAM Roles (Principle of Least Privilege)

```hcl
# iam.tf
resource "aws_iam_role" "stats_agent_task" {
  name = "stats-agent-task-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

# Minimal permissions for Secrets Manager
resource "aws_iam_role_policy" "secrets_access" {
  name = "secrets-access"
  role = aws_iam_role.stats_agent_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          "arn:aws:secretsmanager:${var.region}:${var.account_id}:secret:stats-agent/*"
        ]
        Condition = {
          StringEquals = {
            "secretsmanager:VersionStage" = "AWSCURRENT"
          }
        }
      }
    ]
  })
}

# CloudWatch Logs (required for debugging)
resource "aws_iam_role_policy" "cloudwatch_logs" {
  name = "cloudwatch-logs"
  role = aws_iam_role.stats_agent_task.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ]
      Resource = "arn:aws:logs:${var.region}:${var.account_id}:log-group:/ecs/stats-agent:*"
    }]
  })
}
```

### Azure Security Configuration

#### Resource Architecture

```bash
# Create resource group
az group create --name stats-agent-rg --location eastus

# Create virtual network
az network vnet create \
    --resource-group stats-agent-rg \
    --name stats-agent-vnet \
    --address-prefix 10.0.0.0/16 \
    --subnet-name agents-subnet \
    --subnet-prefix 10.0.1.0/24

# Create private endpoint for Key Vault
az network private-endpoint create \
    --resource-group stats-agent-rg \
    --name stats-agent-keyvault-pe \
    --vnet-name stats-agent-vnet \
    --subnet agents-subnet \
    --private-connection-resource-id $(az keyvault show --name stats-agent-vault --query id -o tsv) \
    --group-id vault \
    --connection-name keyvault-connection
```

#### Azure Container Apps with Managed Identity

```bicep
// main.bicep
resource containerApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: 'stats-agent-eino'
  location: resourceGroup().location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    configuration: {
      ingress: {
        external: true
        targetPort: 8000
        transport: 'http'
      }
      secrets: [
        {
          name: 'gemini-api-key'
          keyVaultUrl: 'https://stats-agent-vault.vault.azure.net/secrets/gemini-api-key'
          identity: 'system'
        }
      ]
    }
    template: {
      containers: [
        {
          name: 'stats-agent'
          image: 'your-registry.azurecr.io/stats-agent:latest'
          env: [
            {
              name: 'GEMINI_API_KEY'
              secretRef: 'gemini-api-key'
            }
          ]
        }
      ]
    }
  }
}

// Grant Key Vault access
resource keyVaultAccessPolicy 'Microsoft.KeyVault/vaults/accessPolicies@2023-07-01' = {
  name: 'add'
  parent: keyVault
  properties: {
    accessPolicies: [
      {
        tenantId: subscription().tenantId
        objectId: containerApp.identity.principalId
        permissions: {
          secrets: ['get', 'list']
        }
      }
    ]
  }
}
```

---

## Local Development Security

### Secure Local Development Setup

1. **Use a secrets manager even locally**:
```bash
# Install 1Password CLI or similar
brew install 1password-cli

# Load secrets into environment (ephemeral)
eval $(op signin)
export GEMINI_API_KEY=$(op read "op://Development/Stats-Agent/gemini-key")
export SERPER_API_KEY=$(op read "op://Development/Stats-Agent/serper-key")
```

2. **Use direnv for project-specific environments**:
```bash
# .envrc (gitignored)
export GEMINI_API_KEY="$(op read 'op://Development/Stats-Agent/gemini-key')"
export SERPER_API_KEY="$(op read 'op://Development/Stats-Agent/serper-key')"
```

3. **Docker secrets for local compose**:
```yaml
# docker-compose.local.yml
services:
  stats-agent-eino:
    environment:
      - GEMINI_API_KEY_FILE=/run/secrets/gemini_key
    secrets:
      - gemini_key

secrets:
  gemini_key:
    file: ./secrets/gemini_key.txt  # gitignored
```

### .gitignore Security Entries

```gitignore
# Secrets - NEVER commit
.env
.env.local
.env.*.local
secrets/
*.key
*.pem
*credentials*
*secret*

# IDE might cache secrets
.idea/
.vscode/settings.json

# Shell history might contain secrets
.bash_history
.zsh_history
```

### Pre-commit Hooks for Secret Detection

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks

  - repo: https://github.com/trufflesecurity/trufflehog
    rev: v3.63.0
    hooks:
      - id: trufflehog
```

Install and run:
```bash
pip install pre-commit
pre-commit install
pre-commit run --all-files
```

---

## Network Security

### TLS Configuration

1. **External Traffic**: Always use TLS 1.2+ for external endpoints
2. **Inter-Agent Communication**: Use mTLS with SPIFFE or service mesh

```yaml
# Istio service mesh example
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: stats-agent-mtls
  namespace: stats-agent
spec:
  mtls:
    mode: STRICT

---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: stats-agent-policy
  namespace: stats-agent
spec:
  rules:
  - from:
    - source:
        principals:
        - "cluster.local/ns/stats-agent/sa/orchestration-agent"
    to:
    - operation:
        methods: ["POST"]
        paths: ["/search", "/synthesize", "/verify"]
```

### Network Policies (Kubernetes)

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: stats-agent-policy
  namespace: stats-agent
spec:
  podSelector:
    matchLabels:
      app: stats-agent
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Only from ingress controller
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - port: 8000
  # Inter-agent communication
  - from:
    - podSelector:
        matchLabels:
          app: stats-agent
    ports:
    - port: 8001
    - port: 8002
    - port: 8003
    - port: 8004
  egress:
  # LLM APIs (HTTPS)
  - to:
    - ipBlock:
        cidr: 0.0.0.0/0
    ports:
    - port: 443
      protocol: TCP
  # DNS
  - to:
    - namespaceSelector: {}
    ports:
    - port: 53
      protocol: UDP
```

---

## API Security

### Rate Limiting

```go
import (
    "golang.org/x/time/rate"
    "sync"
)

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rate:     rate.Limit(rps),
        burst:    burst,
    }
}

func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[key]
    if !exists {
        limiter = rate.NewLimiter(rl.rate, rl.burst)
        rl.limiters[key] = limiter
    }
    return limiter
}

func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("X-API-Key")
            if key == "" {
                key = r.RemoteAddr
            }

            if !rl.GetLimiter(key).Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Input Validation

```go
import (
    "regexp"
    "strings"
)

const (
    MaxTopicLength = 500
    MaxURLLength   = 2048
)

var (
    // Prevent injection attacks
    dangerousPatterns = regexp.MustCompile(`(?i)(script|javascript|vbscript|data:)`)

    // Allow only safe URL schemes
    allowedSchemes = []string{"http://", "https://"}
)

func ValidateTopic(topic string) error {
    if len(topic) > MaxTopicLength {
        return fmt.Errorf("topic exceeds maximum length of %d", MaxTopicLength)
    }

    if dangerousPatterns.MatchString(topic) {
        return fmt.Errorf("topic contains potentially dangerous content")
    }

    return nil
}

func ValidateURL(urlStr string) error {
    if len(urlStr) > MaxURLLength {
        return fmt.Errorf("URL exceeds maximum length")
    }

    for _, scheme := range allowedSchemes {
        if strings.HasPrefix(strings.ToLower(urlStr), scheme) {
            return nil
        }
    }

    return fmt.Errorf("URL must use http or https scheme")
}
```

### SSRF Prevention

The verification agent fetches URLs, which creates SSRF risk. Mitigate with:

```go
import (
    "net"
    "net/url"
)

var (
    privateIPBlocks []*net.IPNet
)

func init() {
    for _, cidr := range []string{
        "127.0.0.0/8",    // Loopback
        "10.0.0.0/8",     // RFC1918
        "172.16.0.0/12",  // RFC1918
        "192.168.0.0/16", // RFC1918
        "169.254.0.0/16", // Link-local
        "::1/128",        // IPv6 loopback
        "fc00::/7",       // IPv6 private
        "fe80::/10",      // IPv6 link-local
    } {
        _, block, _ := net.ParseCIDR(cidr)
        privateIPBlocks = append(privateIPBlocks, block)
    }
}

func isPrivateIP(ip net.IP) bool {
    for _, block := range privateIPBlocks {
        if block.Contains(ip) {
            return true
        }
    }
    return false
}

func SafeFetch(urlStr string) (*http.Response, error) {
    parsed, err := url.Parse(urlStr)
    if err != nil {
        return nil, err
    }

    // Resolve hostname
    ips, err := net.LookupIP(parsed.Hostname())
    if err != nil {
        return nil, err
    }

    for _, ip := range ips {
        if isPrivateIP(ip) {
            return nil, fmt.Errorf("access to private IP addresses is not allowed")
        }
    }

    // Use a custom transport that doesn't follow redirects to private IPs
    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            // Re-validate redirect target
            ips, _ := net.LookupIP(req.URL.Hostname())
            for _, ip := range ips {
                if isPrivateIP(ip) {
                    return fmt.Errorf("redirect to private IP not allowed")
                }
            }
            return nil
        },
        Timeout: 30 * time.Second,
    }

    return client.Get(urlStr)
}
```

---

## Container Security

### Dockerfile Best Practices

```dockerfile
# Dockerfile.secure
# Build stage
FROM golang:1.21-alpine AS builder

# Security: Run as non-root during build
RUN adduser -D -g '' appuser
USER appuser

WORKDIR /app
COPY --chown=appuser:appuser go.mod go.sum ./
RUN go mod download

COPY --chown=appuser:appuser . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /stats-agent ./main.go

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Security: Use non-root user (65532 is nonroot in distroless)
USER nonroot:nonroot

COPY --from=builder /stats-agent /stats-agent

# Security: Read-only filesystem
ENTRYPOINT ["/stats-agent"]
```

### Image Scanning

```bash
# Scan with Trivy
trivy image --severity HIGH,CRITICAL your-registry/stats-agent:latest

# Scan with Grype
grype your-registry/stats-agent:latest
```

### Kubernetes Pod Security

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: stats-agent
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    fsGroup: 65532
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: stats-agent
    image: your-registry/stats-agent:latest
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
    resources:
      limits:
        cpu: "1"
        memory: "512Mi"
      requests:
        cpu: "100m"
        memory: "128Mi"
```

---

## Monitoring and Auditing

### Structured Logging (No Secrets)

```go
import (
    "log/slog"
)

func NewSecureLogger() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            // Redact sensitive fields
            sensitiveKeys := []string{"api_key", "token", "password", "secret", "authorization"}
            for _, key := range sensitiveKeys {
                if strings.Contains(strings.ToLower(a.Key), key) {
                    return slog.Attr{Key: a.Key, Value: slog.StringValue("[REDACTED]")}
                }
            }
            return a
        },
    }))
}

// Usage
logger.Info("processing request",
    slog.String("topic", req.Topic),
    slog.String("api_key", apiKey),  // Will be redacted
    slog.Int("min_stats", req.MinStats),
)
```

### Audit Trail

```go
type AuditEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    Action      string    `json:"action"`
    UserID      string    `json:"user_id,omitempty"`
    Resource    string    `json:"resource"`
    Result      string    `json:"result"`
    SourceIP    string    `json:"source_ip"`
    RequestID   string    `json:"request_id"`
    Details     any       `json:"details,omitempty"`
}

func LogAuditEvent(ctx context.Context, event AuditEvent) {
    event.Timestamp = time.Now()
    event.RequestID = GetRequestID(ctx)

    // Send to audit log stream (CloudWatch, Azure Monitor, etc.)
    auditLogger.Info("audit",
        slog.String("action", event.Action),
        slog.String("user_id", event.UserID),
        slog.String("resource", event.Resource),
        slog.String("result", event.Result),
        slog.String("source_ip", event.SourceIP),
        slog.String("request_id", event.RequestID),
    )
}
```

### Metrics for Security Monitoring

```go
import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    authFailures = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "stats_agent_auth_failures_total",
            Help: "Total authentication failures",
        },
        []string{"reason", "source_ip"},
    )

    rateLimitHits = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "stats_agent_rate_limit_hits_total",
            Help: "Rate limit violations",
        },
        []string{"client_id"},
    )

    suspiciousRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "stats_agent_suspicious_requests_total",
            Help: "Requests flagged as suspicious",
        },
        []string{"reason"},
    )
)

func init() {
    prometheus.MustRegister(authFailures, rateLimitHits, suspiciousRequests)
}
```

---

## Incident Response

### Credential Rotation Procedure

If a credential is compromised:

1. **Immediately rotate** the compromised credential in the secrets manager
2. **Restart affected services** to pick up new credentials
3. **Review audit logs** for unauthorized access
4. **Assess impact** - what data/services were accessible
5. **Notify affected parties** if user data was accessed

```bash
# AWS: Rotate secret
aws secretsmanager rotate-secret --secret-id stats-agent/llm-keys

# Azure: Create new version
az keyvault secret set \
    --vault-name stats-agent-vault \
    --name gemini-api-key \
    --value "new-rotated-key"

# Restart services (ECS)
aws ecs update-service \
    --cluster stats-agent \
    --service stats-agent-eino \
    --force-new-deployment
```

### Security Contact

For security issues with this project, contact the maintainers via GitHub Security Advisories.

---

## Compliance Considerations

### SOC 2 Alignment

| Control | Implementation |
|---------|----------------|
| CC6.1 - Logical Access | Okta integration, IAM policies |
| CC6.2 - System Access | SPIFFE/SPIRE workload identity |
| CC6.3 - Data Classification | Secrets in dedicated managers |
| CC6.6 - Encryption | TLS for transit, KMS for rest |
| CC7.2 - System Monitoring | CloudWatch/Azure Monitor logs |

### GDPR Considerations

- **User queries** may contain personal data
- Implement data retention policies
- Provide data export/deletion capabilities
- Log access to personal data

### PCI-DSS (if handling payment data)

- Network segmentation
- Encryption of cardholder data
- Access control measures
- Regular security testing

---

## Summary: Security Checklist

### Before Deployment

- [ ] Secrets stored in cloud secrets manager (not environment variables)
- [ ] IAM roles follow least privilege principle
- [ ] Container images scanned for vulnerabilities
- [ ] Network policies restrict inter-service communication
- [ ] TLS enabled for all external endpoints
- [ ] Pre-commit hooks detect secret leakage
- [ ] Input validation implemented
- [ ] SSRF protections in place

### Ongoing Operations

- [ ] Regular credential rotation (90 days max)
- [ ] Security monitoring alerts configured
- [ ] Audit logs reviewed periodically
- [ ] Dependency updates for security patches
- [ ] Penetration testing annually
- [ ] Incident response plan documented

### For Enterprise Deployments

- [ ] Okta/IdP integration for user authentication
- [ ] SPIFFE/SPIRE for workload identity
- [ ] Service mesh (Istio/Linkerd) for mTLS
- [ ] Just-in-time access for admin operations
- [ ] SOC 2 controls documented

---

## References

- [AWS Secrets Manager Best Practices](https://docs.aws.amazon.com/secretsmanager/latest/userguide/best-practices.html)
- [Azure Key Vault Security](https://learn.microsoft.com/en-us/azure/key-vault/general/security-features)
- [SPIFFE/SPIRE Documentation](https://spiffe.io/docs/latest/)
- [Okta API Access Management](https://developer.okta.com/docs/concepts/api-access-management/)
- [OWASP API Security Top 10](https://owasp.org/API-Security/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
