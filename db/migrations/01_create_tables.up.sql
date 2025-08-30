create table "principal" (
  "id" text primary key, -- uuid stored as text
  "name" text,
  "created" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  "updated" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ'))
) strict;

create table "file_status" (
  "id" integer primary key not null,
  "name" text not null
) strict;
insert into "file_status"("id", "name") values (1, "UPLOADED");
insert into "file_status"("id", "name") values (2, "PROCESSING");
insert into "file_status"("id", "name") values (3, "PROCESSED_SUCCESSFULLY");
insert into "file_status"("id", "name") values (4, "PROCESSING_FAILED");

create table "file" (
  "id" text primary key, -- uuid stored as text
  "author" text not null references "principal"("id"),
  "file_name" text not null,
  "content_type" text not null,
  "extension" text not null,
  "file_size" integer not null,
  "file_hash" text not null,
  "embedder" text not null,
  "retriever" text not null,
  "location" text not null,
  "status" integer not null references "file_status"("id"),
  "created" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  "updated" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ'))
) strict;

create index "file_idx" on "file"("created");

create table "file_status_evt" (
  "file" text not null references "file"("id"),
  "status" integer not null references "file_status"("id"),
  "message" text,
  "created" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  PRIMARY KEY ("file", "status")
) strict;
