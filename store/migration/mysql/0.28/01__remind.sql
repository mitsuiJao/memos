-- remind: stores reminders registered via #remind tag
CREATE TABLE `remind` (
  `id`              INT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `creator_id`      INT          NOT NULL,
  `memo_uid`        VARCHAR(256) NOT NULL,
  `sequence_number` INT          NOT NULL,
  `description`     TEXT         NOT NULL,
  `scheduled_ts`    BIGINT       NOT NULL,
  `fired`           TINYINT      NOT NULL DEFAULT 0,
  `deleted`         TINYINT      NOT NULL DEFAULT 0,
  UNIQUE(`creator_id`, `sequence_number`)
);

CREATE INDEX `idx_remind_creator_id` ON `remind`(`creator_id`);
CREATE INDEX `idx_remind_scheduled_ts` ON `remind`(`scheduled_ts`);

-- remind_ephemeral_memo: tracks bot-created response memos for auto-deletion after 30 min
CREATE TABLE `remind_ephemeral_memo` (
  `memo_uid`  VARCHAR(256) NOT NULL PRIMARY KEY,
  `expire_ts` BIGINT       NOT NULL
);
