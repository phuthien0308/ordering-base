variable "table_name" {
  type        = string
  description = "The name of the DynamoDB table"
}

variable "hash_key" {
  type        = string
  description = "Partition key attribute name"
}

variable "range_key" {
  type        = string
  description = "Sort key attribute name. Leave empty for simple primary key."
  default     = null
}

variable "attributes" {
  description = "List of all key attributes required by the table and its indexes"
  type = list(object({
    name = string
    type = string # "S" (String), "N" (Number), "B" (Binary)
  }))
}

variable "global_secondary_indexes" {
  description = "List of GSIs to create on this table"
  type = list(object({
    name            = string
    hash_key        = string
    range_key       = optional(string)
    projection_type = string # ALL, KEYS_ONLY, INCLUDE
  }))
  default = []
}

variable "billing_mode" {
  type        = string
  description = "PAY_PER_REQUEST or PROVISIONED"
  default     = "PAY_PER_REQUEST"
}

variable "environment" {
  type        = string
  description = "The environment context (e.g. local)"
}
