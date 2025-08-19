CREATE TABLE "file" (
  "id" TEXT PRIMARY KEY, -- UUID stored as text
  "file_name" TEXT NOT NULL,
  "mime_type" TEXT NOT NULL,
  "extension" TEXT NOT NULL,
  "file_size" INTEGER NOT NULL,
  "file_hash" TEXT NOT NULL,
  "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);
