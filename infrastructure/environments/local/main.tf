# Local infrastructure using AWS resources mocked via LocalStack

module "dynamodb_products" {
  source = "../../modules/dynamodb"

  table_name  = "Products"
  hash_key    = "sku"
  environment = var.environment

  # All attributes used as partition/sort keys or GSI keys must be declared here.
  # Do NOT add data attributes (name, description, etc.) — DynamoDB is schema-less for those.
  attributes = [
    { name = "sku",      type = "S" },
    { name = "category", type = "S" },
    { name = "price",    type = "N" },
    { name = "rating",   type = "N" },
  ]

  global_secondary_indexes = [
    {
      name            = "CategoryPriceIndex"
      hash_key        = "category"
      range_key       = "price"
      projection_type = "ALL"
    },
    {
      name            = "CategoryRatingIndex"
      hash_key        = "category"
      range_key       = "rating"
      projection_type = "ALL"
    },
  ]
}

module "opensearch_cluster" {
  source = "../../modules/opensearch"

  domain_name = "products-search"
  environment = var.environment
}
