package sdk

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Databases describes all the databases related methods that the
// Snowflake API supports.
type Databases interface {
	// List all the databases by pattern.
	List(ctx context.Context, options DatabaseListOptions) ([]*Database, error)
	// Create a new database with the given options.
	Create(ctx context.Context, options DatabaseCreateOptions) (*Database, error)
	// Read an database by its name.
	Read(ctx context.Context, database string) (*Database, error)
	// Update attributes of an existing database.
	Update(ctx context.Context, database string, options DatabaseUpdateOptions) (*Database, error)
	// Delete an database by its name.
	Delete(ctx context.Context, database string) error
}

// databases implements Databases
type databases struct {
	client *Client
}

// Database represents a Snowflake database.
type Database struct {
	Name          string
	IsDefault     string
	IsCurrent     string
	Origin        string
	Owner         string
	Comment       string
	Options       string
	RetentionTime string
	CreatedOn     time.Time
}

type databaseEntity struct {
	Name          sql.NullString `db:"name"`
	IsDefault     sql.NullString `db:"is_default"`
	IsCurrent     sql.NullString `db:"is_current"`
	Origin        sql.NullString `db:"origin"`
	Owner         sql.NullString `db:"owner"`
	Comment       sql.NullString `db:"comment"`
	Options       sql.NullString `db:"options"`
	RetentionTime sql.NullString `db:"retention_time"`
	CreatedOn     sql.NullTime   `db:"created_on"`
}

func (d *databaseEntity) toDatabase() *Database {
	return &Database{
		Name:          d.Name.String,
		IsDefault:     d.IsDefault.String,
		IsCurrent:     d.IsCurrent.String,
		Origin:        d.Origin.String,
		Owner:         d.Owner.String,
		Comment:       d.Comment.String,
		Options:       d.Options.String,
		RetentionTime: d.RetentionTime.String,
		CreatedOn:     d.CreatedOn.Time,
	}
}

// DatabaseListOptions represents the options for listing databases.
type DatabaseListOptions struct {
	Pattern string
}

func (o DatabaseListOptions) validate() error {
	if o.Pattern == "" {
		return errors.New("pattern must not be empty")
	}
	return nil
}

type DatabaseProperties struct {
	// Optional: Specifies a comment for the database.
	Comment *string
}

type DatabaseCreateOptions struct {
	*DatabaseProperties

	// Required: Specifies the identifier for the database; must be unique for your account.
	Name string
}

func (o DatabaseCreateOptions) validate() error {
	if o.Name == "" {
		return errors.New("name must not be empty")
	}
	return nil
}

// DatabaseUpdateOptions represents the options for updating a database.
type DatabaseUpdateOptions struct {
	*DatabaseProperties
}

// List all the databases by pattern.
func (d *databases) List(ctx context.Context, options DatabaseListOptions) ([]*Database, error) {
	if err := options.validate(); err != nil {
		return nil, fmt.Errorf("validate list options: %w", err)
	}

	query := fmt.Sprintf(`SHOW DATABASES LIKE '%s'`, options.Pattern)
	rows, err := d.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("do query: %w", err)
	}
	defer rows.Close()

	entities := []*Database{}
	for rows.Next() {
		var entity databaseEntity
		if err := rows.StructScan(&entity); err != nil {
			return nil, fmt.Errorf("rows scan: %w", err)
		}
		entities = append(entities, entity.toDatabase())
	}
	return entities, nil
}

// Read a database by its name.
func (d *databases) Read(ctx context.Context, databases string) (*Database, error) {
	query := fmt.Sprintf(`SHOW DATABASES LIKE '%s'`, databases)
	rows, err := d.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("do query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	var entity databaseEntity
	if err := rows.StructScan(&entity); err != nil {
		return nil, fmt.Errorf("rows scan: %w", err)
	}
	return entity.toDatabase(), nil
}

func (d *databases) formatDatabaseProperties(properties *DatabaseProperties) string {
	var s string
	if properties.Comment != nil {
		s = s + " comment='" + *properties.Comment + "'"
	}
	return s
}

// Update attributes of an existing database.
func (d *databases) Update(ctx context.Context, database string, options DatabaseUpdateOptions) (*Database, error) {
	if database == "" {
		return nil, errors.New("name must not be empty")
	}
	query := fmt.Sprintf("ALTER DATABASE %s SET", database)
	if options.DatabaseProperties != nil {
		query = query + d.formatDatabaseProperties(options.DatabaseProperties)
	}
	if _, err := d.client.Exec(ctx, query); err != nil {
		return nil, fmt.Errorf("db exec: %w", err)
	}
	return d.Read(ctx, database)
}

// Create a new database with the given options.
func (d *databases) Create(ctx context.Context, options DatabaseCreateOptions) (*Database, error) {
	if err := options.validate(); err != nil {
		return nil, fmt.Errorf("validate create options: %w", err)
	}
	query := fmt.Sprintf("CREATE DATABASE %s", options.Name)
	if options.DatabaseProperties != nil {
		query = query + d.formatDatabaseProperties(options.DatabaseProperties)
	}
	if _, err := d.client.Exec(ctx, query); err != nil {
		return nil, fmt.Errorf("db exec: %w", err)
	}
	return d.Read(ctx, options.Name)
}

// Delete a database by its name.
func (d *databases) Delete(ctx context.Context, database string) error {
	query := fmt.Sprintf(`DROP DATABASE %s`, database)
	if _, err := d.client.Exec(ctx, query); err != nil {
		return fmt.Errorf("db exec: %w", err)
	}
	return nil
}
