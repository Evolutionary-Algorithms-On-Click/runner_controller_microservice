package controller

import (
	"encoding/json"
	"evolve/db/connection"
	"evolve/modules"
	"evolve/util"
	"fmt"
	"net/http"
	"os"
)

func CreateBO(res http.ResponseWriter, req *http.Request) {
	logger := util.NewLogger()
	logger.Info("CreateBO API called.")

	// -----------------------------
	// AUTHENTICATE USER
	// -----------------------------
	user, err := modules.Auth(req)
	if err != nil {
		util.JSONResponse(res, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	// user := map[string]any{
	// 	"email":    "local@test.com",
	// 	"fullName": "Local Test User",
	// 	"id":       "00000000-0000-0000-0000-000000000000",
	// 	"role":     "user",
	// 	"userName": "localtester",
	// }
	logger.Info(fmt.Sprintf("User: %v", user))

	// -----------------------------
	// PARSE JSON BODY
	// -----------------------------
	data, err := util.Body(req)
	if err != nil {
		util.JSONResponse(res, http.StatusBadRequest, err.Error(), nil)
		return
	}

	bo, err := modules.BOFromJSON(data)
	if err != nil {
		util.JSONResponse(res, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// -----------------------------
	// GENERATE PYTHON CODE
	// -----------------------------
	code, err := bo.Code()
	if err != nil {
		util.JSONResponse(res, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// -----------------------------
	// CONNECT DB
	// -----------------------------
	db, err := connection.PoolConn(req.Context())
	if err != nil {
		logger.Error(fmt.Sprintf("CreateBO.PoolConn: %s", err.Error()))
		util.JSONResponse(res, http.StatusInternalServerError, "something went wrong", nil)
		return
	}

	// -----------------------------
	// INSERT RUN METADATA
	// -----------------------------
	runName := fmt.Sprintf("BO-%dD", len(bo.Bounds))

	row := db.QueryRow(req.Context(), `
		INSERT INTO run (name, description, type, command, createdBy)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, runName, "Bayesian Optimization", "bo", "python code.py", user["id"])

	var runID string
	err = row.Scan(&runID)
	if err != nil {
		logger.Error(fmt.Sprintf("CreateBO.row.Scan: %s", err.Error()))
		util.JSONResponse(res, http.StatusInternalServerError, "something went wrong", nil)
		return
	}

	logger.Info(fmt.Sprintf("RunID: %s", runID))

	// -----------------------------
	// ACCESS TABLE ENTRY
	// -----------------------------
	_, err = db.Exec(req.Context(), `
		INSERT INTO access (runID, userID, mode)
		VALUES ($1, $2, $3)
	`, runID, user["id"], "write")

	if err != nil {
		logger.Error(fmt.Sprintf("CreateBO.db.Exec: %s", err.Error()))
		util.JSONResponse(res, http.StatusInternalServerError, "something went wrong", nil)
		return
	}

	// -----------------------------
	// UPLOAD PYTHON CODE TO MINIO
	// -----------------------------
	os.Mkdir("code", 0755)
	pythonPath := fmt.Sprintf("code/%v.py", runID)

	if err := os.WriteFile(pythonPath, []byte(code), 0644); err != nil {
		logger.Error(fmt.Sprintf("CreateBO.WriteFile: %s", err.Error()))
		util.JSONResponse(res, http.StatusInternalServerError, "something went wrong", nil)
		return
	}
	if err := util.UploadFile(req.Context(), runID, "code", "py"); err != nil {
		util.JSONResponse(res, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// -----------------------------
	// UPLOAD INPUT JSON
	// -----------------------------
	inputParams, err := json.Marshal(data)
	if err != nil {
		logger.Error(fmt.Sprintf("CreateBO.Marshal: %s", err.Error()))
		util.JSONResponse(res, http.StatusInternalServerError, "something went wrong", nil)
		return
	}

	os.Mkdir("input", 0755)
	inputPath := fmt.Sprintf("input/%v.json", runID)

	if err := os.WriteFile(inputPath, inputParams, 0644); err != nil {
		logger.Error(fmt.Sprintf("CreateBO.WriteFile(input): %s", err.Error()))
		util.JSONResponse(res, http.StatusInternalServerError, "something went wrong", nil)
		return
	}
	if err := util.UploadFile(req.Context(), runID, "input", "json"); err != nil {
		util.JSONResponse(res, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// -----------------------------
	// CLEAN UP LOCAL FILES
	// -----------------------------
	os.Remove(pythonPath)
	os.Remove(inputPath)

	// -----------------------------
	// PUSH TO REDIS QUEUE
	// -----------------------------
	if err := util.EnqueueRunRequest(req.Context(), runID, "code", "py"); err != nil {
		util.JSONResponse(res, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	// Attach runID to response
	data["runID"] = runID

	util.JSONResponse(res, http.StatusOK, "BO Created Successfully 🎉", data)
}
