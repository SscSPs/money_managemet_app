---
trigger: model_decision
description: when working on creating/updating/handling db structure and/or schema
---


## 🗃️ Database & Migrations

**Migration Files:**
```
migrations/
  000001_initial_schema.up.sql
  000001_initial_schema.down.sql
  000002_add_feature.up.sql
  000002_add_feature.down.sql
```

**Naming Convention:** `000XXX_description.{up|down}.sql`

**Example Migration:**
```sql
-- 000018_add_products.up.sql
CREATE TABLE products (
    product_id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(19, 4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(36) NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_by VARCHAR(36) NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_products_deleted_at ON products(deleted_at);
```

**Run migrations:** Automatically on startup with `make run`
