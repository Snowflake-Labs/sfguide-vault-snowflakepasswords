package snowpasssample

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/database/dbplugin"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
	"github.com/hashicorp/vault/sdk/database/helper/credsutil"
	"github.com/hashicorp/vault/sdk/database/helper/dbutil"
	"github.com/hashicorp/vault/sdk/helper/dbtxn"
	"github.com/hashicorp/vault/sdk/helper/strutil"
	_ "github.com/snowflakedb/gosnowflake"
)

const (
	snowflakeSQLTypeName     = "snowflake"
	defaultSnowflakeRenewSQL = `
alter user {{name}} set DAYS_TO_EXPIRY = {{expiration}};
`
	defaultSnowflakeRotateRootCredentialsSQL = `
alter user {{name}} set PASSWORD = '{{password}}';
`
)

var (
	_ dbplugin.Database = &SnowflakeSQL{}
)

// New implements builtinplugins.BuiltinFactory
func New() (interface{}, error) {
	db := new()
	// Wrap the plugin with middleware to sanitize errors
	dbType := dbplugin.NewDatabaseErrorSanitizerMiddleware(db, db.SecretValues)
	return dbType, nil
}

func new() *SnowflakeSQL {
	connProducer := &connutil.SQLConnectionProducer{}
	connProducer.Type = snowflakeSQLTypeName

	credsProducer := &credsutil.SQLCredentialsProducer{
		DisplayNameLen: 8,
		RoleNameLen:    8,
		UsernameLen:    63,
		Separator:      "_",
	}

	db := &SnowflakeSQL{
		SQLConnectionProducer: connProducer,
		CredentialsProducer:   credsProducer,
	}

	return db
}

// Run instantiates a SnowflakeSQL object, and runs the RPC server for the plugin
func Run(apiTLSConfig *api.TLSConfig) error {
	dbType, err := New()
	if err != nil {
		return err
	}

	dbplugin.Serve(dbType.(dbplugin.Database), api.VaultPluginTLSProvider(apiTLSConfig))

	return nil
}

type SnowflakeSQL struct {
	*connutil.SQLConnectionProducer
	credsutil.CredentialsProducer
}

func (s *SnowflakeSQL) Type() (string, error) {
	return snowflakeSQLTypeName, nil
}

func (s *SnowflakeSQL) getConnection(ctx context.Context) (*sql.DB, error) {
	db, err := s.Connection(ctx)
	if err != nil {
		return nil, err
	}

	return db.(*sql.DB), nil
}

// SetCredentials uses provided information to set/create a user in the
// database. Used by /database/static-roles/:name call from vault.
// Unlike CreateUser, this method requires a username be provided and
// uses the name given, instead of generating a name. This is used for creating
// and setting the password of static accounts, as well as rolling back
// passwords in the database in the event an updated database fails to save in
// Vault's storage. In Snowflake, the user must be owned by USERADMIN role (or
// whatever role vault is using for it's authority) for this to work
func (s *SnowflakeSQL) SetCredentials(ctx context.Context, statements dbplugin.Statements, staticUser dbplugin.StaticUserConfig) (username, password string, err error) {
	if len(statements.Rotation) == 0 {
		statements.Rotation = []string{defaultSnowflakeRotateRootCredentialsSQL}
	}

	username = staticUser.Username
	password = staticUser.Password
	if username == "" || password == "" {
		return "", "", errors.New("must provide both username and password")
	}

	// Get the connection
	db, err := s.getConnection(ctx)
	if err != nil {
		return "", "", err
	}

	// Vault requires the database user already exist, and that the credentials
	// used to execute the rotation statements has sufficient privileges.
	stmts := statements.Rotation

	// Execute each query
	for _, stmt := range stmts {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"name":     staticUser.Username,
				"username": staticUser.Username,
				"password": password,
			}
			if err := dbtxn.ExecuteDBQuery(ctx, db, m, query); err != nil {
				return "", "", err
			}
		}
	}

	return username, password, nil
}

func (s *SnowflakeSQL) CreateUser(ctx context.Context, statements dbplugin.Statements, usernameConfig dbplugin.UsernameConfig, expiration time.Time) (username string, password string, err error) {
	statements = dbutil.StatementCompatibilityHelper(statements)

	if len(statements.Creation) == 0 {
		return "", "", dbutil.ErrEmptyCreationStatement
	}

	username, err = s.GenerateUsername(usernameConfig)
	if err != nil {
		return "", "", err
	}

	password, err = s.GeneratePassword()
	if err != nil {
		return "", "", err
	}

	expirationStr, err := calculateExpirationString(expiration)
	if err != nil {
		return "", "", err
	}

	// Get the connection
	db, err := s.getConnection(ctx)
	if err != nil {
		return "", "", err
	}

	// Execute each query
	for _, stmt := range statements.Creation {
		// it's fine to split the statements on the semicolon.
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"name":       username,
				"username":   username,
				"password":   password,
				"expiration": expirationStr,
			}
			if err := dbtxn.ExecuteDBQuery(ctx, db, m, query); err != nil {
				return "", "", err
			}
		}
	}

	return username, password, nil
}

