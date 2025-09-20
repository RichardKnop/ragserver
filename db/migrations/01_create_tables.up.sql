begin;

create schema if not exists "ragserver";

create table "ragserver"."principal" (
  "id" uuid primary key,
  "name" text,
  "created" timestamp not null default now(),
  "updated" timestamp not null default now()
);

create table "ragserver"."file_status" (
  "id" serial primary key,
  "name" text not null
);
insert into "ragserver"."file_status"("id", "name") values (1, 'UPLOADED');
insert into "ragserver"."file_status"("id", "name") values (2, 'PROCESSING');
insert into "ragserver"."file_status"("id", "name") values (3, 'PROCESSED_SUCCESSFULLY');
insert into "ragserver"."file_status"("id", "name") values (4, 'PROCESSING_FAILED');

create table "ragserver"."file" (
  "id" uuid primary key,
  "author" uuid not null references "ragserver"."principal"("id"),
  "file_name" text not null,
  "content_type" text not null,
  "extension" text not null,
  "file_size" bigint not null,
  "file_hash" text not null,
  "embedder" text not null,
  "retriever" text not null,
  "status" integer not null references "ragserver"."file_status"("id"),
  "created" timestamp not null default now(),
  "updated" timestamp not null default now()
);

create index "file_idx" on "ragserver"."file" using btree("created");

create table "ragserver"."file_status_evt" (
  "file" uuid not null references "ragserver"."file"("id"),
  "status" integer not null references "ragserver"."file_status"("id"),
  "message" text,
  "created" timestamp not null default now(),
  primary key ("file", "status")
);

create table "ragserver"."screening_status" (
  "id" serial primary key,
  "name" text not null
);
insert into "ragserver"."screening_status"("id", "name") values (1, 'REQUESTED');
insert into "ragserver"."screening_status"("id", "name") values (2, 'GENERATING');
insert into "ragserver"."screening_status"("id", "name") values (3, 'COMPLETED');
insert into "ragserver"."screening_status"("id", "name") values (4, 'FAILED');

create table "ragserver"."screening" (
  "id" uuid primary key,
  "author" uuid not null references "ragserver"."principal"("id"),
  "status" integer not null references "ragserver"."screening_status"("id"),
  "created" timestamp not null default now(),
  "updated" timestamp not null default now()
);

create index "screening_idx" on "ragserver"."screening" using btree("created");

create table "ragserver"."screening_status_evt" (
  "screening" uuid not null references "screening"("id"),
  "status" integer not null references "screening_status"("id"),
  "message" text,
  "created" timestamp not null default now(),
  primary key ("screening", "status")
);

create table "ragserver"."screening_file" (
  "screening" uuid not null references "ragserver"."screening"("id"),
  "file" uuid not null references "ragserver"."file"("id"),
  "order" integer not null,
  primary key ("screening", "file")
);

create index "screening_file_screening_idx" on "ragserver"."screening_file" using hash("screening");

create table "ragserver"."question_type" (
  "id" serial primary key,
  "name" text not null
);
insert into "ragserver"."question_type"("id", "name") values (1, 'TEXT');
insert into "ragserver"."question_type"("id", "name") values (2, 'BOOLEAN');
insert into "ragserver"."question_type"("id", "name") values (3, 'METRIC');

create table "ragserver"."question" (
  "id" uuid primary key,
  "author" uuid not null references "ragserver"."principal"("id"),
  "type" integer not null references "ragserver"."question_type"("id"),
  "content" text not null,
  "screening" uuid not null references "ragserver"."screening"("id"),
  "order" integer not null,
  "created" timestamp not null default now(),
  "answered" text
);

create index "question_screening_idx" on "ragserver"."question" using hash("screening");

create table "ragserver"."answer" (
  "question" uuid not null references "ragserver"."question"("id"),
  "response" jsonb not null,
  "created" timestamp not null default now()
);

create index "answer_question_idx" on "ragserver"."answer" using hash("question");

commit;