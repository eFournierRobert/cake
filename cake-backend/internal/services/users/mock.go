package users

import (
	"efournierrobert/cake-backend/internal/handlers/models"
)

// MockService is a hand-written test double for UserService, used by
// the HTTP handler tests.
//
// It deliberately lives in a regular (non _test.go) file so that
// tests in other packages, such as internal/handlers/users, can
// import it.
//
// Each method has a Func field holding a handler for calls: when it
// is set, the method forwards its arguments to it; when it is nil,
// the method returns a zero value and a nil error, so tests only
// need to configure the methods they exercise. Every call also
// records the arguments it received in the matching *Arg fields, so
// tests can assert on what the handler forwarded.
type MockService struct {
	GetUserFunc func(strUuid string) (models.UserDto, error)
	GetUserArg  string

	ModifyUserFunc      func(currentUserUuid string, userUpdate models.UserUpdate) (models.UserDto, error)
	ModifyUserUuidArg   string
	ModifyUserUpdateArg models.UserUpdate

	AdminModifyUserFunc      func(userUuid string, userUpdate models.AdminUserUpdate) (models.UserDto, error)
	AdminModifyUserUuidArg   string
	AdminModifyUserUpdateArg models.AdminUserUpdate

	ChangePasswordFunc    func(currentUserUuid string, newPassword string) error
	ChangePasswordUuidArg string
	ChangePasswordNewArg  string

	GetAllUsersFunc func() ([]models.UserDto, error)

	CreateUserFunc func(userCreateDto models.UserCreate) (models.UserDto, error)
	CreateUserArg  models.UserCreate

	DeleteUserFunc func(userUuid string) error
	DeleteUserArg  string

	LoginFunc        func(username, password string) (string, error)
	LoginUsernameArg string
	LoginPasswordArg string
}

var _ UserService = (*MockService)(nil)

func (m *MockService) GetUser(strUuid string) (models.UserDto, error) {
	m.GetUserArg = strUuid
	if m.GetUserFunc == nil {
		return models.UserDto{}, nil
	}
	return m.GetUserFunc(strUuid)
}

func (m *MockService) ModifyUser(currentUserUuid string, userUpdate models.UserUpdate) (models.UserDto, error) {
	m.ModifyUserUuidArg = currentUserUuid
	m.ModifyUserUpdateArg = userUpdate
	if m.ModifyUserFunc == nil {
		return models.UserDto{}, nil
	}
	return m.ModifyUserFunc(currentUserUuid, userUpdate)
}

func (m *MockService) AdminModifyUser(userUuid string, userUpdate models.AdminUserUpdate) (models.UserDto, error) {
	m.AdminModifyUserUuidArg = userUuid
	m.AdminModifyUserUpdateArg = userUpdate
	if m.AdminModifyUserFunc == nil {
		return models.UserDto{}, nil
	}
	return m.AdminModifyUserFunc(userUuid, userUpdate)
}

func (m *MockService) ChangePassword(currentUserUuid string, newPassword string) error {
	m.ChangePasswordUuidArg = currentUserUuid
	m.ChangePasswordNewArg = newPassword
	if m.ChangePasswordFunc == nil {
		return nil
	}
	return m.ChangePasswordFunc(currentUserUuid, newPassword)
}

func (m *MockService) GetAllUsers() ([]models.UserDto, error) {
	if m.GetAllUsersFunc == nil {
		return nil, nil
	}
	return m.GetAllUsersFunc()
}

func (m *MockService) CreateUser(userCreateDto models.UserCreate) (models.UserDto, error) {
	m.CreateUserArg = userCreateDto
	if m.CreateUserFunc == nil {
		return models.UserDto{}, nil
	}
	return m.CreateUserFunc(userCreateDto)
}

func (m *MockService) DeleteUser(userUuid string) error {
	m.DeleteUserArg = userUuid
	if m.DeleteUserFunc == nil {
		return nil
	}
	return m.DeleteUserFunc(userUuid)
}

func (m *MockService) Login(username, password string) (string, error) {
	m.LoginUsernameArg = username
	m.LoginPasswordArg = password
	if m.LoginFunc == nil {
		return "", nil
	}
	return m.LoginFunc(username, password)
}
