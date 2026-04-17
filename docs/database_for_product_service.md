# Database Strategy for Product Service

## Overview
The `ProductService` requires a robust, scalable, and highly available database strategy. Since products natively have flexible, dynamic attributes (e.g., a TV has different properties than a t-shirt), traditional relational schemas can become unwieldy. 

For local development and future production deployments within the AWS ecosystem, our database architecture heavily leverages **Amazon DynamoDB** as the primary storage layer, and **Amazon OpenSearch** as our read-optimized query layer.

---

## 1. Primary Data Store: DynamoDB

### Why DynamoDB?
- **Schema-less Flexibility:** Our Protobuf contract utilizes a `map<string, string> attributes` field to capture dynamic product specs. DynamoDB natively stores maps and lists without requiring predefined columns, allowing us to seamlessly adapt to new product types.
- **Performance:** DynamoDB provides flat, predictable single-digit millisecond response times regardless of whether the table contains 10 records or 10 billion records. Lookups natively indexing the `sku` are exceptionally fast.
- **Serverless Scaling:** It automatically partitions and scales to handle massive traffic spikes (like Black Friday events) and scales down transparently, eliminating the operational overhead of provisioning instances.

### Table Design (Draft)
- **Table Name:** `Products`
- **Partition Key (Hash Key):** `sku` (String)

---

## 2. Advanced Search & CQRS Layer: OpenSearch

### The Search Requirement
While DynamoDB is incredibly fast for direct `sku` lookups, it is not designed for complex aggregations, wildcard text searches, or querying secondary, non-key attributes efficiently without setting up numerous Global Secondary Indexes (GSIs).

### The Implementation
To power the `SearchProducts` RPC (which handles faceted filtering via `attribute_filters` and text queries), we utilize the **CQRS (Command Query Responsibility Segregation)** pattern:
1. **Writes** (Create, Update, Delete) are sent to **DynamoDB**.
2. **DynamoDB Streams** actively monitors all table mutations.
3. A Lambda function (or background Go worker) listens to this stream and pushes the updated records into an **Amazon OpenSearch cluster**.
4. **Reads** (Searches, aggregations, filtering) are directed straight to **OpenSearch**, providing lightning-fast faceted search without straining the primary database.

---

## 3. Handling ACID Transactions

Because this service operates within a broader `ordering-base` architecture where money and physical stock are involved, data integrity and prevention of race conditions are non-negotiable.

DynamoDB natively supports full ACID compliance for transactions:

1. **`TransactWriteItems` for Multi-Table Ops**
   - When an order is placed, we can atomically deduct product stock, write the order to the `Orders` table, and deduct from an `Accounts` table. 
   - Operations are packaged as a single atomic batch that can modify up to 100 items simultaneously. If one constraint fails (e.g., negative stock), the entire transaction rolls back.

2. **`ConditionExpression` for High-Speed Concurrency**
   - For single-item updates (like decrementing the available inventory of a specific `sku`), we avoid the overhead of heavy transactions by utilizing condition expressions natively (e.g., `Update inventory = inventory - 1 WHERE inventory > 0`). This safely prevents race conditions during high-volume spikes.
