begin;

drop table if exists "ragserver"."answer";
drop table if exists "ragserver"."question";
drop table if exists "ragserver"."screening_file";
drop table if exists "ragserver"."screening_status_evt";
drop table if exists "ragserver"."screening";

drop table if exists "ragserver"."file_status_evt";
drop table if exists "ragserver"."file";

drop table if exists "ragserver"."question_type";
drop table if exists "ragserver"."screening_status";
drop table if exists "ragserver"."file_status";

drop table if exists "ragserver"."principal";

drop schema if exists "ragserver";

commit;