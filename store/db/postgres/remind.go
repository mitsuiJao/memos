package postgres

import (
	"context"
	"strings"

	"github.com/usememos/memos/store"
)

func (d *DB) CreateRemind(ctx context.Context, create *store.Remind) (*store.Remind, error) {
	stmt := `
		INSERT INTO remind (creator_id, memo_uid, sequence_number, description, scheduled_ts)
		VALUES ($1, $2, (SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM remind WHERE creator_id = $1), $3, $4)
		RETURNING id, sequence_number`
	if err := d.db.QueryRowContext(ctx, stmt,
		create.CreatorID,
		create.MemoUID,
		create.Description,
		create.ScheduledTs,
	).Scan(&create.ID, &create.SequenceNumber); err != nil {
		return nil, err
	}
	return create, nil
}

func (d *DB) ListReminds(ctx context.Context, find *store.FindRemind) ([]*store.Remind, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ID != nil {
		where, args = append(where, "id = "+placeholder(len(args)+1)), append(args, *find.ID)
	}
	if find.CreatorID != nil {
		where, args = append(where, "creator_id = "+placeholder(len(args)+1)), append(args, *find.CreatorID)
	}
	if find.SequenceNumber != nil {
		where, args = append(where, "sequence_number = "+placeholder(len(args)+1)), append(args, *find.SequenceNumber)
	}
	if find.MemoUID != nil {
		where, args = append(where, "memo_uid = "+placeholder(len(args)+1)), append(args, *find.MemoUID)
	}
	if find.DueOnly {
		where, args = append(where, "fired = 0 AND deleted = 0 AND scheduled_ts <= "+placeholder(len(args)+1)), append(args, find.NowTs)
	} else if find.ActiveOnly {
		where = append(where, "fired = 0 AND deleted = 0")
	}

	rows, err := d.db.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			memo_uid,
			sequence_number,
			description,
			scheduled_ts,
			fired,
			deleted
		FROM remind
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY sequence_number ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*store.Remind{}
	for rows.Next() {
		r := &store.Remind{}
		var fired, deleted int
		if err := rows.Scan(
			&r.ID,
			&r.CreatorID,
			&r.MemoUID,
			&r.SequenceNumber,
			&r.Description,
			&r.ScheduledTs,
			&fired,
			&deleted,
		); err != nil {
			return nil, err
		}
		r.Fired = fired != 0
		r.Deleted = deleted != 0
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (d *DB) UpdateRemind(ctx context.Context, update *store.UpdateRemind) error {
	set, args := []string{}, []any{}

	if update.Fired != nil {
		v := 0
		if *update.Fired {
			v = 1
		}
		set, args = append(set, "fired = "+placeholder(len(args)+1)), append(args, v)
	}
	if update.Deleted != nil {
		v := 0
		if *update.Deleted {
			v = 1
		}
		set, args = append(set, "deleted = "+placeholder(len(args)+1)), append(args, v)
	}
	if len(set) == 0 {
		return nil
	}

	args = append(args, update.ID)
	if _, err := d.db.ExecContext(ctx, "UPDATE remind SET "+strings.Join(set, ", ")+" WHERE id = "+placeholder(len(args)), args...); err != nil {
		return err
	}
	return nil
}

func (d *DB) DeleteRemind(ctx context.Context, delete *store.DeleteRemind) error {
	if _, err := d.db.ExecContext(ctx, "DELETE FROM remind WHERE id = $1", delete.ID); err != nil {
		return err
	}
	return nil
}

func (d *DB) CreateRemindEphemeralMemo(ctx context.Context, create *store.RemindEphemeralMemo) error {
	stmt := "INSERT INTO remind_ephemeral_memo (memo_uid, expire_ts) VALUES ($1, $2) ON CONFLICT (memo_uid) DO UPDATE SET expire_ts = EXCLUDED.expire_ts"
	if _, err := d.db.ExecContext(ctx, stmt, create.MemoUID, create.ExpireTs); err != nil {
		return err
	}
	return nil
}

func (d *DB) ListRemindEphemeralMemos(ctx context.Context, find *store.FindRemindEphemeralMemo) ([]*store.RemindEphemeralMemo, error) {
	where, args := []string{"1 = 1"}, []any{}

	if find.ExpiredBefore != 0 {
		where, args = append(where, "expire_ts <= "+placeholder(len(args)+1)), append(args, find.ExpiredBefore)
	}

	rows, err := d.db.QueryContext(ctx,
		"SELECT memo_uid, expire_ts FROM remind_ephemeral_memo WHERE "+strings.Join(where, " AND ")+" ORDER BY expire_ts ASC",
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []*store.RemindEphemeralMemo{}
	for rows.Next() {
		e := &store.RemindEphemeralMemo{}
		if err := rows.Scan(&e.MemoUID, &e.ExpireTs); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (d *DB) DeleteRemindEphemeralMemo(ctx context.Context, memoUID string) error {
	if _, err := d.db.ExecContext(ctx, "DELETE FROM remind_ephemeral_memo WHERE memo_uid = $1", memoUID); err != nil {
		return err
	}
	return nil
}

