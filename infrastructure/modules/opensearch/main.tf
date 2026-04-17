resource "aws_opensearch_domain" "this" {
  domain_name           = var.domain_name
  engine_version        = "OpenSearch_2.11" # Specify latest stable available in localstack

  cluster_config {
    instance_type          = "t3.small.search"
    instance_count         = 1
    zone_awareness_enabled = false
  }

  ebs_options {
    ebs_enabled = true
    volume_size = 10
    volume_type = "gp3"
  }

  # For local development with LocalStack, we don't strictly need complex IAM setups,
  # but here we set open access policies mocked down for local container.
  access_policies = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "es:*"
        Principal = "*"
        Effect = "Allow"
        Resource = "arn:aws:es:us-east-1:000000000000:domain/${var.domain_name}/*"
      }
    ]
  })

  tags = {
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}
