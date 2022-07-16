// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/Ahmed-Sermani/go-crawler/api/indexer/proto (interfaces: IndexerClient,Indexer_SearchClient)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	proto "github.com/Ahmed-Sermani/go-crawler/api/indexer/proto"
	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// MockIndexerClient is a mock of IndexerClient interface.
type MockIndexerClient struct {
	ctrl     *gomock.Controller
	recorder *MockIndexerClientMockRecorder
}

// MockIndexerClientMockRecorder is the mock recorder for MockIndexerClient.
type MockIndexerClientMockRecorder struct {
	mock *MockIndexerClient
}

// NewMockIndexerClient creates a new mock instance.
func NewMockIndexerClient(ctrl *gomock.Controller) *MockIndexerClient {
	mock := &MockIndexerClient{ctrl: ctrl}
	mock.recorder = &MockIndexerClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIndexerClient) EXPECT() *MockIndexerClientMockRecorder {
	return m.recorder
}

// Index mocks base method.
func (m *MockIndexerClient) Index(arg0 context.Context, arg1 *proto.Document, arg2 ...grpc.CallOption) (*proto.Document, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Index", varargs...)
	ret0, _ := ret[0].(*proto.Document)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Index indicates an expected call of Index.
func (mr *MockIndexerClientMockRecorder) Index(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Index", reflect.TypeOf((*MockIndexerClient)(nil).Index), varargs...)
}

// Search mocks base method.
func (m *MockIndexerClient) Search(arg0 context.Context, arg1 *proto.Query, arg2 ...grpc.CallOption) (proto.Indexer_SearchClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Search", varargs...)
	ret0, _ := ret[0].(proto.Indexer_SearchClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Search indicates an expected call of Search.
func (mr *MockIndexerClientMockRecorder) Search(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Search", reflect.TypeOf((*MockIndexerClient)(nil).Search), varargs...)
}

// UpdateScore mocks base method.
func (m *MockIndexerClient) UpdateScore(arg0 context.Context, arg1 *proto.UpdateScoreRequest, arg2 ...grpc.CallOption) (*emptypb.Empty, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateScore", varargs...)
	ret0, _ := ret[0].(*emptypb.Empty)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateScore indicates an expected call of UpdateScore.
func (mr *MockIndexerClientMockRecorder) UpdateScore(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateScore", reflect.TypeOf((*MockIndexerClient)(nil).UpdateScore), varargs...)
}

// MockIndexer_SearchClient is a mock of Indexer_SearchClient interface.
type MockIndexer_SearchClient struct {
	ctrl     *gomock.Controller
	recorder *MockIndexer_SearchClientMockRecorder
}

// MockIndexer_SearchClientMockRecorder is the mock recorder for MockIndexer_SearchClient.
type MockIndexer_SearchClientMockRecorder struct {
	mock *MockIndexer_SearchClient
}

// NewMockIndexer_SearchClient creates a new mock instance.
func NewMockIndexer_SearchClient(ctrl *gomock.Controller) *MockIndexer_SearchClient {
	mock := &MockIndexer_SearchClient{ctrl: ctrl}
	mock.recorder = &MockIndexer_SearchClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIndexer_SearchClient) EXPECT() *MockIndexer_SearchClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockIndexer_SearchClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockIndexer_SearchClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockIndexer_SearchClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockIndexer_SearchClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockIndexer_SearchClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockIndexer_SearchClient)(nil).Context))
}

// Header mocks base method.
func (m *MockIndexer_SearchClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockIndexer_SearchClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockIndexer_SearchClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockIndexer_SearchClient) Recv() (*proto.QueryResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*proto.QueryResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockIndexer_SearchClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockIndexer_SearchClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m *MockIndexer_SearchClient) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockIndexer_SearchClientMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockIndexer_SearchClient)(nil).RecvMsg), arg0)
}

// SendMsg mocks base method.
func (m *MockIndexer_SearchClient) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockIndexer_SearchClientMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockIndexer_SearchClient)(nil).SendMsg), arg0)
}

// Trailer mocks base method.
func (m *MockIndexer_SearchClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockIndexer_SearchClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockIndexer_SearchClient)(nil).Trailer))
}