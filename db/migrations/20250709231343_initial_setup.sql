-- Create "emails" table
CREATE TABLE "public"."emails" (
  "id" serial NOT NULL,
  "sender" text NOT NULL,
  "recipient" text NOT NULL,
  "subject" text NOT NULL,
  "body" text NOT NULL,
  "links" jsonb NOT NULL,
  "received_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id")
);
