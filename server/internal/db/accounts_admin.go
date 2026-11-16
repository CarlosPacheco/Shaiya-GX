package db

import (
	"database/sql"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/authwire"
	"github.com/Spelunkern/Shaiya-GX/server/internal/password"
)

// CreateAccount never stores plaintext and never replaces an existing account.
func CreateAccount(name, secret string, admin bool) (int32, error) {
	if err := authwire.Validate(name, secret); err != nil {
		return 0, err
	}
	hash, err := password.Hash(secret)
	if err != nil {
		return 0, err
	}
	pool, err := Pool(UserData)
	if err != nil {
		return 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var uid int32
	err = withTx(c, pool, func(tx *sql.Tx) error {
		var exists int
		if err := tx.QueryRowContext(c, `SELECT COUNT(*) FROM PS_UserData_Users_Master WHERE UserID=?`, name).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			return fmt.Errorf("account already exists")
		}
		var next int64
		if err := tx.QueryRowContext(c, `SELECT COALESCE(MAX(UserUID),0)+1 FROM PS_UserData_Users_Master`).Scan(&next); err != nil {
			return err
		}
		if next > 2147483647 {
			return fmt.Errorf("account identity space exhausted")
		}
		uid = int32(next)
		_, err := tx.ExecContext(c, `INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?,?,?,?,?,0,0,0,'N',0)`, uid, name, hash, admin, 0)
		return err
	})
	return uid, err
}

// SetAccountPassword revokes pending tickets and active sessions through a
// monotonically increasing credential version. World checks it every 30 seconds.
func SetAccountPassword(name, secret string) error {
	if err := authwire.Validate(name, secret); err != nil {
		return err
	}
	hash, err := password.Hash(secret)
	if err != nil {
		return err
	}
	pool, err := Pool(UserData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	return withTx(c, pool, func(tx *sql.Tx) error {
		var uid int32
		if err := tx.QueryRowContext(c, `SELECT UserUID FROM PS_UserData_Users_Master WHERE UserID=?`, name).Scan(&uid); err != nil {
			return err
		}
		if _, err := tx.ExecContext(c, `UPDATE PS_UserData_Users_Master SET Pw=? WHERE UserUID=?`, hash, uid); err != nil {
			return err
		}
		if _, err := tx.ExecContext(c, `INSERT INTO GX_AccountSecurity(UserUID,Version) VALUES(?,2) ON CONFLICT(UserUID) DO UPDATE SET Version=Version+1`, uid); err != nil {
			return err
		}
		_, err := tx.ExecContext(c, `DELETE FROM GX_SessionHandoff WHERE UserUID=?`, uid)
		return err
	})
}
