package definition

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	. "metis/internal/app/itsm/domain"
	"metis/internal/database"
	"metis/internal/handler"
	"metis/internal/model"
)

func TestKnowledgeDocServiceUploadAndListCompletedDocs(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	docSvc := newKnowledgeDocServiceForTest(t, db, serviceDefs)

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-doc", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	doc, err := docSvc.Upload(service.ID, "guide.md", int64(len("hello")), strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("uploads", "itsm", "knowledge", strconv.FormatUint(uint64(service.ID), 10))) })

	if doc.ParseStatus != "pending" || doc.FileType != "text/markdown" || !strings.HasSuffix(doc.FilePath, "guide.md") {
		t.Fatalf("unexpected doc after upload: %+v", doc)
	}
	if _, err := os.Stat(doc.FilePath); err != nil {
		t.Fatalf("expected uploaded file on disk: %v", err)
	}

	var tasks []model.TaskExecution
	if err := db.Where("task_name = ?", "itsm-doc-parse").Find(&tasks).Error; err != nil {
		t.Fatalf("list doc parse tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected one parse task, got %d", len(tasks))
	}

	if err := docSvc.repo.UpdateParseResult(doc.ID, "completed", "parsed hello", ""); err != nil {
		t.Fatalf("UpdateParseResult: %v", err)
	}
	docs, err := docSvc.List(service.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(docs) != 1 || docs[0].ID != doc.ID {
		t.Fatalf("unexpected list result: %+v", docs)
	}
	completed, err := docSvc.repo.ListCompletedByServiceID(service.ID)
	if err != nil {
		t.Fatalf("ListCompletedByServiceID: %v", err)
	}
	if len(completed) != 1 || completed[0].ParsedText != "parsed hello" {
		t.Fatalf("unexpected completed docs: %+v", completed)
	}
}

func TestKnowledgeDocServiceRejectsUnsupportedFileAndLargeUpload(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	docSvc := newKnowledgeDocServiceForTest(t, db, serviceDefs)

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-doc-2", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	if _, err := docSvc.Upload(service.ID, "bad.exe", 3, strings.NewReader("bad")); err == nil || !strings.Contains(err.Error(), "不支持的文件类型") {
		t.Fatalf("expected unsupported file error, got %v", err)
	}
	if _, err := docSvc.Upload(service.ID, "big.md", MaxFileSize+1, bytes.NewReader(make([]byte, 1))); err == nil || !strings.Contains(err.Error(), "文件大小超过限制") {
		t.Fatalf("expected max size error, got %v", err)
	}
}

