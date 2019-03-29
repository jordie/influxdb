package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/influxdata/influxdb"
	pcontext "github.com/influxdata/influxdb/context"
	"github.com/influxdata/influxdb/mock"
	influxtesting "github.com/influxdata/influxdb/testing"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var (
	doc1ID   = influxtesting.MustIDBase16("020f755c3c082010")
	doc2ID   = influxtesting.MustIDBase16("020f755c3c082011")
	doc3ID   = influxtesting.MustIDBase16("020f755c3c082012")
	doc4ID   = influxtesting.MustIDBase16("020f755c3c082013")
	user1ID  = influxtesting.MustIDBase16("020f755c3c082001")
	label1ID = influxtesting.MustIDBase16("020f755c3c082300")
	label2ID = influxtesting.MustIDBase16("020f755c3c082301")
	label3ID = influxtesting.MustIDBase16("020f755c3c082302")
	label1   = influxdb.Label{
		ID:   label1ID,
		Name: "l1",
	}
	label2 = influxdb.Label{
		ID:   label2ID,
		Name: "l2",
	}
	label3 = influxdb.Label{
		ID:   label3ID,
		Name: "l3",
	}
	label3MappingJSON, _ = json.Marshal(influxdb.LabelMapping{
		LabelID: label3ID,
	})
	doc1 = influxdb.Document{
		ID: doc1ID,
		Meta: influxdb.DocumentMeta{
			Name: "doc1",
		},
		Content: "content1",
		Labels: []*influxdb.Label{
			&label1,
		},
	}
	doc2 = influxdb.Document{
		ID: doc2ID,
		Meta: influxdb.DocumentMeta{
			Name: "doc2",
		},
		Content: "content2",
		Labels:  []*influxdb.Label{},
	}
	doc3 = influxdb.Document{
		ID: doc3ID,
		Meta: influxdb.DocumentMeta{
			Name: "doc3",
		},
		Content: "content3",
		Labels: []*influxdb.Label{
			&label2,
		},
	}
	doc4 = influxdb.Document{
		ID: doc4ID,
		Meta: influxdb.DocumentMeta{
			Name: "doc4",
		},
		Content: "content4",
		Labels:  []*influxdb.Label{},
	}
	docs = []*influxdb.Document{
		&doc1,
		&doc2,
	}
	docsResp = `{
		"documents":[
			{
				"id": "020f755c3c082010",
				"links": {
					"self": "/api/v2/documents/template/020f755c3c082010"
				},
				"content": "content1",
				"labels": [
					   {
							"id": "020f755c3c082300",
							"name": "l1"
						}
				  ],
				"meta": {
					"name": "doc1"
				}
			},
			{
				"id": "020f755c3c082011",
				"links": {
					"self": "/api/v2/documents/template/020f755c3c082011"
				},
				"content": "content2", 
				"meta": {
					"name": "doc2"
				}
			}
		]
	}`
	findDocsServiceMock = &mock.DocumentService{
		FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
			return &mock.DocumentStore{
				FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
					return docs, nil
				},
			}, nil
		},
	}
	findDoc1ServiceMock = &mock.DocumentService{
		FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
			return &mock.DocumentStore{
				FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
					return []*influxdb.Document{&doc1}, nil
				},
			}, nil
		},
	}
	findDoc2ServiceMock = &mock.DocumentService{
		FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
			return &mock.DocumentStore{
				FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
					return []*influxdb.Document{&doc2}, nil
				},
			}, nil
		},
	}
)

// NewMockDocumentBackend returns a DocumentBackend with mock services.
func NewMockDocumentBackend() *DocumentBackend {
	return &DocumentBackend{
		Logger: zap.NewNop().With(zap.String("handler", "document")),

		DocumentService: mock.NewDocumentService(),
		LabelService:    mock.NewLabelService(),
	}
}

