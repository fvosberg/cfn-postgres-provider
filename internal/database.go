package internal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

func open(cfg ConnectionConfig) (*conn, error) {
	// TODO enable sslmode?
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password)
	c, err := sql.Open("postgres", psqlInfo)
	return &conn{c, cfg.User}, err
}

type conn struct {
	*sql.DB
	superUser string
}

func (c *conn) CreateUser(ctx context.Context, userName, password string) error {
	stmt := fmt.Sprintf(`DO
$do$
BEGIN
   IF NOT EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = %s) THEN

      CREATE ROLE %s LOGIN PASSWORD %s NOCREATEDB;
   END IF;
END
$do$;`,
		pq.QuoteLiteral(userName),
		pq.QuoteIdentifier(userName),
		pq.QuoteLiteral(password),
	)

	_, err := c.ExecContext(ctx, stmt)
	return err
}

// TODO database exists, but the user doesn't have access
func (c *conn) CreateDBWithOwner(ctx context.Context, dbName, owner string) error {
	// in rds the creating user must be part of the role, the db will be owned by
	stmt := fmt.Sprintf(
		"GRANT %s TO %s",
		pq.QuoteIdentifier(owner),
		pq.QuoteIdentifier(c.superUser),
	)
	_, err := c.ExecContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf(
			"granting role %q to superuser %q failed: %w",
			owner,
			c.superUser,
			err,
		)
	}
	stmt = fmt.Sprintf(
		"CREATE DATABASE %s OWNER %s",
		pq.QuoteIdentifier(dbName),
		pq.QuoteIdentifier(owner),
	)
	_, err = c.ExecContext(ctx, stmt)

	// check if an "database already exists" error has been thrown
	// we have to do that, because CREATE DATABASE isn't allowed in
	// transactions
	if v, ok := err.(*pq.Error); ok && v.Code == "42P04" {
		return nil
	}
	return err
}

type dbConn struct {
	*sql.DB
}

func openDatabase(cfg ConnectionConfig, dbName string) (*dbConn, error) {
	// TODO enable sslmode?
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s sslmode=disable dbname=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbName)
	c, err := sql.Open("postgres", psqlInfo)
	return &dbConn{c}, err
}

func (c *dbConn) CreateExtensions(ctx context.Context, extensions ...string) error {
	for _, ext := range extensions {
		stmt := fmt.Sprintf(
			"create extension if not exists %s",
			pq.QuoteIdentifier(ext),
		)
		_, err := c.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf(
				"creating extension %q failed: %w",
				ext,
				err,
			)
		}
	}
	return nil
}
