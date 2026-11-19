package db

// ServerNotice is a configured global announcement.
type ServerNotice struct {
	Index     uint32
	Message   string
	Frequency uint32
	Fixed     [5]string
}

func ServerNotices() ([]ServerNotice, error) {
	pool, err := Pool(GameDefs)
	if err != nil {
		return nil, err
	}
	ctx, cancel := longCtx()
	defer cancel()
	rows, err := pool.QueryContext(ctx, `SELECT "Index", Message, Frequency,
		COALESCE(FixedHour1, ''), COALESCE(FixedHour2, ''), COALESCE(FixedHour3, ''),
		COALESCE(FixedHour4, ''), COALESCE(FixedHour5, '')
		FROM PS_GameDefs_ServerNotices ORDER BY "Index"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notices []ServerNotice
	for rows.Next() {
		var n ServerNotice
		if err := rows.Scan(&n.Index, &n.Message, &n.Frequency,
			&n.Fixed[0], &n.Fixed[1], &n.Fixed[2], &n.Fixed[3], &n.Fixed[4]); err != nil {
			return nil, err
		}
		notices = append(notices, n)
	}
	return notices, rows.Err()
}
