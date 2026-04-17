# General Database Knowledge

This document serves as an internal wiki for technical concepts, architectural principles, and foundational knowledge regarding databases within our ecosystem.

## Relational Databases (SQL) & ACID Properties

SQL databases (like PostgreSQL and MySQL) have historically been the gold standard for robust, highly structured data due to their incredibly strong adherence to ACID guarantees.

**What is ACID?**
ACID is an acronym representing the four key properties that guarantee database transactions are completely reliable. When building systems handling money, billing, or strict inventory counts, ACID properties prevent race conditions and data corruption.

### 1. Atomicity ("All or Nothing")
In a transaction, you often need multiple steps to succeed together (e.g., deducting an account balance AND creating an active order). Atomicity guarantees that if any single step fails midway (due to a power outage, network drop, or constraint violation), the database perfectly "rolls back" to its previous state. There are never partial completions.

### 2. Consistency ("Rules are Rules")
SQL databases enforce strict schemas and relational rules natively.
- Example: `CHECK (balance >= 0)` or `FOREIGN KEY (user_id) REFERENCES accounts(id)`.
Consistency guarantees that a transaction will never be permitted to execute if it would leave the database in an illegal or invalid state natively defined by these rules.

### 3. Isolation ("Wait Your Turn")
Isolation targets high-concurrency environments—like thousands of users hitting the checkout API on Black Friday. If two users try to buy the absolute last pair of shoes at the exact same millisecond, Isolation guarantees the database treats these transactions as if they executed cleanly one after the other. 

Without Isolation, systems experience critical "Read Phenomena":
* **Dirty Reads:** Transaction B reads data being actively changed by Transaction A *before* A has officially committed. If A rolls back, B possesses a ghost value that never actually existed.
* **Non-Repeatable Reads:** Transaction A reads a row, Transaction B edits and commits changes to that row, then Transaction A reads the same row again and gets a suddenly different result inside the same transaction.
* **Phantom Reads:** Transaction A requests a range of rows (e.g., "all items under $50"). Transaction B inserts a new item under $50. If A asks again, a "phantom" row appears.

To solve these problems, SQL databases provide mechanisms that range on a spectrum from "highly concurrent but slightly risky" to "strictly sequential but slower."

#### How Databases Solve Isolation Problems:
**1. Pessimistic Locking (Row-Level Locking)**
The database physically restricts access to the data. If a developer uses a command like `SELECT ... FOR UPDATE`, the database slaps a lock tightly around those specific rows. Any other transaction attempting to read or modify those rows is literally forced into a waiting queue until the first transaction finishes and unlocks them. 
* *Pros:* 100% guarantees safety.
* *Cons:* It can create bottlenecks (or worst-case scenario: "Deadlocks," where Transaction 1 is waiting on Transaction 2, while Transaction 2 is concurrently waiting on Transaction 1).

**2. Multi-Version Concurrency Control (MVCC)**
Instead of locking data aggressively, MVCC relies on snapshots. When a transaction starts, the database takes an instantaneous snapshot of the data. If the transaction updates a row, the database doesn't overwrite the original row directly; it creates a *new version* of it. 
* This elegantly means **readers don't block writers, and writers don't block readers.** 
* When the transaction finally commits, it checks if any other transaction secretly modified the underlying data in the meantime. If so, it gracefully fails and aborts. PostgreSQL relies heavily on MVCC.

#### The 4 Standard Isolation Levels:
Databases let developers manually choose how strict they want these mechanisms enforced via Isolation Levels. There is always a trade-off: **the stricter the isolation, the safer the data but the slower the performance.**

