package db

import (
	"database/sql"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

// FactionNone is reported while the account has not chosen a faction yet.
const FactionNone = 2

// SaveCountry runs PS_GameData.dbo.usp_Save_Char_Country_E, inserting the
// UserMaxGrow row the first time an account picks a faction.
func SaveCountry(userUID int32, country uint8) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	_, err = pool.ExecContext(c, `INSERT INTO PS_GameData_UserMaxGrow(ServerID,UserUID,Country,MaxGrow) VALUES(@serverid,@useruid,@country,0) ON CONFLICT(ServerID,UserUID) DO UPDATE SET Country=excluded.Country`,
		sql.Named("serverid", config.ServerID), sql.Named("useruid", userUID),
		sql.Named("country", int16(country)))
	return err
}
