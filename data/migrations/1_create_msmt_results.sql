-- +migrate Down
-- +migrate StatementBegin

DROP TABLE `results`;
DROP TABLE `measurements`;
DROP TABLE `urls`;
DROP TABLE `networks`;

-- +migrate StatementEnd

-- +migrate Up
-- +migrate StatementBegin

CREATE TABLE `urls` (
  `url_id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `url` VARCHAR(255) NOT NULL, -- XXX is this long enough?
  `category_code` VARCHAR(5) NOT NULL, -- The citizenlab category code for the
                                       -- site. We use the string NONE to denote
                                       -- no known category code.

  `url_country_code` VARCHAR(2) NOT NULL -- The two letter country code which this
                                     -- URL belongs to
);

-- We create a separate table for networks for 2 reasons:
-- 1. For some of the views where need the total number of measured networks,
-- it's going to be much more efficient to just lookup the count of rows in this
-- table.
-- 2. (most important) We want to avoid duplicating a bunch of information that
-- is going to be common to several networks the user is on.
-- Example:
-- We may wish to add to this table the location from of the probe from the GPS
-- or add support for allowing the user to "correct" a misclassified measurement
-- or distinguishing between wifi and mobile.
CREATE TABLE `networks` (
  `network_id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `network_name` VARCHAR(255) NOT NULL, -- String name representing the network_name which by default is populated based
                               -- on the ASN.
                               -- We use a separate key to reference the rows in
                               -- this tables, because we may wish to "enrich"
                               -- this with more data in the future.
  `network_type` VARCHAR(16) NOT NULL, -- One of wifi, mobile

  `ip` VARCHAR(40) NOT NULL,  -- Stores a string representation of an ipv4 or ipv6 address.
                               -- The longest ip is an ipv6 address like:
                               -- 0000:0000:0000:0000:0000:0000:0000:0000,
                               -- which is 39 chars.
  `asn` INT(4) NOT NULL,
  `network_country_code` VARCHAR(2) NOT NULL -- The two letter country code
);

CREATE TABLE `results` (
    `result_id` INTEGER PRIMARY KEY AUTOINCREMENT,
    -- This can be one of "websites", "im", "performance", "middlebox".
    `test_group_name` VARCHAR(16) NOT NULL,
    -- We use a different start_time and runtime, because we want to also have
    -- data to measure the overhead of creating a report and other factors that
    -- go into the test.
    -- That is to say: `SUM(runtime) FROM measurements` will always be <=
    -- `runtime FROM results` (most times <)
    `result_start_time` DATETIME NOT NULL,
    `result_runtime` REAL,

    -- Used to indicate if the user has seen this result
    `result_is_viewed` TINYINT(1) NOT NULL,

    -- This is a flag used to indicate if the result is done or is currently running.
    `result_is_done` TINYINT(1) NOT NULL,
    `result_data_usage_up` REAL NOT NULL,
    `result_data_usage_down` REAL NOT NULL,
    -- It's probably reasonable to set the maximum length to 260 as this is the
    -- maximum length of file paths on windows.
    `measurement_dir` VARCHAR(260) NOT NULL,

    `network_id` INTEGER NOT NULL,
    CONSTRAINT `fk_network_id`
      FOREIGN KEY(`network_id`)
      REFERENCES `networks`(`network_id`)
);

CREATE TABLE `measurements` (
    `measurement_id` INTEGER PRIMARY KEY AUTOINCREMENT,
    -- This can be one of:
    -- facebook_messenger
    -- telegram
    -- whatsapp
    -- http_header_field_manipulation
    -- http_invalid_request_line
    -- dash
    -- ndt
    `test_name` VARCHAR(64) NOT NULL,
    `measurement_start_time` DATETIME NOT NULL,
    `measurement_runtime` REAL NOT NULL,

    -- Note for golang: we used to have state be one of `done` and `active`, so
    -- this is equivalent to done being true or false.
    -- `state` TEXT,
    `measurement_is_done` TINYINT(1) NOT NULL,
    -- The reason to have a dedicated is_uploaded flag, instead of just using
    -- is_upload_failed, is that we may not have uploaded the measurement due
    -- to a setting.
    `measurement_is_uploaded` TINYINT(1) NOT NULL,

    -- This is the measurement failed to run and the user should be offerred to
    -- re-run it.
    `measurement_is_failed` TINYINT(1) NOT NULL,
    `measurement_failure_msg` VARCHAR(255),

    `measurement_is_upload_failed` TINYINT(1) NOT NULL,
    `measurement_upload_failure_msg` VARCHAR(255),

    -- Is used to indicate that this particular measurement has been re-run and
    -- therefore the UI can take this into account to either hide it from the
    -- result view or at the very least disable the ability to re-run it.
    -- XXX do we also want to have a reference to the re-run measurement?
    `measurement_is_rerun` TINYINT(1) NOT NULL,

    -- This is the server-side report_id returned by the collector. By using
    -- report_id & input, you can query the api to fetch this measurement.
    -- Ex.
    -- GET https://api.ooni.io/api/v1/measurements?input=$INPUT&report_id=$REPORT_ID
    -- Extract the first item from the `result[]` list and then fetch:
    -- `measurement_url` to get the JSON of this measurement row.
    -- These two values (`report_id`, `input`) are useful to fetch a
    -- measurement that has already been processed by the pipeline, to
    -- implement cleanup of already uploaded measurements.
    `report_id` VARCHAR(255), -- This can be NULL when no report file has been
                              -- created.

    `url_id` INTEGER,

    -- This is not yet a feature of the collector, but we are planning to add
    -- this at some point in the near future.
    -- See: https://github.com/ooni/pipeline/blob/master/docs/ooni-uuid.md &
    -- https://github.com/ooni/pipeline/issues/48
    `collector_measurement_id` INT(64),

    -- This indicates in the case of a websites test, that a site is likely
    -- blocked, or for an IM test if the IM tests says the app is likely
    -- blocked, or if a middlebox was detected.
    -- You can `JOIN` a `COUNT()` of this value in the results view to get a count of
    -- blocked sites or blocked IM apps
    `is_anomaly` TINYINT(1),

    -- This is an opaque JSON structure, where we store some of the test_keys
    -- we need for the measurement details views and some result views (ex. the
    -- upload/download speed of NDT, the reason for blocking of a site,
    -- etc.)
    `test_keys` JSON NOT NULL,

    -- The cross table reference to JOIN the two tables together.
    `result_id` INTEGER NOT NULL,


    -- This is a variable used internally to track the path to the on-disk
    -- measurements.json. It may make sense to write one file per entry by
    -- hooking MK and preventing it from writing to a file on disk which may
    -- have many measurements per file.
    `report_file_path` VARCHAR(260) NOT NULL,

    CONSTRAINT `fk_result_id`
      FOREIGN KEY (`result_id`)
      REFERENCES `results`(`result_id`)
      ON DELETE CASCADE, -- If we delete a result we also want
                         -- all the measurements to be deleted as well.
    FOREIGN KEY (`url_id`) REFERENCES `urls`(`url_id`)
);
-- +migrate StatementEnd
