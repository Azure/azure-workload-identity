// Code generated by MockGen. DO NOT EDIT.
// Source: ../azureclient.go

// Package mock_cloud is a generated GoMock package.
package mock_cloud

import (
	context "context"
	reflect "reflect"

	authorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	cloud "github.com/Azure/azure-workload-identity/pkg/cloud"
	gomock "github.com/golang/mock/gomock"
	graph "github.com/microsoftgraph/msgraph-sdk-go/models/microsoft/graph"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// AddFederatedCredential mocks base method.
func (m *MockInterface) AddFederatedCredential(ctx context.Context, objectID string, fc cloud.FederatedCredential) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddFederatedCredential", ctx, objectID, fc)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddFederatedCredential indicates an expected call of AddFederatedCredential.
func (mr *MockInterfaceMockRecorder) AddFederatedCredential(ctx, objectID, fc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddFederatedCredential", reflect.TypeOf((*MockInterface)(nil).AddFederatedCredential), ctx, objectID, fc)
}

// CreateApplication mocks base method.
func (m *MockInterface) CreateApplication(ctx context.Context, displayName string) (*graph.Application, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateApplication", ctx, displayName)
	ret0, _ := ret[0].(*graph.Application)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateApplication indicates an expected call of CreateApplication.
func (mr *MockInterfaceMockRecorder) CreateApplication(ctx, displayName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateApplication", reflect.TypeOf((*MockInterface)(nil).CreateApplication), ctx, displayName)
}

// CreateRoleAssignment mocks base method.
func (m *MockInterface) CreateRoleAssignment(ctx context.Context, scope, roleName, principalID string) (authorization.RoleAssignment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRoleAssignment", ctx, scope, roleName, principalID)
	ret0, _ := ret[0].(authorization.RoleAssignment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRoleAssignment indicates an expected call of CreateRoleAssignment.
func (mr *MockInterfaceMockRecorder) CreateRoleAssignment(ctx, scope, roleName, principalID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRoleAssignment", reflect.TypeOf((*MockInterface)(nil).CreateRoleAssignment), ctx, scope, roleName, principalID)
}

// CreateServicePrincipal mocks base method.
func (m *MockInterface) CreateServicePrincipal(ctx context.Context, appID string, tags []string) (*graph.ServicePrincipal, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateServicePrincipal", ctx, appID, tags)
	ret0, _ := ret[0].(*graph.ServicePrincipal)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateServicePrincipal indicates an expected call of CreateServicePrincipal.
func (mr *MockInterfaceMockRecorder) CreateServicePrincipal(ctx, appID, tags interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateServicePrincipal", reflect.TypeOf((*MockInterface)(nil).CreateServicePrincipal), ctx, appID, tags)
}

// DeleteApplication mocks base method.
func (m *MockInterface) DeleteApplication(ctx context.Context, objectID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteApplication", ctx, objectID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteApplication indicates an expected call of DeleteApplication.
func (mr *MockInterfaceMockRecorder) DeleteApplication(ctx, objectID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteApplication", reflect.TypeOf((*MockInterface)(nil).DeleteApplication), ctx, objectID)
}

// DeleteFederatedCredential mocks base method.
func (m *MockInterface) DeleteFederatedCredential(ctx context.Context, objectID, federatedCredentialID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFederatedCredential", ctx, objectID, federatedCredentialID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFederatedCredential indicates an expected call of DeleteFederatedCredential.
func (mr *MockInterfaceMockRecorder) DeleteFederatedCredential(ctx, objectID, federatedCredentialID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFederatedCredential", reflect.TypeOf((*MockInterface)(nil).DeleteFederatedCredential), ctx, objectID, federatedCredentialID)
}

// DeleteRoleAssignment mocks base method.
func (m *MockInterface) DeleteRoleAssignment(ctx context.Context, roleAssignmentID string) (authorization.RoleAssignment, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRoleAssignment", ctx, roleAssignmentID)
	ret0, _ := ret[0].(authorization.RoleAssignment)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteRoleAssignment indicates an expected call of DeleteRoleAssignment.
func (mr *MockInterfaceMockRecorder) DeleteRoleAssignment(ctx, roleAssignmentID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRoleAssignment", reflect.TypeOf((*MockInterface)(nil).DeleteRoleAssignment), ctx, roleAssignmentID)
}

// DeleteServicePrincipal mocks base method.
func (m *MockInterface) DeleteServicePrincipal(ctx context.Context, objectID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServicePrincipal", ctx, objectID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServicePrincipal indicates an expected call of DeleteServicePrincipal.
func (mr *MockInterfaceMockRecorder) DeleteServicePrincipal(ctx, objectID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServicePrincipal", reflect.TypeOf((*MockInterface)(nil).DeleteServicePrincipal), ctx, objectID)
}

// GetApplication mocks base method.
func (m *MockInterface) GetApplication(ctx context.Context, displayName string) (*graph.Application, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetApplication", ctx, displayName)
	ret0, _ := ret[0].(*graph.Application)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetApplication indicates an expected call of GetApplication.
func (mr *MockInterfaceMockRecorder) GetApplication(ctx, displayName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetApplication", reflect.TypeOf((*MockInterface)(nil).GetApplication), ctx, displayName)
}

// GetFederatedCredential mocks base method.
func (m *MockInterface) GetFederatedCredential(ctx context.Context, objectID, issuer, subject string) (cloud.FederatedCredential, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFederatedCredential", ctx, objectID, issuer, subject)
	ret0, _ := ret[0].(cloud.FederatedCredential)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFederatedCredential indicates an expected call of GetFederatedCredential.
func (mr *MockInterfaceMockRecorder) GetFederatedCredential(ctx, objectID, issuer, subject interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFederatedCredential", reflect.TypeOf((*MockInterface)(nil).GetFederatedCredential), ctx, objectID, issuer, subject)
}

// GetRoleDefinitionIDByName mocks base method.
func (m *MockInterface) GetRoleDefinitionIDByName(ctx context.Context, scope, roleName string) (authorization.RoleDefinition, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleDefinitionIDByName", ctx, scope, roleName)
	ret0, _ := ret[0].(authorization.RoleDefinition)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleDefinitionIDByName indicates an expected call of GetRoleDefinitionIDByName.
func (mr *MockInterfaceMockRecorder) GetRoleDefinitionIDByName(ctx, scope, roleName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleDefinitionIDByName", reflect.TypeOf((*MockInterface)(nil).GetRoleDefinitionIDByName), ctx, scope, roleName)
}

// GetServicePrincipal mocks base method.
func (m *MockInterface) GetServicePrincipal(ctx context.Context, displayName string) (*graph.ServicePrincipal, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServicePrincipal", ctx, displayName)
	ret0, _ := ret[0].(*graph.ServicePrincipal)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServicePrincipal indicates an expected call of GetServicePrincipal.
func (mr *MockInterfaceMockRecorder) GetServicePrincipal(ctx, displayName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServicePrincipal", reflect.TypeOf((*MockInterface)(nil).GetServicePrincipal), ctx, displayName)
}
