package namespace

import (
	"context"
	"database/sql"
	"fmt"
)

// These files aim at providing a way to handle MySQL specific queries
// while being able to run them on a SQLite database.
//
// It uses the sqlparser from the vitess project to parse the queries
// and rewrite them to be SQLite compatible.

// prepareDatabaseForMySQL registers a few views and databases
// that are required for MySQL compatibility
//
// It must be called once before running any query
func prepareDatabaseForMySQL(db *sql.Conn) error {
	// We will register a new database named information_schema
	// First, we check if it already exists
	var exists bool
	row := db.QueryRowContext(context.Background(), `SELECT count(*) FROM pragma_database_list() WHERE name = 'information_schema'`)
	err := row.Scan(&exists)
	if err != nil || row.Err() != nil || exists == false {
		// We consider that the database does not exist
		_, err = db.ExecContext(context.Background(), `ATTACH DATABASE 'file:mymemory.db?immutable=1&mode=memory&cache=shared' AS 'information_schema';`)
		if err != nil {
			return err
		}
	}

	// We create views that will be used to retrive collations, tables, columns, constraints and views
	_, err = db.ExecContext(context.Background(), `
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.COLLATIONS AS
	SELECT
		column1 AS COLLATION_NAME,
		column2 AS CHARACTER_SET_NAME,
		column3 AS ID,
		column4 AS IS_DEFAULT,
		column5 AS IS_COMPILED,
		column6 AS SORTLEN,
		column7 AS PAD_ATTRIBUTE
	FROM (
	VALUES('RTRIM', 'utf8mb4', 0, '', 'YES', 1, 'PAD SPACE'),
		('NOCASE',
		'utf8mb4',
		1,
		'',
		'YES',
		1,
		'NO PAD'),
	('BINARY',
	'utf8mb4',
	2,
	'YES',
	'YES',
	1,
	'NO PAD')
	);`)

	if err != nil {
		return err
	}

	_, err = db.ExecContext(context.Background(), `
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.PARTITIONS AS SELECT
  	"def" AS TABLE_CATALOG,
  	tl.schema AS TABLE_SCHEMA,
    tl.name AS TABLE_NAME,
    CASE
    	WHEN tl.schema = 'information_schema' THEN 'SYSTEM VIEW'
    	WHEN tl.type = 'view' THEN 'VIEW'
    	ELSE 'BASE TABLE'
    END AS TABLE_TYPE,
    NULL AS PARTITION_NAME,
	NULL AS SUBPARTITION_NAME,
    NULL AS PARTITION_ORDINAL_POSITION,
	NULL AS SUBPARTITION_ORDINAL_POSITION,
    NULL AS PARTITION_METHOD,
	NULL AS SUBPARTITION_METHOD,
    NULL AS PARTITION_EXPRESSION,
	NULL AS SUBPARTITION_EXPRESSION,
    NULL AS PARTITION_DESCRIPTION,
    0 AS TABLE_ROWS,
    0 AS AVG_ROW_LENGTH,
    0 AS DATA_LENGTH,
    0 AS MAX_DATA_LENGTH,
    0 AS INDEX_LENGTH,
    0 AS DATA_FREE,
    '1970-01-01 00:00:00' AS CREATE_TIME,
    '1970-01-01 00:00:00' AS UPDATE_TIME,
    NULL AS CHECK_TIME,
    NULL AS CHECKSUM,
    '' AS PARTITION_COMMENT,
    '' AS NODEGROUP,
    NULL AS TABLESPACE_NAME
  FROM
    pragma_table_list tl
  UNION
  SELECT
    "def" AS TABLE_CATALOG,
  	"main" AS TABLE_SCHEMA,
    ml.name AS TABLE_NAME,
    'BASE TABLE' AS TABLE_TYPE,
    NULL AS PARTITION_NAME,
	NULL AS SUBPARTITION_NAME,
    NULL AS PARTITION_ORDINAL_POSITION,
	NULL AS SUBPARTITION_ORDINAL_POSITION,
    NULL AS PARTITION_METHOD,
	NULL AS SUBPARTITION_METHOD,
    NULL AS PARTITION_EXPRESSION,
	NULL AS SUBPARTITION_EXPRESSION,
    NULL AS PARTITION_DESCRIPTION,
    0 AS TABLE_ROWS,
    0 AS AVG_ROW_LENGTH,
    0 AS DATA_LENGTH,
    0 AS MAX_DATA_LENGTH,
    0 AS INDEX_LENGTH,
    0 AS DATA_FREE,
    '1970-01-01 00:00:00' AS CREATE_TIME,
    '1970-01-01 00:00:00' AS UPDATE_TIME,
    NULL AS CHECK_TIME,
    NULL AS CHECKSUM,
    '' AS PARTITION_COMMENT,
    '' AS NODEGROUP,
    NULL AS TABLESPACE_NAME
	  FROM
	  		pragma_module_list ml
	  WHERE ml.name NOT LIKE 'fts%'
	  AND ml.name NOT LIKE 'rtree%'
	  AND ml.name NOT LIKE '%_reader';


	-- INFORMATION_SCHEMA.COLUMNS
CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.COLUMNS AS WITH RECURSIVE table_list AS (
  SELECT name, schema
  FROM pragma_table_list()
  UNION
  SELECT name, "main"
  FROM pragma_module_list()
  WHERE name NOT LIKE 'fts%'
  AND name NOT LIKE 'rtree%'
  AND name NOT LIKE '%_reader'
),
table_info AS (
  SELECT
  	"def" AS TABLE_CATALOG,
  	tl.schema AS TABLE_SCHEMA,
    tl.name AS TABLE_NAME,
    ti.name AS COLUMN_NAME,
    ti.cid AS ORDINAL_POSITION,
    ti.dflt_value AS COLUMN_DEFAULT,
    iif(ti."notnull" AND ti.pk, 'YES', 'NO') AS IS_NULLABLE,
    CASE upper(ti.type)
    	WHEN 'TEXT' THEN 'varchar'
    	WHEN 'INT' THEN 'bigint'
    	WHEN 'REAL' THEN 'float'
    	WHEN 'BLOB' THEN 'blob'
		WHEN 'INTEGER' THEN 'bigint'
		WHEN 'TINYINT' THEN 'bigint'
		WHEN 'SMALLINT' THEN 'bigint'
		WHEN 'MEDIUMINT' THEN 'bigint'
		WHEN 'BIGINT' THEN 'bigint'
		WHEN 'UNSIGNED BIG INT' THEN 'bigint'
		WHEN 'INT2' THEN 'bigint'
		WHEN 'INT8' THEN 'bigint'
		WHEN 'VARCHAR' THEN 'varchar'
		WHEN 'VARCHAR(255)' THEN 'varchar'
		ELSE 'varchar'
    END AS DATA_TYPE,
    65535 AS CHARACTER_MAXIMUM_LENGTH,
    65535 AS CHARACTER_OCTET_LENGTH,
    NULL AS NUMERIC_PRECISION,
    NULL AS NUMERIC_SCALE,
    NULL AS DATETIME_PRECISION,
    'utf8mb3' AS CHARACTER_SET_NAME,
    'BINARY' AS COLLATION_NAME,
    CASE upper(ti.type)
    	WHEN 'TEXT' THEN 'varchar(65535)'
    	WHEN 'INT' THEN 'int'
    	WHEN 'REAL' THEN 'double'
    	WHEN 'BLOB' THEN 'blob'
		WHEN 'INTEGER' THEN 'int'
		WHEN 'TINYINT' THEN 'int'
		WHEN 'SMALLINT' THEN 'int'
		WHEN 'MEDIUMINT' THEN 'int'
		WHEN 'BIGINT' THEN 'int'
		WHEN 'UNSIGNED BIG INT' THEN 'int'
		WHEN 'INT2' THEN 'int'
		WHEN 'INT8' THEN 'int'
		WHEN 'VARCHAR' THEN 'varchar(65535)'
		WHEN 'VARCHAR(255)' THEN 'varchar(65535)'
		ELSE 'varchar(65535)'

    END AS COLUMN_TYPE,
    iif(ti.pk, 'PRI', '') AS COLUMN_KEY,
    iif(tl.schema='information_schema', 'select', 'select,insert,update,references') AS PRIVILEGES,
    '' AS COLUMN_COMMENT,
    '' AS GENERATION_EXPRESSION,
    NULL AS SRS_ID,
	'' AS EXTRA
  FROM
    table_list tl,
    pragma_table_info(tl.name) ti
)
SELECT * FROM table_info;`)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(context.Background(), `
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.TABLE_CONSTRAINTS AS WITH RECURSIVE table_list AS (
		SELECT name, schema
		FROM pragma_table_list()
		WHERE "type" = 'table'
		UNION
		SELECT name, "main"
		FROM pragma_module_list()
		WHERE name NOT LIKE 'fts%'
		AND name NOT LIKE 'rtree%'
		AND name NOT LIKE '%_reader'
	  ),
	  table_info AS (
		SELECT
			"def" AS CONSTRAINT_CATALOG,
			tl.schema AS CONSTRAINT_SCHEMA,
			concat('pk_', tl.name, '_', ti.name) AS CONSTRAINT_NAME,
			tl.schema AS TABLE_SCHEMA,
		  tl.name AS TABLE_NAME,
		  'PRIMARY KEY' AS CONSTRAINT_TYPE,
		  'YES' AS ENFORCED
		FROM
		  table_list tl,
		  pragma_table_info(tl.name) ti
		WHERE ti.pk
	  )
	  SELECT * FROM table_info;
	  
	  -- INFORMATION_SCHEMA.TABLES
	   CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.TABLES AS SELECT
			"def" AS TABLE_CATALOG,
			tl.schema AS TABLE_SCHEMA,
		  tl.name AS TABLE_NAME,
		  CASE
			  WHEN tl.schema = 'information_schema' THEN 'SYSTEM VIEW'
			  WHEN tl.type = 'view' THEN 'VIEW'
			  ELSE 'BASE TABLE'
		  END AS TABLE_TYPE,
		  'SQLite' as ENGINE,
		  10 AS VERSION,
		  iif(tl.type = 'BASE TABLE', 'Dynamic', NULL) AS ROW_FORMAT,
		  0 AS TABLE_ROWS,
		  0 AS AVG_ROW_LENGTH,
		  0 AS DATA_LENGTH,
		  0 AS MAX_DATA_LENGTH,
		  0 AS INDEX_LENGTH,
		  0 AS DATA_FREE,
		  NULL AS AUTO_INCREMENT,
		  '1970-01-01 00:00:00' AS CREATE_TIME,
		  '1970-01-01 00:00:00' AS UPDATE_TIME,
		  NULL AS CHECK_TIME,
		  'BINARY' AS TABLE_COLLATION,
		  NULL AS CHECKSUM,
		  '' AS CREATE_OPTIONS,
		  '' AS TABLE_COMMENT
		FROM
		  pragma_table_list tl
		UNION
		SELECT
		  "def" AS TABLE_CATALOG,
		  "main" AS TABLE_SCHEMA,
		  ml.name AS TABLE_NAME,
		  'BASE TABLE' AS TABLE_TYPE,
		  'SQLite' as ENGINE,
		  10 AS VERSION,
		  'Dynamic' AS ROW_FORMAT,
		  0 AS TABLE_ROWS,
		  0 AS AVG_ROW_LENGTH,
		  0 AS DATA_LENGTH,
		  0 AS MAX_DATA_LENGTH,
		  0 AS INDEX_LENGTH,
		  0 AS DATA_FREE,
		  NULL AS AUTO_INCREMENT,
		  '1970-01-01 00:00:00' AS CREATE_TIME,
		  '1970-01-01 00:00:00' AS UPDATE_TIME,
		  NULL AS CHECK_TIME,
		  'BINARY' AS TABLE_COLLATION,
		  NULL AS CHECKSUM,
		  '' AS CREATE_OPTIONS,
		  '' AS TABLE_COMMENT
		FROM
		  pragma_module_list ml
		WHERE ml.name NOT LIKE 'fts%'
		AND ml.name NOT LIKE 'rtree%'
		AND ml.name NOT LIKE '%_reader';`)

	if err != nil {
		return fmt.Errorf("error creating view INFORMATION_SCHEMA.TABLES: %w", err)
	}

	_, err = db.ExecContext(context.Background(), `
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.VIEWS AS SELECT
	'def' AS TABLE_CATALOG,
  	tl.schema AS TABLE_SCHEMA,
    tl.name AS TABLE_NAME,
    (SELECT sql FROM sqlite_schema sch WHERE sch.name = tl.name LIMIT 1)
    AS VIEW_DEFINITION,
    'NONE' AS CHECK_OPTION,
    'NO' AS IS_UPDATABLE,
    'root@localhost' AS DEFINER,
    'INVOKER' AS SECURITY_TYPE,
    'utf8mb3' AS CHARACTER_SET_CLIENT,
    'BINARY' AS COLLATION_CONNECTION
	FROM
		pragma_table_list tl
	WHERE tl.type = 'view';

	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.STATISTICS AS WITH RECURSIVE table_list AS (
		SELECT name, schema
		FROM pragma_table_list()
		WHERE "type" = 'table'
		UNION
		SELECT name, "main"
		FROM pragma_module_list()
		WHERE name NOT LIKE 'fts%'
		AND name NOT LIKE 'rtree%'
		AND name NOT LIKE '%_reader'
	  ),
	  table_info AS (
		SELECT DISTINCT
			"def" AS TABLE_CATALOG,
			tl.schema AS TABLE_SCHEMA,
			tl.name AS TABLE_NAME,
			0 AS NON_UNIQUE,
			tl.schema AS INDEX_SCHEMA,
			'PRIMARY' AS INDEX_NAME,
			1 AS SEQ_IN_INDEX,
			ti.name AS COLUMN_NAME,
			NULL AS COLLATION,
			NULL AS SUB_PART,
			NULL AS PACKED,
			'' AS NULLABLE,
			'BTREE' AS INDEX_TYPE,
			'' AS COMMENT,
			'' AS INDEX_COMMENT,
			'YES' AS IS_VISIBLE,
			NULL AS EXPRESSION
		FROM
		  table_list tl,
		  pragma_table_info(tl.name) ti
		 WHERE ti.pk
	  )
	  SELECT * FROM table_info;
	  


	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.SCHEMATA AS SELECT 
		'def' AS TABLE_CATALOG,
		name AS SCHEMA_NAME,
		'utf8mb3' AS DEFAULT_CHARACTER_SET_NAME,
		'BINARY' AS DEFAULT_COLLATION_NAME,
		NULL AS SQL_PATH,
		'NO' AS DEFAULT_ENCRYPTION
	FROM pragma_database_list();
	
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.KEY_COLUMN_USAGE AS WITH RECURSIVE table_list AS (
		SELECT name, schema
		FROM pragma_table_list()
		WHERE "type" = 'table'
		UNION
		SELECT name, "main"
		FROM pragma_module_list()
		WHERE name NOT LIKE 'fts%'
		AND name NOT LIKE 'rtree%'
		AND name NOT LIKE '%_reader'
	  ),
	  table_info AS (
		SELECT
			"def" AS CONSTRAINT_CATALOG,
			tl.schema AS CONSTRAINT_SCHEMA,
			concat('pk_', tl.name, '_', ti.name) AS CONSTRAINT_NAME,
			"def" AS TABLE_CATALOG,
			tl.schema AS TABLE_SCHEMA,
		  tl.name AS TABLE_NAME,
		  ti.name AS COLUMN_NAME,
		  1 as ORDINAL_POSITION,
		  NULL AS POSITION_IN_UNIQUE_CONSTRAINT,
		  NULL AS REFERENCED_TABLE_SCHEMA,
		  NULL AS REFERENCED_TABLE_NAME,
		  NULL AS REFERENCED_COLUMN_NAME
		FROM
		  table_list tl,
		  pragma_table_info(tl.name) ti
	  )
	  SELECT * FROM table_info;
	
	`)
	if err != nil {
		return fmt.Errorf("error creating view INFORMATION_SCHEMA.VIEWS: %w", err)
	}

	_, err = db.ExecContext(context.Background(), `CREATE VIEW IF NOT EXISTS dual AS SELECT 'x' AS dummy;`)
	if err != nil {
		return err
	}

	// Those views are empty but created so that SQLite does not return a missing table error
	_, err = db.ExecContext(context.Background(), `
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.STATISTICS AS
	SELECT
		'' AS TABLE_CATALOG,
		'' AS TABLE_SCHEMA,
		'' AS TABLE_NAME,
		0 AS NON_UNIQUE,
		'' AS INDEX_SCHEMA,
		'' AS INDEX_NAME,
		0 AS SEQ_IN_INDEX,
		'' AS COLUMN_NAME,
		'' AS COLLATION,
		0 AS CARDINALITY,
		0 AS SUB_PART,
		'' AS PACKED,
		'' AS NULLABLE,
		'' AS INDEX_TYPE,
		'' AS COMMENT,
		'' AS INDEX_COMMENT,
		'YES' AS IS_VISIBLE,
		NULL AS EXPRESSION
		WHERE FALSE;
		
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.TRIGGERS AS
	SELECT
		'' AS TRIGGER_CATALOG,
		'' AS TRIGGER_SCHEMA,
		'' AS TRIGGER_NAME,
		'' AS EVENT_MANIPULATION,
		'' AS EVENT_OBJECT_CATALOG,
		'' AS EVENT_OBJECT_SCHEMA,
		'' AS EVENT_OBJECT_TABLE,
		'' AS ACTION_ORDER,
		'' AS ACTION_CONDITION,
		'' AS ACTION_STATEMENT,
		'' AS ACTION_ORIENTATION,
		'' AS ACTION_TIMING,
		'' AS ACTION_REFERENCE_OLD_TABLE,
		'' AS ACTION_REFERENCE_NEW_TABLE,
		'' AS ACTION_REFERENCE_OLD_ROW,
		'' AS ACTION_REFERENCE_NEW_ROW,
		'' AS CREATED,
		'' AS SQL_MODE,
		'' AS DEFINER,
		'' AS CHARACTER_SET_CLIENT,
		'' AS COLLATION_CONNECTION,
		'' AS DATABASE_COLLATION
	WHERE FALSE;


	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.ROUTINES AS
	SELECT
		'' AS SPECIFIC_NAME,
		'' AS ROUTINE_CATALOG,
		'' AS ROUTINE_SCHEMA,
		'' AS ROUTINE_NAME,
		'' AS ROUTINE_TYPE,
		'' AS DATA_TYPE,
		'' AS CHARACTER_MAXIMUM_LENGTH,
		'' AS CHARACTER_OCTET_LENGTH,
		NULL AS NUMERIC_PRECISION,
		NULL AS NUMERIC_SCALE,
		NULL AS DATETIME_PRECISION,
		'' AS CHARACTER_SET_NAME,
		'' AS COLLATION_NAME,
		'' AS DTD_IDENTIFIER,
		'' AS ROUTINE_BODY,
		'' AS ROUTINE_DEFINITION,
		'' AS EXTERNAL_NAME,
		'' AS EXTERNAL_LANGUAGE,
		'' AS PARAMETER_STYLE,
		'' AS IS_DETERMINISTIC,
		'' AS SQL_DATA_ACCESS,
		'' AS SQL_PATH,
		'' AS SECURITY_TYPE,
		'' AS CREATED,
		'' AS LAST_ALTERED,
		'' AS SQL_MODE,
		'' AS ROUTINE_COMMENT,
		'' AS DEFINER,
		'' AS CHARACTER_SET_CLIENT,
		'' AS COLLATION_CONNECTION,
		'' AS DATABASE_COLLATION
	WHERE FALSE;
		
	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.EVENTS AS
	SELECT
		'' AS EVENT_CATALOG,
		'' AS EVENT_SCHEMA,
		'' AS EVENT_NAME,
		'' AS DEFINER,
		'' AS TIME_ZONE,
		'' AS EVENT_BODY,
		'' AS EVENT_DEFINITION,
		'' AS EVENT_TYPE,
		'' AS EXECUTE_AT,
		'' AS INTERVAL_VALUE,
		'' AS INTERVAL_FIELD,
		'' AS SQL_MODE,
		'' AS STARTS,
		'' AS ENDS,
		'' AS STATUS,
		'' AS ON_COMPLETION,
		'' AS CREATED,
		'' AS LAST_ALTERED,
		'' AS LAST_EXECUTED,
		'' AS EVENT_COMMENT,
		'' AS ORIGINATOR,
		'' AS CHARACTER_SET_CLIENT,
		'' AS COLLATION_CONNECTION,
		'' AS DATABASE_COLLATION
	WHERE FALSE;

	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.TABLE_PRIVILEGES AS
	SELECT
		'' AS GRANTEE,
		'' AS TABLE_CATALOG,
		'' AS TABLE_SCHEMA,
		'' AS TABLE_NAME,
		'' AS PRIVILEGE_TYPE,
		'' AS IS_GRANTABLE
	WHERE FALSE;

	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS AS
	SELECT
		'' AS CONSTRAINT_CATALOG,
		'' AS CONSTRAINT_SCHEMA,
		'' AS CONSTRAINT_NAME,
		'' AS UNIQUE_CONSTRAINT_CATALOG,
		'' AS UNIQUE_CONSTRAINT_SCHEMA,
		'' AS UNIQUE_CONSTRAINT_NAME,
		'' AS MATCH_OPTION,
		'' AS UPDATE_RULE,
		'' AS DELETE_RULE,
		'' AS TABLE_NAME,
		'' AS REFERENCED_TABLE_NAME
	WHERE FALSE;

	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.COLUMN_PRIVILEGES AS
	SELECT
		'' AS GRANTEE,
		'' AS TABLE_CATALOG,
		'' AS TABLE_SCHEMA,
		'' AS TABLE_NAME,
		'' AS COLUMN_NAME,
		'' AS PRIVILEGE_TYPE,
		'' AS IS_GRANTABLE
	WHERE FALSE;

	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.USER_PRIVILEGES AS
	SELECT
		'' AS GRANTEE,
		'' AS TABLE_CATALOG,
		'' AS PRIVILEGE_TYPE,
		'NO' AS IS_GRANTABLE
	WHERE FALSE;

	CREATE VIEW IF NOT EXISTS INFORMATION_SCHEMA.SCHEMA_PRIVILEGES AS
	SELECT
		'' AS GRANTEE,
		'' AS TABLE_CATALOG,
		'' AS TABLE_SCHEMA,
		'' AS PRIVILEGE_TYPE,
		'' AS IS_GRANTABLE
	WHERE FALSE;
	`)
	if err != nil {
		return fmt.Errorf("error creating empty views: %w", err)
	}

	// Now we do the same fake view for the dabase mysql
	// Check if the database already exists
	row = db.QueryRowContext(context.Background(), `SELECT count(*) FROM pragma_database_list() WHERE name = 'mysql'`)
	err = row.Scan(&exists)
	if err != nil || row.Err() != nil || exists == false {
		// We consider that the database does not exist
		_, err = db.ExecContext(context.Background(), `ATTACH DATABASE 'file:mymemory2.db?immutable=1&mode=memory&cache=shared' AS 'mysql';`)
		if err != nil {
			return fmt.Errorf("error attaching database mysql: %w", err)
		}
	}

	_, err = db.ExecContext(context.Background(), `
	CREATE VIEW IF NOT EXISTS mysql.user AS SELECT
	column1 AS Host,
	column2 AS User,
	column3 AS Select_priv,
	column4 AS Insert_priv,
	column5 AS Update_priv,
	column6 AS Delete_priv,
	column7 AS Create_priv,
	column8 AS Drop_priv,
	column9 AS Reload_priv,
	column10 AS Shutdown_priv,
	column11 AS Process_priv,
	column12 AS File_priv,
	column13 AS Grant_priv,
	column14 AS References_priv,
	column15 AS Index_priv,
	column16 AS Alter_priv,
	column17 AS Show_db_priv,
	column18 AS Super_priv,
	column19 AS Create_tmp_table_priv,
	column20 AS Lock_tables_priv,
	column21 AS Execute_priv,
	column22 AS Repl_slave_priv,
	column23 AS Repl_client_priv,
	column24 AS Create_view_priv,
	column25 AS Show_view_priv,
	column26 AS Create_routine_priv,
	column27 AS Alter_routine_priv,
	column28 AS Create_user_priv,
	column29 AS Event_priv,
	column30 AS Trigger_priv,
	column31 AS Create_tablespace_priv,
	column32 AS ssl_type,
	column33 AS ssl_cipher,
	column34 AS x509_issuer,
	column35 AS x509_subject,
	column36 AS max_questions,
	column37 AS max_updates,
	column38 AS max_connections,
	column39 AS max_user_connections,
	column40 AS plugin,
	column41 AS authentication_string,
	column42 AS password_expired,
	column43 AS password_last_changed,
	column44 AS password_lifetime,
	column45 AS account_locked,
	column46 AS Create_role_priv,
	column47 AS Drop_role_priv,
	column48 AS Password_reuse_history,
	column49 AS Password_reuse_time,
	column50 AS Password_require_current,
	column51 AS User_attributes -- Next time MySQL, have a bigger table
	FROM (
	VALUES
	('localhost', 'root', 'Y', 'Y', 'Y', 'Y', 'Y',
	'Y', 'Y', 'Y', 'Y', 'Y',
	'Y', 'Y', 'Y', 'Y', 'Y',
	'Y', 'Y', 'Y', 'Y', 'Y',
	'Y', 'Y', 'Y', 'Y', 'Y',
	'Y', 'Y', 'Y', 'Y',
	'', '', '', '', 0, 0, 0, 0,
	'mysql_native_password', '*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19',
	'N', '1970-01-01 00:00:00', NULL, 'N', 'N', 'N',
	NULL, NULL, NULL, NULL));


	CREATE VIEW IF NOT EXISTS mysql.procs_priv AS SELECT
	'' AS Host,
	'' AS Db,
	'' AS User,
	'' AS Routine_name,
	'' AS Routine_type,
	'' AS Grantor,
	'' AS Proc_priv,
	'' AS Timestamp
	WHERE FALSE;

	CREATE VIEW IF NOT EXISTS mysql.role_edges AS SELECT
	'' AS FROM_HOST,
	'' AS FROM_USER,
	'' AS TO_HOST,
	'' AS TO_USER,
	'' AS WITH_ADMIN_OPTION
	WHERE FALSE;
	`)
	if err != nil {
		return fmt.Errorf("error creating views for mysql database: %w", err)
	}

	return nil

}
