ALTER TABLE game_results
    DROP COLUMN IF EXISTS class_name;

ALTER TABLE participants
    DROP COLUMN IF EXISTS class_name;
