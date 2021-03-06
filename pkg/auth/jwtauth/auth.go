package jwtauth

import (
	"context"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/wangwei518/gin-admin/pkg/auth"
)

const defaultKey = "gin-admin"

var defaultOptions = options{
	tokenType:     "Bearer",
	expired:       7200,
	signingMethod: jwt.SigningMethodHS512,
	signingKey:    []byte(defaultKey),
	keyfunc: func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, auth.ErrInvalidToken
		}
		return []byte(defaultKey), nil
	},
}

type options struct {
	signingMethod jwt.SigningMethod
	signingKey    interface{}
	keyfunc       jwt.Keyfunc
	expired       int
	tokenType     string
}

// Option 定义参数项
type Option func(*options)

// CustomClaims with user specified field
type CustomClaims struct {
	View string `json:"view"`
	jwt.StandardClaims
}

// SetSigningMethod 设定签名方式
func SetSigningMethod(method jwt.SigningMethod) Option {
	return func(o *options) {
		o.signingMethod = method
	}
}

// SetSigningKey 设定签名key
func SetSigningKey(key interface{}) Option {
	return func(o *options) {
		o.signingKey = key
	}
}

// SetKeyfunc 设定验证key的回调函数
func SetKeyfunc(keyFunc jwt.Keyfunc) Option {
	return func(o *options) {
		o.keyfunc = keyFunc
	}
}

// SetExpired 设定令牌过期时长(单位秒，默认7200)
func SetExpired(expired int) Option {
	return func(o *options) {
		o.expired = expired
	}
}

// New 创建认证实例
func New(store Storer, opts ...Option) *JWTAuth {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}

	return &JWTAuth{
		opts:  &o,
		store: store,
	}
}

// JWTAuth jwt认证
type JWTAuth struct {
	opts  *options
	store Storer
}

// GenerateToken 生成令牌
func (a *JWTAuth) GenerateToken(ctx context.Context, userID string, userView string) (auth.TokenInfo, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(a.opts.expired) * time.Second).Unix()

	//create claims with custom field
	claims := CustomClaims{
		userView, // custom: userView
		jwt.StandardClaims{
			IssuedAt:  now.Unix(), // JWT:iss
			ExpiresAt: expiresAt,  // JWT:exp
			NotBefore: now.Unix(), // JWT:nbf
			Subject:   userID,     // JWT:sub
		},
	}

	token := jwt.NewWithClaims(a.opts.signingMethod, claims)

	// Output example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MDAwLCJpc3MiOiJ0ZXN0In0.QsODzZu3lUZMVdhbO76u3Jv02iYCvEHcYVUI1kOWEU0 <nil>
	tokenString, err := token.SignedString(a.opts.signingKey)
	if err != nil {
		return nil, err
	}

	// return tokenInfo struct
	tokenInfo := &tokenInfo{
		ExpiresAt:   expiresAt,
		TokenType:   a.opts.tokenType,
		AccessToken: tokenString,
	}
	return tokenInfo, nil
}

// parseToken 解析令牌
func (a *JWTAuth) parseToken(tokenString string) (*CustomClaims, error) {

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, a.opts.keyfunc)
	if err != nil {
		return nil, err
	} else if !token.Valid {
		return nil, auth.ErrInvalidToken
	}

	return token.Claims.(*CustomClaims), nil
}

// bind method to store
func (a *JWTAuth) callStore(fn func(Storer) error) error {
	if store := a.store; store != nil {
		return fn(store)
	}
	return nil
}

// DestroyToken 销毁令牌
func (a *JWTAuth) DestroyToken(ctx context.Context, tokenString string) error {
	claims, err := a.parseToken(tokenString)
	if err != nil {
		return err
	}

	// 如果设定了存储，则将未过期的令牌放入
	return a.callStore(func(store Storer) error {
		expired := time.Unix(claims.ExpiresAt, 0).Sub(time.Now())
		return store.Set(ctx, tokenString, expired)
	})
}

// ParseUserID 解析用户ID
func (a *JWTAuth) ParseUserID(ctx context.Context, tokenString string) (string, string, error) {
	if tokenString == "" {
		return "", "", auth.ErrInvalidToken
	}

	claims, err := a.parseToken(tokenString)
	if err != nil {
		return "", "", err
	}

	// check blacklist
	err = a.callStore(func(store Storer) error {
		exists, err := store.Check(ctx, tokenString)
		if err != nil {
			return err
		} else if exists {
			return auth.ErrInvalidToken
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}

	return claims.Subject, claims.View, nil
}

// Release 释放资源
func (a *JWTAuth) Release() error {
	return a.callStore(func(store Storer) error {
		return store.Close()
	})
}
