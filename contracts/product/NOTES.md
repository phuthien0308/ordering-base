# Product Service Architecture Notes

## Database & Search Strategy
For a highly dynamic product catalog where the `attributes` vary heavily per product, the standard architectural pattern is a dual-datastore approach:

1. **Primary Datastore (NoSQL - e.g., MongoDB, DynamoDB)**
   - **Why NoSQL?** The schema-less nature of document databases handles the dynamic `map<string, string> attributes` natively without complex and slow SQL join tables (like the Entity-Attribute-Value pattern).
   - **Use Case:** Creating, Updating, getting a single Product by SKU, and Deleting.

2. **Search Engine (Elasticsearch / OpenSearch)**
   - **Why Elasticsearch?** While NoSQL is great for storage, it struggles with complex, multi-field indexing and full-text search. Elasticsearch is built explicitly for this.
   - **Use Case:** Powering the `SearchProducts` gRPC endpoint. It allows for full-text search (e.g., "red running shoes") and **faceted search** (filtering strictly by `color=red` and `brand=nike`).

## Syncing Data (The CQRS Pattern)
Because we have two databases, they need to be kept in sync:
- **Write Path:** When a `CreateProduct` or `UpdateProduct` RPC is called, the service writes the source of truth to the NoSQL database. 
- **Syncing:** The system then emits an event (e.g., to Kafka or RabbitMQ, or using Change Data Capture like Debezium) which is picked up by a worker that indexes the new/updated product into Elasticsearch.
- **Read Path:** The `SearchProducts` RPC bypasses the NoSQL database completely and directly queries Elasticsearch to return incredibly fast results.

## Future gRPC Expansion
When Elasticsearch is ready to be integrated, the `SearchProductsRequest` should be expanded to allow the frontend to pass structured filters:

```protobuf
message SearchProductsRequest {
    string query = 1; // Full text search term (e.g., "running shoes")
    int32 page_size = 2; // Pagination
    string page_token = 3;  
    
    // Allows UI to pass exact matching facets 
    // e.g., {"color": "red", "brand": "nike", "size": "10"}
    map<string, string> attribute_filters = 4; 
}
```
