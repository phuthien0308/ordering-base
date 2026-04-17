output "domain_arn" {
  value = aws_opensearch_domain.this.arn
}

output "domain_endpoint" {
  value = aws_opensearch_domain.this.endpoint
}
