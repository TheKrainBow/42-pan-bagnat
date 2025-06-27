package core

import (
	"backend/database"
	"fmt"
	"time"
)

func LogModule(moduleID, level, message string, err error) error {
	// Build the JSON meta payload
	meta := map[string]interface{}{}
	if err != nil {
		meta["error"] = err.Error()
	}

	database.InsertModuleLog(database.ModuleLog{
		ModuleID: moduleID,
		Level:    level,
		Message:  message,
		Meta:     meta,
	})

	ts := time.Now().Format(time.RFC3339)
	if err != nil {
		fmt.Printf("%s [%s] [module:%s] %s – error: %v\n",
			ts, level, moduleID, message, err,
		)
	} else {
		fmt.Printf("%s [%s] [module:%s] %s\n",
			ts, level, moduleID, message,
		)
	}

	// 3) Return the DB error (nil if success)
	if err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return nil
}
