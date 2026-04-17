# Infrastructure

This directory contains Terraform configuration to provision local development infrastructure using [LocalStack](https://docs.localstack.cloud/).

## Project Structure

```
infrastructure/
├── environments/
│   └── local/          # The CALLER — run terraform apply from here
│       ├── main.tf     # Declares which modules to use and with what config
│       ├── providers.tf
│       └── variables.tf
└── modules/
    ├── dynamodb/       # Reusable DynamoDB module (generic, never run directly)
    └── opensearch/     # Reusable OpenSearch module (generic, never run directly)
```

**Modules** are reusable building blocks. **Environments** are callers that assemble those modules with environment-specific configuration. You always run `terraform apply` from an environment directory, not from a module directory.

---

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.5.0
- [LocalStack](https://docs.localstack.cloud/getting-started/installation/) running locally on port `4566`.

Store your LocalStack auth token in a `.env` file (git-ignored, never commit this):

```bash
# .env
LOCALSTACK_AUTH_TOKEN=your-token-here
```

Then start LocalStack:
```bash
export $(cat .env) && localstack start
```

---

## Running Locally

```bash
cd infrastructure/environments/local
terraform init
terraform apply
```

This provisions all resources defined in `environments/local/main.tf` — currently:
- A `Products` DynamoDB table with `sku` as the partition key and two GSIs (`CategoryPriceIndex`, `CategoryRatingIndex`)
- A `products-search` OpenSearch domain

---

## Adding a New Table

To add a new DynamoDB table, add a new `module` block in `environments/local/main.tf`:

```hcl
module "dynamodb_orders" {
  source     = "../../modules/dynamodb"
  table_name = "Orders"
  hash_key   = "user_id"
  range_key  = "order_date"   # optional sort key
  environment = var.environment

  attributes = [
    { name = "user_id",    type = "S" },
    { name = "order_date", type = "S" },
    { name = "status",     type = "S" },
  ]

  global_secondary_indexes = [
    {
      name            = "StatusIndex"
      hash_key        = "status"
      projection_type = "ALL"
    }
  ]
}
```

The `modules/dynamodb` code itself never needs to change.

---

## Module Reference

### DynamoDB Module (`modules/dynamodb`)

| Variable | Type | Required | Description |
|---|---|---|---|
| `table_name` | `string` | ✅ | DynamoDB table name |
| `hash_key` | `string` | ✅ | Partition key attribute name |
| `range_key` | `string` | ❌ | Sort key attribute name (omit for simple primary key) |
| `attributes` | `list(object)` | ✅ | All key attributes used by the table or any index |
| `global_secondary_indexes` | `list(object)` | ❌ | GSI definitions |
| `billing_mode` | `string` | ❌ | `PAY_PER_REQUEST` (default) or `PROVISIONED` |
| `environment` | `string` | ✅ | Environment tag |

**Outputs**

| Output | Description |
|---|---|
| `table_arn` | ARN of the DynamoDB table |
| `table_name` | Name of the DynamoDB table |

---

### OpenSearch Module (`modules/opensearch`)

| Variable | Type | Required | Description |
|---|---|---|---|
| `domain_name` | `string` | ✅ | OpenSearch domain name |
| `environment` | `string` | ✅ | Environment tag |

**Outputs**

| Output | Description |
|---|---|
| `domain_arn` | ARN of the OpenSearch domain |
| `domain_endpoint` | Endpoint URL of the domain |

---

## Teardown

```bash
cd infrastructure/environments/local
terraform destroy
```

## State

State is stored locally at `environments/local/terraform.tfstate`. This file is environment-specific and is git-ignored — do not commit it.
