-- +migrate Down
-- +migrate StatementBegin

PRAGMA foreign_keys=off;
ALTER TABLE `measurements` RENAME TO `_measurements_new`;

CREATE TABLE `measurements` (
    `measurement_id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `test_name` VARCHAR(64) NOT NULL,
    `measurement_start_time` DATETIME NOT NULL,
    `measurement_runtime` REAL NOT NULL,

    `measurement_is_done` TINYINT(1) NOT NULL,
    `measurement_is_uploaded` TINYINT(1) NOT NULL,
    `measurement_is_failed` TINYINT(1) NOT NULL,
    `measurement_failure_msg` VARCHAR(255),
    `measurement_is_upload_failed` TINYINT(1) NOT NULL,
    `measurement_upload_failure_msg` VARCHAR(255),
    `measurement_is_rerun` TINYINT(1) NOT NULL,
    `report_id` VARCHAR(255),
    `url_id` INTEGER,
    `collector_measurement_id` INT(64),
    `is_anomaly` TINYINT(1),
    `test_keys` JSON NOT NULL,
    `result_id` INTEGER NOT NULL,
    `report_file_path` VARCHAR(260) NOT NULL,
    CONSTRAINT `fk_result_id`
      FOREIGN KEY (`result_id`)
      REFERENCES `results`(`result_id`)
      ON DELETE CASCADE,
    FOREIGN KEY (`url_id`) REFERENCES `urls`(`url_id`)
);

INSERT INTO measurements (
`measurement_id`,
`test_name`,
`measurement_start_time`,
`measurement_runtime`,
`measurement_is_done`,
`measurement_is_uploaded`,
`measurement_is_failed`,
`measurement_failure_msg`,
`measurement_is_upload_failed`,
`measurement_upload_failure_msg`,
`measurement_is_rerun`,
`report_id`,
`url_id`,
`collector_measurement_id`,
`is_anomaly`,
`test_keys`,
`result_id`,
`report_file_path`
)
  SELECT `measurement_id`,
`test_name`,
`measurement_start_time`,
`measurement_runtime`,
`measurement_is_done`,
`measurement_is_uploaded`,
`measurement_is_failed`,
`measurement_failure_msg`,
`measurement_is_upload_failed`,
`measurement_upload_failure_msg`,
`measurement_is_rerun`,
`report_id`,
`url_id`,
`collector_measurement_id`,
`is_anomaly`,
`test_keys`,
`result_id`,
`report_file_path`
  FROM _measurements_new;

DROP TABLE _measurements_new;

PRAGMA foreign_keys=on;

-- +migrate StatementEnd

-- +migrate Up
-- +migrate StatementBegin

PRAGMA foreign_keys=off;

-- SQLite3 does not support adding columns or dropping constraints, so we need
-- to re-create the table and copy the data over.

ALTER TABLE `measurements` RENAME TO `_measurements_old`;

CREATE TABLE `measurements` (
    `measurement_id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `test_name` VARCHAR(64) NOT NULL,
    `measurement_start_time` DATETIME NOT NULL,
    `measurement_runtime` REAL NOT NULL,

    `measurement_is_done` TINYINT(1) NOT NULL,
    `measurement_is_uploaded` TINYINT(1) NOT NULL,
    `measurement_is_failed` TINYINT(1) NOT NULL,
    `measurement_failure_msg` VARCHAR(255),
    `measurement_is_upload_failed` TINYINT(1) NOT NULL,
    `measurement_upload_failure_msg` VARCHAR(255),
    `measurement_is_rerun` TINYINT(1) NOT NULL,
    `report_id` VARCHAR(255),
    `url_id` INTEGER,
    `collector_measurement_id` INT(64),
    `is_anomaly` TINYINT(1),
    `test_keys` JSON NOT NULL,
    `result_id` INTEGER NOT NULL,
    `report_file_path` VARCHAR(260),
    `measurement_file_path` TEXT,
    CONSTRAINT `fk_result_id`
      FOREIGN KEY (`result_id`)
      REFERENCES `results`(`result_id`)
      ON DELETE CASCADE,
    FOREIGN KEY (`url_id`) REFERENCES `urls`(`url_id`)
);

INSERT INTO measurements (
`measurement_id`,
`test_name`,
`measurement_start_time`,
`measurement_runtime`,
`measurement_is_done`,
`measurement_is_uploaded`,
`measurement_is_failed`,
`measurement_failure_msg`,
`measurement_is_upload_failed`,
`measurement_upload_failure_msg`,
`measurement_is_rerun`,
`report_id`,
`url_id`,
`collector_measurement_id`,
`is_anomaly`,
`test_keys`,
`result_id`,
`report_file_path`,
`measurement_file_path`
)
  SELECT `measurement_id`,
`test_name`,
`measurement_start_time`,
`measurement_runtime`,
`measurement_is_done`,
`measurement_is_uploaded`,
`measurement_is_failed`,
`measurement_failure_msg`,
`measurement_is_upload_failed`,
`measurement_upload_failure_msg`,
`measurement_is_rerun`,
`report_id`,
`url_id`,
`collector_measurement_id`,
`is_anomaly`,
`test_keys`,
`result_id`,
`report_file_path`,
NULL
  FROM _measurements_old;

DROP TABLE _measurements_old;

PRAGMA foreign_keys=on;

-- +migrate StatementEnd
