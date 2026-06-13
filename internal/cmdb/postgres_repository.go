package cmdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type datastore interface {
	queryer
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type PostgreSQLRepository struct {
	db datastore
}

func NewPostgreSQLRepository(db datastore) Repository {
	if db == nil {
		return nil
	}
	return &PostgreSQLRepository{db: db}
}

func (r *PostgreSQLRepository) ListApplications(ctx context.Context, query ApplicationListQuery) ([]ApplicationSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT app_id::text,
       app_code,
       app_name,
       environment,
       status,
       lifecycle,
       owner_team_code,
       owner_team_name,
       language,
       repository_url,
       instance_count,
       active_instance_count,
       updated_at,
       cache_version
FROM app_read_model
WHERE ($1 = '' OR environment = $1)
  AND ($2 = '' OR status = $2)
  AND (
      $3 = ''
      OR app_code ILIKE '%' || $3 || '%'
      OR app_name ILIKE '%' || $3 || '%'
  )
ORDER BY app_code ASC
LIMIT $4 OFFSET $5`,
		query.Environment,
		query.Status,
		query.Search,
		query.Limit,
		query.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ApplicationSummary, 0)
	for rows.Next() {
		var item ApplicationSummary
		var ownerTeamCode sql.NullString
		var ownerTeamName sql.NullString
		var language sql.NullString
		var repositoryURL sql.NullString
		if err := rows.Scan(
			&item.AppID,
			&item.AppCode,
			&item.AppName,
			&item.Environment,
			&item.Status,
			&item.Lifecycle,
			&ownerTeamCode,
			&ownerTeamName,
			&language,
			&repositoryURL,
			&item.InstanceCount,
			&item.ActiveInstanceCount,
			&item.UpdatedAt,
			&item.CacheVersion,
		); err != nil {
			return nil, err
		}
		item.OwnerTeamCode = stringValue(ownerTeamCode)
		item.OwnerTeamName = stringValue(ownerTeamName)
		item.Language = stringValue(language)
		item.RepositoryURL = stringValue(repositoryURL)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) GetApplication(ctx context.Context, appID string) (*ApplicationDetail, error) {
	return r.getApplication(ctx, appID)
}

func (r *PostgreSQLRepository) CreateApplication(ctx context.Context, request CreateApplicationRequest) (*ApplicationDetail, error) {
	labels, err := labelsJSON(request.Labels)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackQuietly(tx)

	var ciID string
	if err := tx.QueryRowContext(ctx, `
INSERT INTO ci_core (
    ci_type,
    ci_code,
    ci_name,
    display_name,
    status,
    lifecycle,
    environment,
    source_system,
    external_id,
    labels
) VALUES (
    'application',
    $1,
    $2,
    $2,
    $3,
    $4,
    $5,
    'stellatlas-api',
    $1,
    $6::jsonb
)
RETURNING ci_id::text`,
		request.AppCode,
		request.AppName,
		request.Status,
		request.Lifecycle,
		request.Environment,
		labels,
	).Scan(&ciID); err != nil {
		return nil, mapMutationError(err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO app_read_model (
    app_id,
    app_code,
    app_name,
    environment,
    status,
    lifecycle,
    owner_team_code,
    owner_team_name,
    language,
    repository_url,
    cache_version,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    NULLIF($7, ''),
    NULLIF($8, ''),
    NULLIF($9, ''),
    NULLIF($10, ''),
    1,
    now()
)`,
		ciID,
		request.AppCode,
		request.AppName,
		request.Environment,
		request.Status,
		request.Lifecycle,
		request.OwnerTeamCode,
		request.OwnerTeamName,
		request.Language,
		request.RepositoryURL,
	); err != nil {
		return nil, mapMutationError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.getApplication(ctx, ciID)
}

func (r *PostgreSQLRepository) UpdateApplication(ctx context.Context, request UpdateApplicationRequest) (*ApplicationDetail, error) {
	labels, err := labelsJSON(request.Labels)
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackQuietly(tx)

	var ciID string
	if err := tx.QueryRowContext(ctx, `
SELECT app_id::text
FROM app_read_model
WHERE app_id::text = $1 OR app_code = $1`,
		request.AppID,
	).Scan(&ciID); err != nil {
		return nil, mapSelectOneError(err)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE ci_core
SET ci_name = $2,
    display_name = $2,
    status = $3,
    lifecycle = $4,
    environment = $5,
    labels = $6::jsonb,
    deleted_at = CASE WHEN $3 = 'deleted' THEN COALESCE(deleted_at, now()) ELSE NULL END
WHERE ci_id = $1`,
		ciID,
		request.AppName,
		request.Status,
		request.Lifecycle,
		request.Environment,
		labels,
	); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE app_read_model
SET app_name = $2,
    environment = $3,
    status = $4,
    lifecycle = $5,
    owner_team_code = NULLIF($6, ''),
    owner_team_name = NULLIF($7, ''),
    language = NULLIF($8, ''),
    repository_url = NULLIF($9, ''),
    cache_version = cache_version + 1,
    updated_at = now()
WHERE app_id = $1`,
		ciID,
		request.AppName,
		request.Environment,
		request.Status,
		request.Lifecycle,
		request.OwnerTeamCode,
		request.OwnerTeamName,
		request.Language,
		request.RepositoryURL,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.getApplication(ctx, ciID)
}

func (r *PostgreSQLRepository) DeleteApplication(ctx context.Context, appID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackQuietly(tx)

	var ciID string
	if err := tx.QueryRowContext(ctx, `
SELECT app_id::text
FROM app_read_model
WHERE app_id::text = $1 OR app_code = $1`,
		appID,
	).Scan(&ciID); err != nil {
		return mapSelectOneError(err)
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE ci_core
SET status = 'deleted',
    lifecycle = 'retired',
    deleted_at = COALESCE(deleted_at, now())
WHERE ci_id = $1`,
		ciID,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE app_read_model
SET status = 'deleted',
    lifecycle = 'retired',
    cache_version = cache_version + 1,
    updated_at = now()
WHERE app_id = $1`,
		ciID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgreSQLRepository) ListApplicationOwners(ctx context.Context, appID string) ([]PersonRelation, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT app_id::text,
       person_id::text,
       person_code,
       person_name,
       email,
       role,
       relation_source,
       valid_from,
       valid_to,
       observed_at
FROM app_owner_read_model
WHERE app_id::text = $1
   OR app_id = (
      SELECT app_id
      FROM app_read_model
      WHERE app_code = $1
      ORDER BY environment ASC
      LIMIT 1
   )
ORDER BY role ASC, person_name ASC`,
		appID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PersonRelation, 0)
	for rows.Next() {
		var item PersonRelation
		var email sql.NullString
		var validTo sql.NullTime
		if err := rows.Scan(
			&item.AppID,
			&item.PersonID,
			&item.PersonCode,
			&item.PersonName,
			&email,
			&item.Role,
			&item.RelationSource,
			&item.ValidFrom,
			&validTo,
			&item.ObservedAt,
		); err != nil {
			return nil, err
		}
		item.Email = stringValue(email)
		item.ValidTo = timePtr(validTo)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) ListApplicationInstances(ctx context.Context, query InstanceListQuery) ([]InstanceSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT app_id::text,
       instance_ci_id::text,
       instance_external_id,
       environment,
       region,
       zone,
       private_ip,
       public_ip,
       port,
       version,
       runtime_status,
       resource_version,
       observed_at,
       updated_at
FROM app_instance_snapshot
WHERE (
      app_id::text = $1
      OR app_id = (
          SELECT app_id
          FROM app_read_model
          WHERE app_code = $1
          ORDER BY environment ASC
          LIMIT 1
      )
  )
  AND ($2 = '' OR environment = $2)
ORDER BY observed_at DESC
LIMIT $3 OFFSET $4`,
		query.AppID,
		query.Environment,
		query.Limit,
		query.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]InstanceSummary, 0)
	for rows.Next() {
		var item InstanceSummary
		var instanceCIID sql.NullString
		var region sql.NullString
		var zone sql.NullString
		var privateIP sql.NullString
		var publicIP sql.NullString
		var port sql.NullInt64
		var version sql.NullString
		var resourceVersion sql.NullString
		if err := rows.Scan(
			&item.AppID,
			&instanceCIID,
			&item.InstanceExternalID,
			&item.Environment,
			&region,
			&zone,
			&privateIP,
			&publicIP,
			&port,
			&version,
			&item.RuntimeStatus,
			&resourceVersion,
			&item.ObservedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.InstanceCIID = stringValue(instanceCIID)
		item.Region = stringValue(region)
		item.Zone = stringValue(zone)
		item.PrivateIP = stringValue(privateIP)
		item.PublicIP = stringValue(publicIP)
		item.Port = intValue(port)
		item.Version = stringValue(version)
		item.ResourceVersion = stringValue(resourceVersion)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) getApplication(ctx context.Context, appID string) (*ApplicationDetail, error) {
	var item ApplicationDetail
	var ownerTeamCode sql.NullString
	var ownerTeamName sql.NullString
	var language sql.NullString
	var repositoryURL sql.NullString
	var labelsRaw []byte
	if err := r.db.QueryRowContext(ctx, `
SELECT app.app_id::text,
       app.app_code,
       app.app_name,
       app.environment,
       app.status,
       app.lifecycle,
       app.owner_team_code,
       app.owner_team_name,
       app.language,
       app.repository_url,
       app.instance_count,
       app.active_instance_count,
       core.labels,
       core.created_at,
       app.updated_at,
       app.cache_version
FROM app_read_model app
JOIN ci_core core ON core.ci_id = app.app_id
WHERE app.app_id::text = $1 OR app.app_code = $1`,
		strings.TrimSpace(appID),
	).Scan(
		&item.CIID,
		&item.AppCode,
		&item.AppName,
		&item.Environment,
		&item.Status,
		&item.Lifecycle,
		&ownerTeamCode,
		&ownerTeamName,
		&language,
		&repositoryURL,
		&item.InstanceCount,
		&item.ActiveInstanceCount,
		&labelsRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CacheVersion,
	); err != nil {
		return nil, mapSelectOneError(err)
	}
	item.AppID = item.AppCode
	item.OwnerTeamCode = stringValue(ownerTeamCode)
	item.OwnerTeamName = stringValue(ownerTeamName)
	item.Language = stringValue(language)
	item.RepositoryURL = stringValue(repositoryURL)
	item.Labels = mapStringValue(labelsRaw)
	naming, _ := ValidateStandardAppID(item.AppID)
	item.Naming = naming
	return &item, nil
}

func labelsJSON(labels map[string]string) (string, error) {
	if labels == nil {
		labels = map[string]string{}
	}
	data, err := json.Marshal(labels)
	if err != nil {
		return "", fmt.Errorf("marshal labels: %w", err)
	}
	return string(data), nil
}

func mapStringValue(data []byte) map[string]string {
	if len(data) == 0 {
		return nil
	}
	values := map[string]string{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func mapMutationError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "duplicate key") {
		return ErrApplicationDuplicate
	}
	return err
}

func mapSelectOneError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrApplicationNotFound
	}
	return err
}

func rollbackQuietly(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func stringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func intValue(value sql.NullInt64) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int64)
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