| Isolation Level    | Dirty Read | Non-Repeatable Read | Phantom Read | Default in        |
|--------------------|:----------:|:-------------------:|:------------:|-------------------|
| Read Uncommitted   | ✅ Possible | ✅ Possible         | ✅ Possible  | —                 |
| Read Committed     | ❌ Prevented| ✅ Possible         | ✅ Possible  | PostgreSQL, MSSQL |
| Repeatable Read    | ❌ Prevented| ❌ Prevented        | ✅ Possible  | MySQL             |
| Serializable       | ❌ Prevented| ❌ Prevented        | ❌ Prevented | —                 |

---

**1. Read Uncommitted** — No isolation at all.
```sql
SET TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;
BEGIN;
SELECT SUM(total) FROM orders; -- May read uncommitted, in-flight orders
COMMIT;
```
* *Example:* You query total daily revenue. A $5,000 order is mid-transaction at that exact moment. Your query reads that $5,000 and includes it, but the order subsequently fails and rolls back. You generated a report with phantom money that never existed.
* *Use when:* You need maximum speed and approximate results are acceptable (e.g., informal dashboards, non-critical analytics). **Avoid for anything financial.**

---

**2. Read Committed** — Only reads committed data. The practical default for most applications.
```sql
SET TRANSACTION ISOLATION LEVEL READ COMMITTED;
BEGIN;
SELECT stock FROM products WHERE sku = 'LAPTOP-01'; -- Only sees committed values
COMMIT;
```
* *Example:* A laptop shows `1` in stock. User A starts checking out but hasn't finalized. User B views the product page and the database honestly says `1` is in stock (it hides User A's uncommitted draft). Safe, but if B queries again in the same transaction *after* A commits their purchase, they might see `0` — the read was not "repeatable".
* *Use when:* General-purpose reads where perfect consistency between multiple queries in the same transaction is not strictly required.

---

**3. Repeatable Read** — Guarantees a row read once stays frozen for the duration of the transaction.
```sql
SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;
BEGIN;
SELECT balance FROM accounts WHERE id = 'User_123'; -- Returns $100
-- ... some application logic runs ...
SELECT balance FROM accounts WHERE id = 'User_123'; -- Still returns $100, even if another txn committed a change
COMMIT;
```
* *Example:* A billing job audits `User_123` and sees $100. Mid-audit, the user buys a $20 shirt. If the auditor queries the same row again inside the *same* transaction, the database returns the frozen snapshot of $100, keeping the script's logic mathematically coherent from start to finish.
* *Use when:* Multi-step reports or billing logic that reads the same rows multiple times and requires consistency across those reads.

---

**4. Serializable** — The strictest level. Transactions execute as if they are the only one running.
```sql
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
BEGIN;
SELECT COUNT(*) FROM promo_codes WHERE active = true; -- Returns 5
-- In another session, no INSERT can sneak in here
COMMIT;
```
* *Example:* You query "5 active promo codes in the database." Simultaneously, an admin inserts a 6th. Under lower levels that 6th row could appear mid-transaction as a "phantom." Under Serializable, the admin's insert is blocked until your entire transaction exits cleanly.
* *Use when:* Absolute correctness is critical (e.g., financial ledgers, double-spend prevention, inventory allocation at high concurrency). Expect throughput to drop significantly under heavy load.

---

#### Deep Dive: Why Can't Repeatable Read Prevent Phantom Reads?

This is a subtle but important distinction. The answer lies in **what Repeatable Read actually locks**.

**Repeatable Read's mental model:**
> *"I will protect every row you have already seen. Nobody can change or delete those specific rows for the duration of your transaction."*

The key phrase is **"rows you have already seen."** It only locks **existing rows that were read**. It is completely blind to rows that do not yet exist.

A Phantom Read is not caused by an existing row being *modified*. It is caused by a brand new row being *inserted* that matches the original query's conditions. Since that row didn't exist when Transaction A first ran its query, Repeatable Read had nothing to lock — you simply cannot lock something that doesn't exist yet.

