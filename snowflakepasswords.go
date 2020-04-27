package snowflakepasswords

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

// QUESTIONS
// get expiration logic figured out
// USERADMIN and days for expiry, or SECURITYADMIN to use NETWORK_POLICY hack?
/////// can't use the MINS_TO_BYPASS_NETWORK_POLICY because it's a system
/////// level setting not even accountadmin can change
// ADD IN HANDLING FOR USER DOES NOT EXIT ERRORS IN CASE SOMEONE ELSE DELETES USER BEFORE VAULT GETS AROUND TO IT

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
// database. Unlike CreateUser, this method requires a username be provided and
// uses the name given, instead of generating a name. This is used for creating
// and setting the password of static accounts, as well as rolling back
// passwords in the database in the event an updated database fails to save in
// Vault's storage.
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

	// Check if the user exists - NAME MUST BE CAPS
	// This syntax also means we must have an extra grant for the role:
	// grant imported privileges on database snowflake to role USERADMIN;
	var exists bool
	err = db.QueryRowContext(ctx, "SELECT exists (select name from SNOWFLAKE.ACCOUNT_USAGE.USERS where name = '$1');", strings.ToUpper(username)).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
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

	// since postgres seems to take a date for expiry, this works
	// we need a simple number of days
	/* expirationStr, err := s.GenerateExpiration(expiration)
	if err != nil {
		return "", "", err
	} */

	// create time.Time object
	currentTime := time.Now()

	// get the diff
	timeDiff := expiration.Sub(currentTime)

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

	// since postgres seems to take a date for expiry, this works
	// we need a simple number of days
	/* expirationStr, err := s.GenerateExpiration(expiration)
	if err != nil {
		return "", "", err
	} */

	// create time.Time object
	currentTime := time.Now()

	// get the diff
	timeDiff := expiration.Sub(currentTime)

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

	// Check if the role exists - NAME MUST BE CAPS
	// This syntax also means we must have an extra grant for the role:
	// grant imported privileges on database snowflake to role USERADMIN;
	// SINCE THIS IS A SELECT IT NEEDS A WAREHOUSE TO RUN, AND THAT MEANS A
	// BIG CHANGE TO THE USERADMIN USER ACTING AS VAULT. SINCE THERE IS NO
	// NEED TO CHECK, REMOVING THIS LOGIC FOR NOW TO ANALYZE IF IT'S FINE
	// TO HAVE IT GONE. SUSEPCT IT WILL BE FINE.
	/* var exists bool
	err = db.QueryRowContext(ctx, "SELECT exists (select name from SNOWFLAKE.ACCOUNT_USAGE.USERS where name = '$1');", strings.ToUpper(username)).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if !exists {
		return nil
	} */

	// Drop this user
	stmt, err := db.PrepareContext(ctx, fmt.Sprintf(
		`drop user %s;`, strings.ToUpper(username)))
	if err != nil {
		// DO THIS -- ADD IN HANDLING FOR USER DOES NOT EXIT ERRORS IN CASE SOMEONE
		// ELSE DELETES USER BEFORE VAULT GETS AROUND TO IT
		return err
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
