package main

import (
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLocalDataSurvivesRestart(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", "default_state.json"), []byte(`{"version":1,"orders":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Server{root: root, eventClients: map[chan int64]bool{}}
	if err := s.loadState(); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.state["testOrder"] = "persisted"
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	s2 := &Server{root: root}
	if err := s2.loadState(); err != nil {
		t.Fatal(err)
	}
	if s2.state["testOrder"] != "persisted" || s2.version != 2 {
		t.Fatal("data lost after restart")
	}
}
func TestDatabaseWriteFailureRestoresCommittedState(t *testing.T) {
	for _, failure := range []string{"outage", "conflict"} {
		t.Run(failure, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			s := &Server{db: db, root: t.TempDir(), version: 7, dbVersion: 7, state: map[string]interface{}{"version": 7, "order": "changed"}, committedState: []byte(`{"version":7,"order":"saved"}`)}
			ex := mock.ExpectExec("UPDATE imfohsa_private.documents").WithArgs(sqlmock.AnyArg(), int64(8), int64(7))
			if failure == "outage" {
				ex.WillReturnError(errors.New("connection refused"))
			} else {
				ex.WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery("SELECT payload, version").WillReturnRows(sqlmock.NewRows([]string{"payload", "version"}).AddRow([]byte(`{"version":7,"order":"saved"}`), 7))
			}
			if err := s.saveLocked(); err == nil {
				t.Fatal("failure ignored")
			}
			if s.state["order"] != "saved" || s.version != 7 {
				t.Fatal("unsaved changes retained")
			}
			if _, err := os.Stat(s.statePath()); !os.IsNotExist(err) {
				t.Fatal("database failure fell back to disk")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestDatabaseWriteCommitsVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, version: 7, dbVersion: 7, state: map[string]interface{}{"version": 7, "order": "changed"}, eventClients: map[chan int64]bool{}}
	mock.ExpectExec("UPDATE imfohsa_private.documents").WithArgs(sqlmock.AnyArg(), int64(8), int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	s.mu.Lock()
	err = s.saveLocked()
	s.mu.Unlock()
	if err != nil || s.version != 8 || s.dbVersion != 8 {
		t.Fatalf("version not committed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestExistingDatabaseStateWinsOverLocalSeed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, root: t.TempDir()}
	mock.ExpectQuery("SELECT payload").WithArgs("state").WillReturnRows(sqlmock.NewRows([]string{"payload"}).AddRow([]byte(`{"version":42}`)))
	b, err := s.loadDocument("state", s.statePath())
	if err != nil || string(b) != `{"version":42}` {
		t.Fatalf("database seed replaced: %s %v", b, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestAdminLookupFailureDoesNotGrantManagerRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db}
	mock.ExpectQuery("SELECT payload").WithArgs("admins").WillReturnError(errors.New("outage"))
	if role := s.roleForUser("unknown"); managerRole(role) {
		t.Fatal("database outage elevated privileges")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestStoredUploadCanBeServedAfterLocalFileRemoved(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, root: t.TempDir()}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT content FROM imfohsa_private.uploads WHERE name=$1")).WithArgs("photo.png").WillReturnRows(sqlmock.NewRows([]string{"content"}).AddRow([]byte("persisted photo")))
	w := httptest.NewRecorder()
	s.uploadsServe(w, httptest.NewRequest("GET", "/uploads/photo.png", nil))
	if w.Code != 200 || w.Body.String() != "persisted photo" {
		t.Fatal("database upload unavailable")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestPlaceholderCredentialsAreRejectedWithoutDisclosure(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://postgres:[YOUR-PASSWORD]@db.pzeyfhgjojwskriwbeam.supabase.co/postgres")
	if err := (&Server{}).openDatabase(); err == nil {
		t.Fatal("placeholder accepted")
	}
}
