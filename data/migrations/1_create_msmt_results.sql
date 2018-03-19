-- +migrate Down
-- +migrate StatementBegin

DROP TABLE `results`;
DROP TABLE `measurements`;

-- +migrate StatementEnd

-- +migrate Up
-- +migrate StatementBegin

CREATE TABLE `results` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `name` VARCHAR(255),
    `start_time` DATETIME,
    `end_time` DATETIME,
    `summary` JSON,
    `done` TINYINT(1),
    `data_usage_up` INTEGER,
    `data_usage_down` INTEGER
);

CREATE TABLE `measurements` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `name` VARCHAR(255),
    `start_time` DATETIME,
    `runtime` INTEGER,
    `summary` JSON,
    `ip` VARCHAR(255),
    `asn` VARCHAR(16),
    `country` VARCHAR(2),
    `network_name` VARCHAR(255),
    `state` TEXT,
    `failure` VARCHAR(255),
    `upload_failure` VARCHAR(255),
    `uploaded` TINYINT(1),
    `report_file` VARCHAR(255),
    `report_id` VARCHAR(255),
    `input` VARCHAR(255),
    `result_id` INTEGER REFERENCES `results` (`id`) ON DELETE SET NULL ON UPDATE CASCADE
);

-- +migrate StatementEnd
