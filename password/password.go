package password

import (
	"github.com/hiendv/gate"
	"github.com/pkg/errors"
)

// ErrForbidden is thrown when an user is forbidden to take an action on an object
var ErrForbidden = errors.New("forbidden")

// ErrNoAbilities is thrown when an user has no abilities
var ErrNoAbilities = errors.New("there is no abilities")

// LoginFunc is the handler of password-based authentication
type LoginFunc func(username, password string) (gate.User, error)

// Driver is password-based authentication
type Driver struct {
	config       gate.Config
	dependencies *gate.Dependencies
	handler      LoginFunc
}

// New is the constructor for Driver
func New(config gate.Config, dependencies *gate.Dependencies, handler LoginFunc) *Driver {
	jwtConfig, err := gate.NewHMACJWTConfig("HS256", config.JWTSigningKey(), config.JWTExpiration(), config.JWTSkipClaimsValidation())
	if err != nil {
		return nil
	}

	dependencies.SetJWTService(gate.NewJWTService(jwtConfig))
	dependencies.SetMatcher(gate.NewMatcher())
	return &Driver{config, dependencies, handler}
}

// GetConfig returns authentication configuration
func (auth Driver) GetConfig() gate.Config {
	return auth.config
}

// UserService returns user service from the dependencies or throws an error if the service is invalid
func (auth Driver) UserService() (gate.UserService, error) {
	if auth.dependencies == nil {
		return nil, errors.New("invalid dependencies")
	}

	if auth.dependencies.UserService() == nil {
		return nil, errors.New("invalid user service")
	}

	return auth.dependencies.UserService(), nil
}

// RoleService returns role service from the dependencies or throws an error if the service is invalid
func (auth Driver) RoleService() (gate.RoleService, error) {
	if auth.dependencies == nil {
		return nil, errors.New("invalid dependencies")
	}

	if auth.dependencies.RoleService() == nil {
		return nil, errors.New("invalid role service")
	}

	return auth.dependencies.RoleService(), nil
}

// TokenService returns token service from the dependencies or throws an error if the service is invalid
func (auth Driver) TokenService() (gate.TokenService, error) {
	if auth.dependencies == nil {
		return nil, errors.New("invalid dependencies")
	}

	if auth.dependencies.TokenService() == nil {
		return nil, errors.New("invalid token service")
	}

	return auth.dependencies.TokenService(), nil
}

// JWTService returns JWT service from the dependencies or throws an error if the service is invalid
func (auth Driver) JWTService() (gate.JWTService, error) {
	if auth.dependencies == nil {
		return gate.JWTService{}, errors.New("invalid dependencies")
	}

	return auth.dependencies.JWTService(), nil
}

// Matcher returns Matcher instance from the dependencies or throws an error if the instance is invalid
func (auth Driver) Matcher() (gate.Matcher, error) {
	if auth.dependencies == nil {
		return gate.Matcher{}, errors.New("invalid dependencies")
	}

	return auth.dependencies.Matcher(), nil
}

// Login resolves password-based authentication with the given handler and credentials
func (auth Driver) Login(credentials map[string]string) (user gate.User, err error) {
	username, ok := credentials["username"]
	if !ok {
		err = errors.New("missing username")
		return
	}

	password, ok := credentials["password"]
	if !ok {
		err = errors.New("missing password")
		return
	}

	user, err = auth.handler(username, password)
	if err != nil {
		err = errors.Wrap(err, "could not login")
	}
	return
}

// IssueJWT issues and stores a JWT for a specific user
func (auth Driver) IssueJWT(user gate.User) (token gate.JWT, err error) {
	service, err := auth.JWTService()
	if err != nil {
		return
	}

	claims := service.NewClaims(user)
	token, err = service.Issue(claims)
	if err != nil {
		err = errors.Wrap(err, "could not issue JWT")
		return
	}

	err = auth.StoreJWT(token)
	if err != nil {
		err = errors.Wrap(err, "could not store JWT")
	}
	return
}

// StoreJWT stores a JWT using the given token service
func (auth Driver) StoreJWT(token gate.JWT) (err error) {
	service, err := auth.TokenService()
	if err != nil {
		return
	}

	return service.Store(token)
}

// ParseJWT parses a JWT string to a JWT
func (auth Driver) ParseJWT(tokenString string) (token gate.JWT, err error) {
	service, err := auth.JWTService()
	if err != nil {
		return
	}

	token, err = service.Parse(tokenString)
	if err != nil {
		err = errors.Wrap(err, "could not parse token")
	}

	return
}

// Authenticate performs the authentication using JWT
func (auth Driver) Authenticate(tokenString string) (user gate.User, err error) {
	token, err := auth.ParseJWT(tokenString)
	if err != nil {
		err = errors.Wrap(err, "could not parse the token")
		return
	}

	user, err = auth.GetUserFromJWT(token)
	if err != nil {
		err = errors.Wrap(err, "could not get the user")
	}
	return
}

// Authorize performs the authorization when a given user takes an action on an object using RBAC
func (auth Driver) Authorize(user gate.User, action, object string) (err error) {
	abilities, err := auth.GetUserAbilities(user)
	if err != nil {
		err = errors.Wrap(err, "could not get the abilities")
		return
	}

	if len(abilities) == 0 {
		err = ErrNoAbilities
		return
	}

	if !auth.authorizationCheck(action, object, abilities) {
		err = ErrForbidden
	}
	return
}

// GetUserFromJWT returns a user from a given JWT
func (auth Driver) GetUserFromJWT(token gate.JWT) (user gate.User, err error) {
	service, err := auth.UserService()
	if err != nil {
		return
	}

	user, err = service.FindOneByID(token.UserID)
	if err != nil {
		err = errors.Wrap(err, "could not find the user with the given id")
	}
	return
}

// GetUserAbilities returns a user's abilities
func (auth Driver) GetUserAbilities(user gate.User) (abilities []gate.UserAbility, err error) {
	roleIDs := user.GetRoles()
	if len(roleIDs) == 0 {
		return
	}

	service, err := auth.RoleService()
	if err != nil {
		return
	}

	roles, err := service.FindByIDs(roleIDs)
	if err != nil {
		err = errors.Wrap(err, "could not fetch roles")
		return
	}

	for _, role := range roles {
		abilities = append(abilities, role.GetAbilities()...)
	}
	return
}

func (auth Driver) authorizationCheck(action, object string, abilities []gate.UserAbility) (found bool) {
	matcher, err := auth.Matcher()
	if err != nil {
		return
	}

	for _, ability := range abilities {
		if ability.GetAction() == "" {
			continue
		}

		if ability.GetObject() == "" {
			continue
		}

		actionMatch, err := matcher.Match(action, ability.GetAction())
		if err != nil || !actionMatch {
			continue
		}

		objectMatch, err := matcher.Match(object, ability.GetObject())
		if err != nil || !objectMatch {
			continue
		}

		found = true
		break
	}

	return
}
