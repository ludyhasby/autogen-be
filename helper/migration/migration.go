package helpermigration

import (
	"log/slog"
	"logisfy/internal/entity"
	"os"

	"gorm.io/gorm"
)

func AutoMigrate(DB *gorm.DB) {

	// Enum
	ExecEnum(DB)

	err := DB.AutoMigrate(
		&entity.AMRWeightConfigEntity{},
		&entity.AMRConfigEntity{},
		&entity.AMREntity{},
		&entity.UserEntity{},
		&entity.AMRDetailEntity{},
		&entity.AMRDetailResultEntity{},
		&entity.NewsEntity{},
	)
	if err != nil {
		slog.Error("Error migrate database", err)
		os.Exit(1)
	}

	// Relation
	ExecForeignKeys(DB)
}

func ExecEnum(DB *gorm.DB) {

	// Enum
	err := CreateExtensionEnum(DB)
	if err != nil {
		slog.Error("create enum error", "method", "ExecEnum()", "sub_method", "CreateExtensionEnum()", "error", err.Error())
		os.Exit(1)
	}
	err = CreateLocationTypeEnum(DB)
	if err != nil {
		slog.Error("create enum error", "method", "ExecEnum()", "sub_method", "CreateLocationTypeEnum()", "error", err.Error())
		os.Exit(1)
	}
	err = CreateRoleEnum(DB)
	if err != nil {
		slog.Error("create enum error", "method", "ExecEnum()", "sub_method", "CreateRoleEnum()", "error", err.Error())
		os.Exit(1)
	}
	err = CreateStageProcessEnum(DB)
	if err != nil {
		slog.Error("create enum error", "method", "ExecEnum()", "sub_method", "CreateStageProcessEnum()", "error", err.Error())
		os.Exit(1)
	}
	err = CreateVoltageTypeEnum(DB)
	if err != nil {
		slog.Error("create enum error", "method", "ExecEnum()", "sub_method", "CreateStageProcessEnum()", "error", err.Error())
		os.Exit(1)
	}
	err = CreateMeasurementTypeEnum(DB)
	if err != nil {
		slog.Error("create enum error", "method", "ExecEnum()", "sub_method", "CreateStageProcessEnum()", "error", err.Error())
		os.Exit(1)
	}
}

func ExecForeignKeys(db *gorm.DB) {
	print("FK RUNNING")
	// FK: amr.user_id → users.user_id
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_amr_user'
			) THEN
				ALTER TABLE "` + (&entity.AMREntity{}).TableName() + `"
				ADD CONSTRAINT fk_amr_user
				FOREIGN KEY (user_id)
				REFERENCES "` + (&entity.UserEntity{}).TableName() + `"(user_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END$$;
	`).Error; err != nil {
		slog.Error("Err migrate FK amr.user", "error", "ExecForeignKeys", "sub_method", "db.Exec()", "error", err.Error())
		os.Exit(1)
	}
	// FK: amr_weight_config.amr_id → amr.amr_id
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_amr_amr_weight_config'
			) THEN
				ALTER TABLE "` + (&entity.AMRWeightConfigEntity{}).TableName() + `"
				ADD CONSTRAINT fk_amr_amr_weight_config
				FOREIGN KEY (amr_id)
				REFERENCES "` + (&entity.AMREntity{}).TableName() + `"(amr_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END$$;
	`).Error; err != nil {
		slog.Error("Err migrate FK amr.amr_weight_config", "error", "ExecForeignKeys", "sub_method", "db.Exec()", "error", err.Error())
		os.Exit(1)
	}
	// FK: amr_config.amr_id → amr.amr_id
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_amr_amr_config'
			) THEN
				ALTER TABLE "` + (&entity.AMRConfigEntity{}).TableName() + `"
				ADD CONSTRAINT fk_amr_amr_config
				FOREIGN KEY (amr_id)
				REFERENCES "` + (&entity.AMREntity{}).TableName() + `"(amr_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END$$;
	`).Error; err != nil {
		slog.Error("Err migrate FK amr.amr_config", "error", "ExecForeignKeys", "sub_method", "db.Exec()", "error", err.Error())
		os.Exit(1)
	}
	// FK: amr_detail.amr_id → amr.amr_id
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_amr_amr_detail'
			) THEN
				ALTER TABLE "` + (&entity.AMRDetailEntity{}).TableName() + `"
				ADD CONSTRAINT fk_amr_amr_detail
				FOREIGN KEY (amr_id)
				REFERENCES "` + (&entity.AMREntity{}).TableName() + `"(amr_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END$$;
	`).Error; err != nil {
		slog.Error("Err migrate FK amr.amr_detail", "error", "ExecForeignKeys", "sub_method", "db.Exec()", "error", err.Error())
		os.Exit(1)
	}
	// FK: amr_detail_result.amr_detail_id → amr_detail.amr_detail_id
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_amr_detail_amr_detail_result'
			) THEN
				ALTER TABLE "` + (&entity.AMRDetailResultEntity{}).TableName() + `"
				ADD CONSTRAINT fk_amr_detail_amr_detail_result
				FOREIGN KEY (amr_detail_id)
				REFERENCES "` + (&entity.AMRDetailEntity{}).TableName() + `"(amr_detail_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT;
			END IF;
		END$$;
	`).Error; err != nil {
		slog.Error("Err migrate FK amr_detail.amr_detail_result", "error", "ExecForeignKeys", "sub_method", "db.Exec()", "error", err.Error())
		os.Exit(1)
	}

	print("END FK")
}
