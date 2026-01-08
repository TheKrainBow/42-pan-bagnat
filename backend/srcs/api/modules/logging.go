package modules

import (
	"backend/api/auth"
	"backend/core"
	"net/http"
)

func logSSHKeyUsage(r *http.Request, module core.Module, action string) {
	if module.SSHKeyID == "" {
		return
	}
	user, _ := r.Context().Value(auth.UserCtxKey).(*core.User)
	moduleID := module.ID
	msg := action
	_ = core.AppendSSHKeyEvent(module.SSHKeyID, user, &moduleID, msg)
}