**The exact failure timeline:**
```
Transaction A                          Transaction B
──────────────────────────────────────────────────────────
SELECT * FROM products
WHERE price < 50;
-- Returns 3 rows. RR locks those 3 rows.

                                       INSERT INTO products
                                       VALUES ('new-sku', 45);
                                       COMMIT; ✅

SELECT * FROM products
WHERE price < 50;
-- NOW returns 4 rows! 👻 The "phantom" row slipped through.
```

Transaction A's first query returned 3 rows — Repeatable Read dutifully locked all 3. But Transaction B inserted a completely *new* row. Since it was never part of A's original result set, there was no lock on it and it slipped right through.

**How Serializable closes the gap — Predicate Locking:**

Instead of locking individual *rows*, Serializable locks the **query condition (predicate) itself**. When Transaction A runs `WHERE price < 50`, the database registers a lock on the *concept* of "any row where price is under 50":
> *"Nothing may insert a row that would satisfy this condition until Transaction A completes."*

This is called a **Predicate Lock** (PostgreSQL implements this via its Serializable Snapshot Isolation algorithm). It is why Serializable is significantly more expensive — instead of managing locks on a small set of known rows, the database must track and enforce the boundaries of every query condition across the entire session.

### 4. Durability ("Written in Stone")
Once the database replies indicating the transaction was a "Success," that data is permanently secured. Using Write-Ahead Logging (WAL) on physical drives, Durability ensures that even if the server crashes abruptly milliseconds later, the data will still exist identically upon reboot.

---

## SQL vs. NoSQL (DynamoDB)

While traditional SQL databases are famous for built-in ACID guarantees, modern NoSQL databases (like AWS DynamoDB) have evolved to incorporate similar transactional capabilities, making the choice between the two more nuanced.

### When to choose SQL:
- The data is highly relational requiring complex `JOIN` statements.
- Strict schema enforcement at the database level is heavily preferred.
- Standard cross-table analytics and reporting are required out-of-the-box.
- Ideal for: *Core accounting, user accounts, heavy financial ledgers.*

### When to choose NoSQL (DynamoDB):
- The data schema needs to be highly fluid (e.g., dynamic product attributes maps where enforcing columns is impossible).
- True infinite scaling and absolutely flat latency is the primary goal regardless of size.
- Flexible pricing schemas (Serverless / Pay-Per-Request) are desired.
- Implementing an event-driven architecture (utilizing DynamoDB Streams).
- Ideal for: *Product catalogs, user session stores, shopping carts, event sourcing.*

---

## Concurrency Control: Pessimistic vs. Optimistic Locking

When multiple transactions try to read or write the same data simultaneously, the database (or application) needs a strategy to prevent conflicts. There are two fundamental philosophies: **Pessimistic** and **Optimistic** locking.

The core difference is rooted in a single assumption about your system:
- **Pessimistic:** *"Conflicts will happen often. Let's prevent them upfront."*
- **Optimistic:** *"Conflicts are rare. Let's allow work to proceed freely and only check at the end."*

---

### Pessimistic Locking

Pessimistic locking assumes that conflicts are likely, so it **prevents** concurrent access by acquiring a lock before doing any work. Other transactions that want the same data are forced to wait in a queue until the lock is released.

This is the mechanism SQL databases use natively via `SELECT ... FOR UPDATE`.

**How it works:**
```
Transaction A                         Transaction B
──────────────────────────────────────────────────────────
BEGIN;
SELECT stock FROM products
WHERE sku = 'LAPTOP-01'
FOR UPDATE;               ← Acquires row lock
-- stock = 1

                                      BEGIN;
                                      SELECT stock FROM products
                                      WHERE sku = 'LAPTOP-01'
                                      FOR UPDATE;
                                      -- ⏳ BLOCKED. Waiting for A's lock.

UPDATE products SET stock = 0
WHERE sku = 'LAPTOP-01';
COMMIT;                   ← Releases lock

                                      -- ✅ Lock acquired. Reads stock = 0.
                                      -- App logic: out of stock, abort.
                                      ROLLBACK;
```

