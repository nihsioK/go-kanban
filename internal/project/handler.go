package project

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/nihsioK/go-kanban/internal/app"
)

// Create godoc
// @Summary Create a new project
// @Description Create a new project for the authenticated user
// @Tags projects
// @Accept json
// @Produce json
// @Param project body Project true "Project details"
// @Success 201 {object} Project
// @Failure 400 {object} app.ErrorResponse
// @Failure 500 {object} app.ErrorResponse
// @Security BearerAuth
// @Router /projects [post]
func Create(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var project Project
		if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
			app.RespondWithError(w, http.StatusBadRequest, "Invalid payload")
			return
		}

		claims := r.Context().Value("claims").(*app.Claims)
		project.UserID = claims.ID

		err := a.DB.QueryRow(`INSERT INTO projects 
			(user_id, name, repo_url, site_url, description, dependencies, dev_dependencies, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			project.UserID, project.Name, project.RepoURL, project.SiteURL,
			project.Description, pq.Array(project.Dependencies), pq.Array(project.DevDependencies), "active",
		).Scan(&project.ID)

		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Failed to create project")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
	}
}

// Update godoc
// @Summary Update an existing project
// @Description Update the details of an existing project
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Param project body Project true "Updated project details"
// @Success 200 {object} Project
// @Failure 400 {object} app.ErrorResponse
// @Failure 404 {object} app.ErrorResponse
// @Failure 403 {object} app.ErrorResponse
// @Failure 500 {object} app.ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [put]
func Update(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		var project Project

		if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
			app.RespondWithError(w, http.StatusBadRequest, "Invalid payload")
			return
		}

		claims := r.Context().Value("claims").(*app.Claims)

		var storedUserID string
		if err := a.DB.QueryRow("SELECT user_id FROM projects WHERE id=$1", id).Scan(&storedUserID); err != nil {
			if err == sql.ErrNoRows {
				app.RespondWithError(w, http.StatusNotFound, "Project not found")
			} else {
				app.RespondWithError(w, http.StatusInternalServerError, "Query error")
			}
			return
		}
		if storedUserID != claims.ID {
			app.RespondWithError(w, http.StatusForbidden, "Not authorized")
			return
		}

		_, err := a.DB.Exec(`UPDATE projects 
			SET name=$1, repo_url=$2, site_url=$3, description=$4, dependencies=$5, dev_dependencies=$6, status=$7
			WHERE id=$8 AND user_id=$9`,
			project.Name, project.RepoURL, project.SiteURL, project.Description,
			pq.Array(project.Dependencies), pq.Array(project.DevDependencies), project.Status, id, claims.ID,
		)

		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Update failed")
			return
		}

		project.ID = id
		project.UserID = claims.ID
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
	}
}

// GetAll godoc
// @Summary Get all projects for the authenticated user
// @Description Retrieve all projects for the authenticated user
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {array} Project
// @Failure 500 {object} app.ErrorResponse
// @Security BearerAuth
// @Router /projects [get]
func GetAll(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value("claims").(*app.Claims)
		rows, err := a.DB.Query(`SELECT id, user_id, name, repo_url, site_url, description, dependencies, dev_dependencies, status 
			FROM projects WHERE user_id=$1`, claims.ID)
		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch projects")
			return
		}
		defer rows.Close()

		var projects []Project
		for rows.Next() {
			var p Project
			if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.SiteURL, &p.Description,
				pq.Array(&p.Dependencies), pq.Array(&p.DevDependencies), &p.Status); err != nil {
				app.RespondWithError(w, http.StatusInternalServerError, "Scan error")
				return
			}
			projects = append(projects, p)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}
}

// GetOne godoc
// @Summary Get a single project by ID
// @Description Retrieve a single project by its ID for the authenticated user
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Success 200 {object} Project
// @Failure 404 {object} app.ErrorResponse
// @Failure 403 {object} app.ErrorResponse
// @Failure 500 {object} app.ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [get]
func GetOne(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		claims := r.Context().Value("claims").(*app.Claims)

		var p Project
		err := a.DB.QueryRow(`SELECT id, user_id, name, repo_url, site_url, description, dependencies, dev_dependencies, status 
			FROM projects WHERE id=$1 AND user_id=$2`, id, claims.ID).
			Scan(&p.ID, &p.UserID, &p.Name, &p.RepoURL, &p.SiteURL, &p.Description,
				pq.Array(&p.Dependencies), pq.Array(&p.DevDependencies), &p.Status)

		if err != nil {
			if err == sql.ErrNoRows {
				app.RespondWithError(w, http.StatusNotFound, "Project not found")
			} else {
				app.RespondWithError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	}
}

// Delete godoc
// @Summary Delete a project
// @Description Delete a project by ID for the authenticated user
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID"
// @Success 204
// @Failure 404 {object} app.ErrorResponse
// @Failure 403 {object} app.ErrorResponse
// @Failure 500 {object} app.ErrorResponse
// @Security BearerAuth
// @Router /projects/{id} [delete]
func Delete(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		claims := r.Context().Value("claims").(*app.Claims)

		var owner string
		err := a.DB.QueryRow("SELECT user_id FROM projects WHERE id=$1", id).Scan(&owner)
		if err != nil {
			if err == sql.ErrNoRows {
				app.RespondWithError(w, http.StatusNotFound, "Project not found")
			} else {
				app.RespondWithError(w, http.StatusInternalServerError, "Error checking ownership")
			}
			return
		}
		if owner != claims.ID {
			app.RespondWithError(w, http.StatusForbidden, "Not authorized to delete")
			return
		}

		_, err = a.DB.Exec("DELETE FROM projects WHERE id=$1 AND user_id=$2", id, claims.ID)
		if err != nil {
			app.RespondWithError(w, http.StatusInternalServerError, "Delete failed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	}
}