func TestService_handleDeleteDocumentLabel(t *testing.T) {
	type fields struct {
		DocumentService influxdb.DocumentService
		LabelService    influxdb.LabelService
	}
	type args struct {
		authorizer influxdb.Authorizer
		documentID influxdb.ID
		labelID    influxdb.ID
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "bad doc id",
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"url missing resource id"}`,
			},
		},
		{
			name: "bad label id",
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc1ID,
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"label id is missing"}`,
			},
		},
		{
			name: "label not found",
			fields: fields{
				DocumentService: findDoc2ServiceMock,
				LabelService: &mock.LabelService{
					FindLabelByIDFn: func(context.Context, influxdb.ID) (*influxdb.Label, error) {
						return nil, &influxdb.Error{
							Code: influxdb.ENotFound,
							Msg:  "label not found",
						}
					},
				},
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc2ID,
				labelID:    label1ID,
			},
			wants: wants{
				statusCode:  http.StatusNotFound,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"not found", "message":"label not found"}`,
			},
		},
		{
			name: "regular get labels",
			fields: fields{
				DocumentService: &mock.DocumentService{
					FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
						return &mock.DocumentStore{
							FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
								return []*influxdb.Document{&doc3}, nil
							},
							UpdateDocumentFn: func(ctx context.Context, d *influxdb.Document, opts ...influxdb.DocumentOptions) error {
								return nil
							},
						}, nil
					},
				},
				LabelService: &mock.LabelService{
					FindLabelByIDFn: func(context.Context, influxdb.ID) (*influxdb.Label, error) {
						return &label2, nil
					},
					DeleteLabelMappingFn: func(context.Context, *influxdb.LabelMapping) error {
						return nil
					},
				},
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc3ID,
				labelID:    label2ID,
			},
			wants: wants{
				statusCode: http.StatusNoContent,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documentBackend := NewMockDocumentBackend()
			documentBackend.DocumentService = tt.fields.DocumentService
			documentBackend.LabelService = tt.fields.LabelService
			h := NewDocumentHandler(documentBackend)
			r := httptest.NewRequest("DELETE", "http://any.url", nil)
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), tt.args.authorizer))
			r = r.WithContext(context.WithValue(r.Context(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "ns",
						Value: "template",
					},
					{
						Key:   "id",
						Value: tt.args.documentID.String(),
					},
					{
						Key:   "lid",
						Value: tt.args.labelID.String(),
					},
				}))
			w := httptest.NewRecorder()
			h.handleDeleteDocumentLabel(w, r)
			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleDeleteDocumentLabel() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleDeleteDocumentLabel() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); !eq {
				t.Errorf("%q. handleDeleteDocumentLabel() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handlePostDocumentLabel(t *testing.T) {
	type fields struct {
		DocumentService influxdb.DocumentService
		LabelService    influxdb.LabelService
	}
	type args struct {
		body       *bytes.Buffer
		authorizer influxdb.Authorizer
		documentID influxdb.ID
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "bad doc id",
			args: args{
				body: new(bytes.Buffer),
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"url missing id"}`,
			},
		},
		{
			name: "doc not found",
			fields: fields{
				DocumentService: &mock.DocumentService{
					FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
						return &mock.DocumentStore{
							FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
								return nil, &influxdb.Error{
									Code: influxdb.ENotFound,
									Msg:  "doc not found",
								}
							},
						}, nil
					},
				},
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc2ID,
				body:       new(bytes.Buffer),
			},
			wants: wants{
				statusCode:  http.StatusNotFound,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"not found", "message":"doc not found"}`,
			},
		},
		{
			name: "empty post a label",
			fields: fields{
				DocumentService: &mock.DocumentService{
					FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
						return &mock.DocumentStore{
							FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
								return []*influxdb.Document{&doc3}, nil
							},
							UpdateDocumentFn: func(ctx context.Context, d *influxdb.Document, opts ...influxdb.DocumentOptions) error {
								return nil
							},
						}, nil
					},
				},
				LabelService: &mock.LabelService{
					FindLabelByIDFn: func(context.Context, influxdb.ID) (*influxdb.Label, error) {
						return &label2, nil
					},
					DeleteLabelMappingFn: func(context.Context, *influxdb.LabelMapping) error {
						return nil
					},
				},
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc3ID,
				body:       new(bytes.Buffer),
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"Invalid post label map request"}`,
			},
		},
		{
			name: "regular post a label",
			fields: fields{
				DocumentService: &mock.DocumentService{
					FindDocumentStoreFn: func(context.Context, string) (influxdb.DocumentStore, error) {
						return &mock.DocumentStore{
							FindDocumentsFn: func(ctx context.Context, opts ...influxdb.DocumentFindOptions) ([]*influxdb.Document, error) {
								return []*influxdb.Document{&doc4}, nil
							},
							UpdateDocumentFn: func(ctx context.Context, d *influxdb.Document, opts ...influxdb.DocumentOptions) error {
								return nil
							},
						}, nil
					},
				},
				LabelService: &mock.LabelService{
					CreateLabelMappingFn: func(context.Context, *influxdb.LabelMapping) error {
						return nil
					},
					FindLabelByIDFn: func(context.Context, influxdb.ID) (*influxdb.Label, error) {
						return &label3, nil
					},
					DeleteLabelMappingFn: func(context.Context, *influxdb.LabelMapping) error {
						return nil
					},
				},
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc3ID,
				body:       bytes.NewBuffer(label3MappingJSON),
			},
			wants: wants{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
				body: `{"label": {
					    "id": "020f755c3c082302",
					    "name": "l3"
					  },
					  "links": {"self": "/api/v2/labels/020f755c3c082302"}}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documentBackend := NewMockDocumentBackend()
			documentBackend.DocumentService = tt.fields.DocumentService
			documentBackend.LabelService = tt.fields.LabelService
			h := NewDocumentHandler(documentBackend)
			r := httptest.NewRequest("POST", "http://any.url", tt.args.body)
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), tt.args.authorizer))
			r = r.WithContext(context.WithValue(r.Context(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "ns",
						Value: "template",
					},
					{
						Key:   "id",
						Value: tt.args.documentID.String(),
					},
				}))
			w := httptest.NewRecorder()
			h.handlePostDocumentLabel(w, r)
			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handlePostDocumentLabel() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handlePostDocumentLabel() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); !eq {
				t.Errorf("%q. handlePostDocumentLabel() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handleGetDocumentLabels(t *testing.T) {
	type fields struct {
		DocumentService influxdb.DocumentService
		LabelService    influxdb.LabelService
	}
	type args struct {
		queryParams map[string][]string
		authorizer  influxdb.Authorizer
		documentID  influxdb.ID
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "invalid document id",
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"url missing id"}`,
			},
		},
		{
			name: "regular get labels",
			fields: fields{
				DocumentService: findDoc1ServiceMock,
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc1ID,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body: `{"labels": [{
			"id": "020f755c3c082300",
			"name": "l1"
		}],"links":{"self":"/api/v2/labels"}}`},
		},
		{
			name: "find no labels",
			fields: fields{
				DocumentService: findDoc2ServiceMock,
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
				documentID: doc1ID,
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body:        `{"labels": [],"links":{"self":"/api/v2/labels"}}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documentBackend := NewMockDocumentBackend()
			documentBackend.DocumentService = tt.fields.DocumentService
			documentBackend.LabelService = tt.fields.LabelService
			h := NewDocumentHandler(documentBackend)
			r := httptest.NewRequest("GET", "http://any.url", nil)
			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), tt.args.authorizer))
			r = r.WithContext(context.WithValue(r.Context(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "ns",
						Value: "template",
					},
					{
						Key:   "id",
						Value: tt.args.documentID.String(),
					},
				}))
			w := httptest.NewRecorder()
			h.handleGetDocumentLabel(w, r)
			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetDocumentLabel() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetDocumentLabel() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetDocumentLabel() = ***%s***", tt.name, diff)
			}
		})
	}
}

func TestService_handleGetDocuments(t *testing.T) {
	type fields struct {
		DocumentService influxdb.DocumentService
	}
	type args struct {
		queryParams map[string][]string
		authorizer  influxdb.Authorizer
	}
	type wants struct {
		statusCode  int
		contentType string
		body        string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		wants  wants
	}{
		{
			name: "get all documents without org or orgID",
			fields: fields{
				DocumentService: findDocsServiceMock,
			},
			args: args{
				authorizer: &influxdb.Session{UserID: user1ID},
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"Please provide either org or orgID"}`,
			},
		},
		{
			name: "get all documents with both org and orgID",
			fields: fields{
				DocumentService: findDocsServiceMock,
			},
			args: args{
				queryParams: map[string][]string{
					"orgID": []string{"020f755c3c082002"},
					"org":   []string{"org1"},
				},
				authorizer: &influxdb.Session{UserID: user1ID},
			},
			wants: wants{
				statusCode:  http.StatusBadRequest,
				contentType: "application/json; charset=utf-8",
				body:        `{"code":"invalid", "message":"Please provide either org or orgID, not both"}`,
			},
		},
		{
			name: "get all documents with orgID",
			fields: fields{
				DocumentService: findDocsServiceMock,
			},
			args: args{
				queryParams: map[string][]string{
					"orgID": []string{"020f755c3c082002"},
				},
				authorizer: &influxdb.Session{UserID: user1ID},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body:        docsResp,
			},
		},
		{
			name: "get all documents with org name",
			fields: fields{
				DocumentService: findDocsServiceMock,
			},
			args: args{
				queryParams: map[string][]string{
					"org": []string{"org1"},
				},
				authorizer: &influxdb.Session{UserID: user1ID},
			},
			wants: wants{
				statusCode:  http.StatusOK,
				contentType: "application/json; charset=utf-8",
				body:        docsResp,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			documentBackend := NewMockDocumentBackend()
			documentBackend.DocumentService = tt.fields.DocumentService
			h := NewDocumentHandler(documentBackend)
			r := httptest.NewRequest("GET", "http://any.url", nil)
			qp := r.URL.Query()
			for k, vs := range tt.args.queryParams {
				for _, v := range vs {
					qp.Add(k, v)
				}
			}
			r.URL.RawQuery = qp.Encode()
			r = r.WithContext(pcontext.SetAuthorizer(r.Context(), tt.args.authorizer))
			r = r.WithContext(context.WithValue(r.Context(),
				httprouter.ParamsKey,
				httprouter.Params{
					{
						Key:   "ns",
						Value: "template",
					}}))
			w := httptest.NewRecorder()
			h.handleGetDocuments(w, r)
			res := w.Result()
			content := res.Header.Get("Content-Type")
			body, _ := ioutil.ReadAll(res.Body)

			if res.StatusCode != tt.wants.statusCode {
				t.Errorf("%q. handleGetDocuments() = %v, want %v", tt.name, res.StatusCode, tt.wants.statusCode)
			}
			if tt.wants.contentType != "" && content != tt.wants.contentType {
				t.Errorf("%q. handleGetDocuments() = %v, want %v", tt.name, content, tt.wants.contentType)
			}
			if eq, diff, _ := jsonEqual(string(body), tt.wants.body); tt.wants.body != "" && !eq {
				t.Errorf("%q. handleGetDocuments() = ***%s***", tt.name, diff)
			}
		})
	}
}