func (s *SnowflakeSQL) RenewUser(ctx context.Context, statements dbplugin.Statements, username string, expiration time.Time) error {
	statements = dbutil.StatementCompatibilityHelper(statements)

	renewStmts := statements.Renewal
	if len(renewStmts) == 0 {
		renewStmts = []string{defaultSnowflakeRenewSQL}
	}

	db, err := s.getConnection(ctx)
	if err != nil {
		return err
	}

	expirationStr, err := calculateExpirationString(expiration)
	if err != nil {
		return err
	}

	for _, stmt := range renewStmts {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"name":       username,
				"username":   username,
				"expiration": expirationStr,
			}
			if err := dbtxn.ExecuteDBQuery(ctx, db, m, query); err != nil {
				return err
			}
		}
	}

	return err
}

func (s *SnowflakeSQL) RevokeUser(ctx context.Context, statements dbplugin.Statements, username string) error {
	statements = dbutil.StatementCompatibilityHelper(statements)

	if len(statements.Revocation) == 0 {
		return s.defaultRevokeUser(ctx, username)
	}

	return s.customRevokeUser(ctx, username, statements.Revocation)
}

func (s *SnowflakeSQL) customRevokeUser(ctx context.Context, username string, revocationStmts []string) error {
	db, err := s.getConnection(ctx)
	if err != nil {
		return err
	}

	for _, stmt := range revocationStmts {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}

			m := map[string]string{
				"name":     username,
				"username": username,
			}
			if err := dbtxn.ExecuteDBQuery(ctx, db, m, query); err != nil {
				return err
			}
		}
	}

	return err
}

func (s *SnowflakeSQL) defaultRevokeUser(ctx context.Context, username string) error {
	db, err := s.getConnection(ctx)
	if err != nil {
		return err
	}

	// Drop this user
	stmt, err := db.PrepareContext(ctx, fmt.Sprintf(
		`drop user %s;`, strings.ToUpper(username)))
	if err != nil {
		errString := err.Error()

		// the 002003 (02000) error means the user isn't there. something may
		// have already dropped it. that's fine. but this is a bit brittle as
		// the error may change at some point and this would need updating
		if !(strings.Contains(errString, "002003 (02000)")) {
			return err
		}
	}

	defer stmt.Close()
	if _, err := stmt.ExecContext(ctx); err != nil {
		return err
	}

	return nil
}

func (s *SnowflakeSQL) RotateRootCredentials(ctx context.Context, statements []string) (map[string]interface{}, error) {
	if len(s.Username) == 0 || len(s.Password) == 0 {
		return nil, errors.New("username and password are required to rotate")
	}

	rotateStatements := statements
	if len(rotateStatements) == 0 {
		rotateStatements = []string{defaultSnowflakeRotateRootCredentialsSQL}
	}

	db, err := s.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	password, err := s.GeneratePassword()
	if err != nil {
		return nil, err
	}

	for _, stmt := range rotateStatements {
		for _, query := range strutil.ParseArbitraryStringSlice(stmt, ";") {
			query = strings.TrimSpace(query)
			if len(query) == 0 {
				continue
			}
			m := map[string]string{
				"name":     s.Username,
				"username": s.Username,
				"password": password,
			}
			if err := dbtxn.ExecuteDBQuery(ctx, db, m, query); err != nil {
				return nil, err
			}
		}
	}

	// Close the database connection to ensure no new connections come in
	if err := db.Close(); err != nil {
		return nil, err
	}

	s.RawConfig["password"] = password
	return s.RawConfig, nil
}

func calculateExpirationString(expiration time.Time) (string, error) {
	// create time.Time object
	currentTime := time.Now()

	// get the diff
	expirationTime := expiration
	timeDiff := expirationTime.Sub(currentTime)

	// since it's going to be a string, just make it one
	diffStr := timeDiff.String()

	// split into pieces
	pieces := strings.SplitAfterN(diffStr, "h", 2)

	// extract the hours
	expirationStr := strings.ReplaceAll(pieces[0], "h", "")

	// translate the expiration into whole days from hours
	// SEEMS TO BE LAST THING FROM GETTING THIS WORKING
	// TAKING OUT FOR NOW TO SEE...
	// expirationStr = "1" // for now
	if expirationInt, err := strconv.Atoi(expirationStr); err == nil {
		// this number will be in hours
		if (expirationInt / 24) < 1 {
			// anything less than 24 hours becomes 1 day
			expirationStr = "1"
		} else {
			// anything more than 24 hours is rounded down to least possible days
			expirationStr = strconv.Itoa(int(math.Floor(float64(expirationInt / 24))))
		}

		return expirationStr, nil
	} else {
		return "", err
	}
}
