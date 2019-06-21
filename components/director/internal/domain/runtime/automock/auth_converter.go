// Code generated by mockery v1.0.0. DO NOT EDIT.
package automock

import graphql "github.com/kyma-incubator/compass/components/director/internal/graphql"
import mock "github.com/stretchr/testify/mock"
import model "github.com/kyma-incubator/compass/components/director/internal/model"

// AuthConverter is an autogenerated mock type for the AuthConverter type
type AuthConverter struct {
	mock.Mock
}

// ToGraphQL provides a mock function with given fields: in
func (_m *AuthConverter) ToGraphQL(in *model.Auth) *graphql.Auth {
	ret := _m.Called(in)

	var r0 *graphql.Auth
	if rf, ok := ret.Get(0).(func(*model.Auth) *graphql.Auth); ok {
		r0 = rf(in)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*graphql.Auth)
		}
	}

	return r0
}
