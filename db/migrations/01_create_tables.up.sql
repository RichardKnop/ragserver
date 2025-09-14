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
  primary key ("file", "status")
) strict;

create table "screening_status" (
  "id" integer primary key not null,
  "name" text not null
) strict;
insert into "screening_status"("id", "name") values (1, "REQUESTED");
insert into "screening_status"("id", "name") values (2, "GENERATING");
insert into "screening_status"("id", "name") values (3, "COMPLETED");
insert into "screening_status"("id", "name") values (4, "FAILED");

create table "screening" (
  "id" text primary key, -- uuid stored as text
  "author" text not null references "principal"("id"),
  "status" integer not null references "screening_status"("id"),
  "created" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  "updated" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ'))
) strict;

create index "screening_idx" on "screening"("created");

create table "screening_status_evt" (
  "screening" text not null references "screening"("id"),
  "status" integer not null references "screening_status"("id"),
  "message" text,
  "created" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
  primary key ("screening", "status")
) strict;

create table "screening_file" (
  "screening" text not null references "screening"("id"),
  "file" text not null references "file"("id"),
  "order" integer not null,
  primary key ("screening", "file")
) strict;

create index "screening_file_screening_idx" on "screening_file"("screening");

create table "question_type" (
  "id" integer primary key not null,
  "name" text not null
) strict;
insert into "question_type"("id", "name") values (1, "TEXT");
insert into "question_type"("id", "name") values (2, "BOOLEAN");
insert into "question_type"("id", "name") values (3, "METRIC");

create table "question" (
  "id" text primary key, -- uuid stored as text
  "author" text not null references "principal"("id"),
  "type" integer not null references "question_type"("id"),
  "content" text not null,
  "screening" text not null references "screening"("id"),
  "order" integer not null,
  "created" text not null default (strftime('%Y-%m-%dT%H:%M:%fZ'))
) strict;

create index "question_screening_idx" on "question"("screening");
