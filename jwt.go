package gate

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// JWTService is the service which manages JWTs
type JWTService struct {
	config           JWTConfig
	Now              func() time.Time
	GenerateClaimsID func() string
}

// JWTConfig is the configuration for JWT service
type JWTConfig struct {
	method               jwt.SigningMethod
	signKey              interface{}
	verifyKey            interface{}
	expiration           time.Duration
	skipClaimsValidation bool
}

// JWTClaims are JWT claims with user's information
type JWTClaims struct {
	User UserInfo `json:"user"`
	jwt.StandardClaims
}

// JWT is the JSON Web Token
type JWT struct {
	ID        string
	Value     string
	UserID    string
	ExpiredAt time.Time
	IssuedAt  time.Time
}

// NewHMACJWTConfig is the constructor for JWTConfig using HMAC signing method
func NewHMACJWTConfig(alg string, key interface{}, expiration time.Duration, skipClaimsValidation bool) (config JWTConfig, err error) {
	method := jwt.GetSigningMethod(alg)
	if method == nil {
		err = errors.New("invalid JWT algorithm")
		return
	}

	config = JWTConfig{method, key, key, expiration, skipClaimsValidation}
	return
}

// NewJWTService is the constructor for JWTService
func NewJWTService(config JWTConfig) JWTService {
	return JWTService{
		config,
		func() time.Time {
			return time.Now().Local()
		},
		func() string {
			return uuid.NewV4().String()
		},
	}
}

// NewTokenFromClaims constructs a token from JWT claims
func (service JWTService) NewTokenFromClaims(claims JWTClaims) (token JWT) {
	token.ID = claims.Id
	token.UserID = claims.User.ID
	token.ExpiredAt = time.Unix(claims.ExpiresAt, 0)
	token.IssuedAt = time.Unix(claims.IssuedAt, 0)
	return
}

// Issue generates a token from JWT claims with the service configuration
func (service JWTService) Issue(claims JWTClaims) (token JWT, err error) {
	obj := jwt.NewWithClaims(service.config.method, claims)
	if obj == nil {
		err = errors.New("could not create JWT")
		return
	}

	key, err := service.getSigningKey()
	if err != nil {
		err = errors.Wrap(err, "could not sign JWT")
		return
	}

	str, err := obj.SignedString(key)
	if err != nil {
		err = errors.Wrap(err, "could not sign JWT")
		return
	}

	token = service.NewTokenFromClaims(claims)
	token.Value = str
	return
}

// Parse resolves a token string to a JWT with the service configuration
func (service JWTService) Parse(tokenString string) (token JWT, err error) {
	parser := new(jwt.Parser)
	parser.SkipClaimsValidation = service.config.skipClaimsValidation
	obj, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, service.getVerifyingKey)
	if err != nil {
		err = errors.Wrap(err, "could not parse JWT")
		return
	}

	if !obj.Valid {
		err = errors.New("invalid JWT")
		return
	}

	claims, ok := obj.Claims.(*JWTClaims)
	if !ok {
		err = errors.New("invalid claims")
		return
	}

	if claims == nil {
		err = errors.New("invalid claims")
		return
	}

	token = service.NewTokenFromClaims(*claims)
	token.Value = tokenString
	return
}

func (service JWTService) getSigningKey() (key interface{}, err error) {
	switch service.config.method.(type) {
	default:
		err = errors.New("invalid key")
		return
	case *jwt.SigningMethodHMAC:
		keyStr, ok := service.config.signKey.(string)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = []byte(keyStr)
	case *jwt.SigningMethodRSA:
		keyRSA, ok := service.config.signKey.(*rsa.PrivateKey)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = keyRSA
	case *jwt.SigningMethodRSAPSS:
		keyRSA, ok := service.config.signKey.(*rsa.PrivateKey)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = keyRSA
	case *jwt.SigningMethodECDSA:
		keyECDSA, ok := service.config.signKey.(*ecdsa.PrivateKey)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = keyECDSA
	}

	return
}

func (service JWTService) getVerifyingKey(token *jwt.Token) (key interface{}, err error) {
	switch service.config.method.(type) {
	default:
		err = errors.New("invalid algorithm")
		return
	case *jwt.SigningMethodHMAC:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			err = errors.Errorf("unexpected signing method: %v", token.Header["alg"])
			return
		}

		keyStr, ok := service.config.verifyKey.(string)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = []byte(keyStr)

	case *jwt.SigningMethodRSA:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			err = errors.Errorf("unexpected signing method: %v", token.Header["alg"])
			return
		}

		keyRSA, ok := service.config.verifyKey.(*rsa.PrivateKey)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = keyRSA

	case *jwt.SigningMethodRSAPSS:
		if _, ok := token.Method.(*jwt.SigningMethodRSAPSS); !ok {
			err = errors.Errorf("unexpected signing method: %v", token.Header["alg"])
			return
		}

		keyRSA, ok := service.config.verifyKey.(*rsa.PrivateKey)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = keyRSA

	case *jwt.SigningMethodECDSA:
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			err = errors.Errorf("unexpected signing method: %v", token.Header["alg"])
			return
		}

		keyECDSA, ok := service.config.verifyKey.(*ecdsa.PrivateKey)
		if !ok {
			err = errors.New("invalid key")
			return
		}

		key = keyECDSA
	}

	return key, nil
}

// NewClaims generates JWTClaims for a specific user
func (service JWTService) NewClaims(user User) JWTClaims {
	return JWTClaims{
		User: UserInfo{
			ID:       user.GetID(),
			Username: user.GetUsername(),
			Roles:    user.GetRoles(),
		},
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: service.Now().Add(service.config.expiration).Unix(),
			IssuedAt:  service.Now().Unix(),
			Id:        service.GenerateClaimsID(),
		},
	}
}
