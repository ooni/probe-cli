-- +migrate Down
-- +migrate StatementBegin

ALTER TABLE `results`
DROP COLUMN result_is_uploaded;

-- +migrate StatementEnd

-- +migrate Up
-- +migrate StatementBegin

ALTER TABLE `results`
ADD COLUMN result_is_uploaded TINYINT(1) DEFAULT 1 NOT NULL;

-- +migrate StatementEnd