func TestKnowledgeDocHandlerUploadListAndDelete(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	docSvc := newKnowledgeDocServiceForTest(t, db, serviceDefs)
	h := &KnowledgeDocHandler{svc: docSvc}

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-doc-handler", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("uploads", "itsm", "knowledge", strconv.FormatUint(uint64(service.ID), 10))) })

	buildMultipart := func(filename string, body string) (*bytes.Buffer, string) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := io.Copy(part, strings.NewReader(body)); err != nil {
			t.Fatalf("write multipart body: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}
		return &buf, writer.FormDataContentType()
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/services/:id/knowledge-documents", h.Upload)
	r.GET("/services/:id/knowledge-documents", h.List)
	r.DELETE("/services/:id/knowledge-documents/:docId", h.Delete)

	req := httptest.NewRequest(http.MethodGet, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Fatalf("empty list status=%d body=%s", rec.Code, rec.Body.String())
	}

	body, contentType := buildMultipart("guide.md", "hello")
	req = httptest.NewRequest(http.MethodPost, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", rec.Code, rec.Body.String())
	}
	var uploadResp handler.R
	if err := json.Unmarshal(rec.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploadResp.Code != 0 {
		t.Fatalf("unexpected upload response: %+v", uploadResp)
	}

	req = httptest.NewRequest(http.MethodGet, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "guide.md") {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}

	body, contentType = buildMultipart("bad.exe", "bad")
	req = httptest.NewRequest(http.MethodPost, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad ext status=%d body=%s", rec.Code, rec.Body.String())
	}

	body, contentType = buildMultipart("huge.md", strings.Repeat("x", MaxFileSize+1))
	req = httptest.NewRequest(http.MethodPost, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversize status=%d body=%s", rec.Code, rec.Body.String())
	}

	body, contentType = buildMultipart("guide.md", "hello")
	req = httptest.NewRequest(http.MethodPost, "/services/bad/knowledge-documents", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid service id upload status=%d body=%s", rec.Code, rec.Body.String())
	}

	body, contentType = buildMultipart("guide.md", "hello")
	req = httptest.NewRequest(http.MethodPost, "/services/999/knowledge-documents", body)
	req.Header.Set("Content-Type", contentType)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing service upload status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/services/bad/knowledge-documents", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid service id list status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/services/999/knowledge-documents", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing service list status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents/bad", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid doc id delete status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/services/999/knowledge-documents/1", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing service delete status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/services/"+strconv.FormatUint(uint64(service.ID), 10)+"/knowledge-documents/999", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing doc delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestKnowledgeDocRepoConstructorsAndParseFailure(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-doc-parse", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	injector := do.New()
	do.ProvideValue(injector, &database.DB{DB: db})
	do.ProvideValue(injector, serviceDefs)
	do.Provide(injector, NewKnowledgeDocRepo)
	do.Provide(injector, NewKnowledgeDocService)

	repo := do.MustInvoke[*KnowledgeDocRepo](injector)
	svc := do.MustInvoke[*KnowledgeDocService](injector)
	doc := &ServiceKnowledgeDocument{
		ServiceID:   service.ID,
		FileName:    "missing.md",
		FilePath:    filepath.Join("uploads", "itsm", "knowledge", "missing.md"),
		FileSize:    5,
		FileType:    "text/markdown",
		ParseStatus: "pending",
	}
	if err := repo.Create(doc); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := repo.GetByID(doc.ID)
	if err != nil || got.ID != doc.ID {
		t.Fatalf("GetByID=%+v err=%v", got, err)
	}
	if err := svc.ParseDocument(doc.ID); err == nil {
		t.Fatalf("expected ParseDocument to fail for missing file")
	}
	reloaded, err := repo.GetByID(doc.ID)
	if err != nil {
		t.Fatalf("reload doc: %v", err)
	}
	if reloaded.ParseStatus != "failed" || reloaded.ParseError == "" {
		t.Fatalf("expected failed parse result, got %+v", reloaded)
	}
	if err := repo.Delete(doc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestKnowledgeDocServiceParseDocumentCompletesTextFile(t *testing.T) {
	db := newTestDB(t)
	catSvc := newCatalogServiceForTest(t, db)
	serviceDefs := newServiceDefServiceForTest(t, db)
	docSvc := newKnowledgeDocServiceForTest(t, db, serviceDefs)

	root, err := catSvc.Create("Root", "root", "", "", nil, 10)
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	service, err := serviceDefs.Create(&ServiceDefinition{Name: "VPN", Code: "vpn-doc-parse-ok", CatalogID: root.ID, EngineType: "smart", CollaborationSpec: "spec"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}

	doc, err := docSvc.Upload(service.ID, "guide.txt", int64(len("hello knowledge doc")), strings.NewReader("hello knowledge doc"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("uploads", "itsm", "knowledge", strconv.FormatUint(uint64(service.ID), 10))) })

	if err := docSvc.ParseDocument(doc.ID); err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}

	reloaded, err := docSvc.repo.GetByID(doc.ID)
	if err != nil {
		t.Fatalf("reload parsed doc: %v", err)
	}
	if reloaded.ParseStatus != "completed" {
		t.Fatalf("parse status = %q, want completed", reloaded.ParseStatus)
	}
	if !strings.Contains(reloaded.ParsedText, "hello knowledge doc") || reloaded.ParseError != "" {
		t.Fatalf("unexpected parse result: %+v", reloaded)
	}
}
