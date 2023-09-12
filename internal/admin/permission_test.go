package admin

import (
	"reflect"
	"testing"
)

func Test_GetPermissions(t *testing.T) {
	tests := []struct {
		name        string
		user        *User
		authContext AuthContext
		wantFunc    func() []ResourcePermission // Function that returns expected slice
	}{
		{
			name: "User is admin",
			user: &User{IsAdmin: ptr(true)},
			authContext: &AuthContextMock{
				IsAdminFunc: func() bool {
					return true
				},
			},
			wantFunc: getAdminPermissions, // Reference the original function
		},
		{
			name: "User is not admin",
			user: &User{IsAdmin: ptr(false)},
			authContext: &AuthContextMock{
				IsAdminFunc: func() bool {
					return false
				},
			},
			wantFunc: getUserPermissions, // Reference the original function
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPermissions(tt.authContext)
			want := tt.wantFunc() // Get the expected data using the function

			if !reflect.DeepEqual(got, want) {
				t.Errorf("GetPermissions() = %v, want %v", got, want)
			}
		})
	}
}
