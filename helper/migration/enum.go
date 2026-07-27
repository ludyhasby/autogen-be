package helpermigration

import (
	"fmt"
	coreenum "logisfy/core/enum"
	"strings"

	"gorm.io/gorm"
)

func CreateExtensionEnum(db *gorm.DB) error {
	values := make([]string, 0)

	for _, v := range coreenum.CTXEnumExtensionValues {
		values = append(values, fmt.Sprintf("'%s'", v))
	}

	query := fmt.Sprintf(`
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_type WHERE typname = 'extension_enum'
		) THEN
			CREATE TYPE extension_enum AS ENUM (%s);
		END IF;
	END$$;
	`, strings.Join(values, ", "))

	return db.Exec(query).Error
}

func CreateLocationTypeEnum(db *gorm.DB) error {
	values := make([]string, 0)

	for _, v := range coreenum.CTXEnumLocationTypeValues {
		values = append(values, fmt.Sprintf("'%s'", v))
	}

	query := fmt.Sprintf(`
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_type WHERE typname = 'location_type_enum'
		) THEN
			CREATE TYPE location_type_enum AS ENUM (%s);
		END IF;
	END$$;
	`, strings.Join(values, ", "))

	return db.Exec(query).Error
}

func CreateRoleEnum(db *gorm.DB) error {
	values := make([]string, 0)

	for _, v := range coreenum.CTXEnumRoleValues {
		values = append(values, fmt.Sprintf("'%s'", v))
	}

	query := fmt.Sprintf(`
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_type WHERE typname = 'role_enum'
		) THEN
			CREATE TYPE role_enum AS ENUM (%s);
		END IF;
	END$$;
	`, strings.Join(values, ", "))

	return db.Exec(query).Error
}

func CreateStageProcessEnum(db *gorm.DB) error {
	values := make([]string, 0)

	for _, v := range coreenum.CTXEnumStageProcessValues {
		values = append(values, fmt.Sprintf("'%s'", v))
	}

	query := fmt.Sprintf(`
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_type WHERE typname = 'stage_process_enum'
		) THEN
			CREATE TYPE stage_process_enum AS ENUM (%s);
		END IF;
	END$$;
	`, strings.Join(values, ", "))

	return db.Exec(query).Error
}

func CreateVoltageTypeEnum(db *gorm.DB) error {
	values := make([]string, 0)

	for _, v := range coreenum.CTXEnumVoltageTypeValues {
		values = append(values, fmt.Sprintf("'%s'", v))
	}

	query := fmt.Sprintf(`
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_type WHERE typname = 'voltage_type_enum'
		) THEN
			CREATE TYPE voltage_type_enum AS ENUM (%s);
		END IF;
	END$$;
	`, strings.Join(values, ", "))

	return db.Exec(query).Error
}

func CreateMeasurementTypeEnum(db *gorm.DB) error {
	values := make([]string, 0)

	for _, v := range coreenum.CTXEnumMeasurementTypeValues {
		values = append(values, fmt.Sprintf("'%s'", v))
	}

	query := fmt.Sprintf(`
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_type WHERE typname = 'measurement_type_enum'
		) THEN
			CREATE TYPE measurement_type_enum AS ENUM (%s);
		END IF;
	END$$;
	`, strings.Join(values, ", "))

	return db.Exec(query).Error
}
