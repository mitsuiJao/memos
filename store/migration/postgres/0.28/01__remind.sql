-- remind: stores reminders registered via #remind tag
CREATE TABLE remind (
  id              SERIAL  PRIMARY KEY,
  creator_id      INTEGER NOT NULL,
  memo_uid        TEXT    NOT NULL,
  sequence_number INTEGER NOT NULL,
  description     TEXT    NOT NULL DEFAULT '',
  scheduled_ts    BIGINT  NOT NULL,
  fired           INTEGER NOT NULL DEFAULT 0,
  deleted         INTEGER NOT NULL DEFAULT 0,
  UNIQUE(creator_id, sequence_number)
);

CREATE INDEX idx_remind_creator_id ON remind(creator_id);
CREATE INDEX idx_remind_scheduled_ts ON remind(scheduled_ts);

-- remind_ephemeral_memo: tracks bot-created response memos for auto-deletion after 30 min
CREATE TABLE remind_ephemeral_memo (
  memo_uid  TEXT   NOT NULL PRIMARY KEY,
  expire_ts BIGINT NOT NULL
);
