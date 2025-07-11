// File: schema.hcl

// Use a variable to represent the "public" schema in PostgreSQL.
schema "public" {}

table "emails" {
  // Associate this table with the public schema.
  schema = schema.public

  // Define the columns with PostgreSQL-specific types.
  column "id" {
    null = false
    type = serial // 'serial' is a PostgreSQL-specific auto-incrementing integer.
  }
  column "sender" {
    null = false
    type = text
  }
  column "recipient" {
    null = false
    type = text
  }
  column "subject" {
    null = false
    type = text
  }
  column "body" {
    null = false
    type = text
  }
  column "links" {
    null = false
    type = jsonb // Use 'jsonb' here as well.
  }
  column "received_at" {
    null    = false
    type    = timestamptz // Use timestamp with time zone.
    default = sql("CURRENT_TIMESTAMP")
  }
  // Define the primary key.
  primary_key {
    columns = [column.id]
  }
}