**Pros:**
- Absolute safety — impossible for two transactions to corrupt the same data simultaneously.
- Simple to reason about — the first transaction always wins cleanly.

**Cons:**
- **Throughput bottleneck** — all other transactions queue up and wait, severely limiting concurrency under heavy load.
- **Deadlock risk** — if Transaction A locks row X and waits for row Y, while Transaction B locks row Y and waits for row X, the system freezes. Databases detect this and forcibly abort one transaction, but it adds complexity.
- **Poor fit for distributed systems** — acquiring a meaningful lock across 100 distributed nodes is impractical.

**Best used for:** High-contention scenarios where correctness is non-negotiable and write conflicts are genuinely frequent (e.g., a single bank account being debited by multiple concurrent payment processors).

---

### Optimistic Locking

Optimistic locking assumes that conflicts are rare. Instead of blocking, it **lets all transactions proceed freely** and only checks for conflicts at commit time. If a conflict is detected, one transaction is aborted and the caller must retry.

This is done by tracking a **version number** (or timestamp) on each row. A transaction reads the version along with the data, does its work, and at commit time verifies the version is still the same. If another transaction changed the row in the meantime, the version will have incremented and the commit fails.

**How it works (SQL with a `version` column):**
```sql
-- Step 1: Read the row and capture its version
SELECT stock, version FROM products WHERE sku = 'LAPTOP-01';
-- Returns: stock = 1, version = 42

-- Step 2: Do application logic...

-- Step 3: Commit, but ONLY if the version hasn't changed
UPDATE products
SET stock = 0, version = version + 1
WHERE sku = 'LAPTOP-01' AND version = 42;
-- If another transaction changed version to 43, this affects 0 rows → conflict detected → retry.
```

**How DynamoDB implements it natively** using `ConditionExpression`:
```go
_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
    TableName: aws.String("Products"),
    Key: map[string]types.AttributeValue{
        "sku": &types.AttributeValueMemberS{Value: "LAPTOP-01"},
    },
    UpdateExpression: aws.String("SET stock = :newStock, version = :newVersion"),
    // Only update if the version we read is still current
    ConditionExpression: aws.String("version = :expectedVersion"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":newStock":        &types.AttributeValueMemberN{Value: "0"},
        ":newVersion":      &types.AttributeValueMemberN{Value: "43"},
        ":expectedVersion": &types.AttributeValueMemberN{Value: "42"},
    },
})
if err != nil {
    // ConditionalCheckFailedException → version mismatch → retry
}
```

**Pros:**
- **Much higher throughput** — no waiting. All transactions run in parallel and only the rare conflicting one gets retried.
- **Natural fit for distributed systems** — no cross-node lock coordination needed.
- **No deadlocks** — transactions never wait on each other, so deadlock is architecturally impossible.

**Cons:**
- **Retry complexity** — the application must handle retries gracefully on conflict, adding code complexity.
- **Poor under high contention** — if many transactions constantly compete for the same row (e.g., a single trending product's stock), the retry rate explodes and throughput degrades badly ("retry storm").
- **Wasted work** — a transaction might do significant computation only to be aborted at the final step.

**Best used for:** Low-to-moderate contention scenarios where conflicts are genuinely rare (e.g., updating a product's description, user profile edits, or anything spread across many independent keys).

---

### Side-by-Side Comparison

| Factor | Pessimistic Locking | Optimistic Locking |
|---|---|---|
| Core assumption | Conflicts are frequent | Conflicts are rare |
| Mechanism | Lock before reading | Version-check before committing |
| Concurrency | Low (transactions queue) | High (transactions run in parallel) |
| Deadlock risk | Yes | No |
| Retry logic needed | No | Yes |
| Best environment | Single-node SQL, high contention | Distributed NoSQL, low contention |
| DynamoDB support | Via `TransactWriteItems` | Via `ConditionExpression` (native) |
