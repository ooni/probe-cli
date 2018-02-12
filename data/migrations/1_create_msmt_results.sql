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
    `startTime` DATETIME,
    `endTime` DATETIME,
    `summary` JSON,
    `done` TINYINT(1),
    `dataUsageUp` INTEGER,
    `dataUsageDown` INTEGER,
    `createdAt` DATETIME NOT NULL,
    `updatedAt` DATETIME NOT NULL
);

CREATE TABLE `measurements` (
    `id` INTEGER PRIMARY KEY AUTOINCREMENT,
    `name` VARCHAR(255),
    `startTime` DATETIME,
    `endTime` DATETIME,
    `summary` JSON,
    `ip` VARCHAR(255),
    `asn` INTEGER,
    `country` VARCHAR(2),
    `networkName` VARCHAR(255),
    `state` TEXT,
    `failure` VARCHAR(255),
    `reportFile` VARCHAR(255),
    `reportId` VARCHAR(255),
    `input` VARCHAR(255),
    `measurementId` VARCHAR(255),
    `createdAt` DATETIME NOT NULL,
    `updatedAt` DATETIME NOT NULL,
    `resultId` INTEGER REFERENCES `results` (`id`) ON DELETE SET NULL ON UPDATE CASCADE
);

-- +migrate StatementEnd
