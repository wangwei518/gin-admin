package bll

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"sort"

	"github.com/LyricTian/captcha"
	"github.com/wangwei518/gin-admin/internal/app/bll"
	"github.com/wangwei518/gin-admin/internal/app/model"
	"github.com/wangwei518/gin-admin/internal/app/schema"
	"github.com/wangwei518/gin-admin/pkg/auth"
	"github.com/wangwei518/gin-admin/pkg/errors"
	"github.com/wangwei518/gin-admin/pkg/util"
	"github.com/google/wire"
)

var _ bll.ILogin = (*Login)(nil)

// LoginSet 注入Login
var LoginSet = wire.NewSet(wire.Struct(new(Login), "*"), wire.Bind(new(bll.ILogin), new(*Login)))

// Login 登录管理
type Login struct {
	Auth            auth.Auther
	UserModel       model.IUser
	UserRoleModel   model.IUserRole
	RoleModel       model.IRole
	RoleMenuModel   model.IRoleMenu
	MenuModel       model.IMenu
	MenuActionModel model.IMenuAction
}

// GetCaptcha 获取图形验证码信息
/*func (a *Login) GetCaptcha(ctx context.Context, length int) (*schema.LoginCaptcha, error) {
	captchaID := captcha.NewLen(length)
	item := &schema.LoginCaptcha{
		CaptchaID: captchaID,
	}
	return item, nil
}*/

// ResCaptcha 生成并响应图形验证码
/*func (a *Login) ResCaptcha(ctx context.Context, w http.ResponseWriter, captchaID string, width, height int) error {
	err := captcha.WriteImage(w, captchaID, width, height)
	if err != nil {
		if err == captcha.ErrNotFound {
			return errors.ErrNotFound
		}
		return errors.WithStack(err)
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "image/png")
	return nil
}*/

// Verify 登录验证
func (a *Login) Verify(ctx context.Context, userName, password string) (*schema.User, error) {
	// 检查是否是超级用户
	root := GetRootUser()
	if userName == root.UserName && root.Password == password {
		return root, nil
	}

	// 从数据库搜索用户信息并验证（drop）
/*	result, err := a.UserModel.Query(ctx, schema.UserQueryParam{
		UserName: userName,
	})
	if err != nil {
		return nil, err
	} else if len(result.Data) == 0 {
		return nil, errors.ErrInvalidUserName
	}

	item := result.Data[0]
	if item.Password != util.SHA1HashString(password) {
		return nil, errors.ErrInvalidPassword
	} else if item.Status != 1 {
		return nil, errors.ErrUserDisable
	}

	return item, nil*/

	// ---------------------------------------------------------------
	// 向LDAP server验证用户信息
	// ---------------------------------------------------------------
	// 用来获取查询权限的 bind 用户。如果 ldap 禁止了匿名查询，那我们就需要先用这个帐户 bind 以下才能开始查询
	// bind 的账号通常要使用完整的 DN 信息。例如 cn=manager,dc=example,dc=org
	// 在 AD 上，则可以用诸如 mananger@example.org 的方式来 bind
	bindusername := "readonly"
	bindpassword := "password"

	l, err := DialURL("ldap://ldap.example.com:389")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// Reconnect with TLS
	// 建立 StartTLS 连接，这是建立纯文本上的 TLS 协议，允许你将非加密的通讯升级为 TLS 加密而不需要另外使用一个新的端口。
	// 邮件的 POP3 ，IMAP 也有支持类似的 StartTLS，这些都是有 RFC 的
	err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Fatal(err)
	}

	// First bind with a read only user
	// 先用我们的 bind 账号给 bind 上去
	err = l.Bind(bindusername, bindpassword)
	if err != nil {
		log.Fatal(err)
	}

	// Search for the given username
	// 这样我们就有查询权限了，可以构造查询请求了
	searchRequest := NewSearchRequest(
		"dc=example,dc=com",
		ScopeWholeSubtree, NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", userName),
		[]string{"dn"},
		nil,
	)

	// 好了现在可以搜索了，返回的是一个数组sr
	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	// 如果没有数据返回或者超过1条数据返回，这对于用户认证而言都是不允许的。
	// 前这意味着没有查到用户，后者意味着存在重复数据
	if len(sr.Entries) != 1 {
		log.Fatal("User does not exist or too many entries returned")
	}

	// 如果没有意外，那么我们就可以获取用户的实际 DN 了
	userdn := sr.Entries[0].DN

	// Bind as the user to verify their password
	// 拿这个 dn 和他的密码去做 bind 验证
	err = l.Bind(userdn, password)
	if err != nil {
		log.Fatal(err)
	}

	// Rebind as the read only user for any further queries
	// 如果后续还需要做其他操作，那么使用最初的 bind 账号重新 bind 回来。恢复初始权限。
	err = l.Bind(bindusername, bindpassword)
	if err != nil {
		log.Fatal(err)
	}

	return &schema.User{
		RecordID: userName,
		UserName: userName,
		RealName: userName,
	}, nil
}

// GenerateToken 生成令牌
func (a *Login) GenerateToken(ctx context.Context, userID string) (*schema.LoginTokenInfo, error) {
	tokenInfo, err := a.Auth.GenerateToken(ctx, userID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	item := &schema.LoginTokenInfo{
		AccessToken: tokenInfo.GetAccessToken(),
		TokenType:   tokenInfo.GetTokenType(),
		ExpiresAt:   tokenInfo.GetExpiresAt(),
	}
	return item, nil
}

// DestroyToken 销毁令牌
func (a *Login) DestroyToken(ctx context.Context, tokenString string) error {
	err := a.Auth.DestroyToken(ctx, tokenString)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (a *Login) checkAndGetUser(ctx context.Context, userID string) (*schema.User, error) {
	user, err := a.UserModel.Get(ctx, userID)
	if err != nil {
		return nil, err
	} else if user == nil {
		return nil, errors.ErrInvalidUser
	} else if user.Status != 1 {
		return nil, errors.ErrUserDisable
	}
	return user, nil
}

// GetLoginInfo 获取当前用户登录信息
func (a *Login) GetLoginInfo(ctx context.Context, userID string) (*schema.UserLoginInfo, error) {
	if isRoot := CheckIsRootUser(ctx, userID); isRoot {
		root := GetRootUser()
		loginInfo := &schema.UserLoginInfo{
			UserName: root.UserName,
			RealName: root.RealName,
		}
		return loginInfo, nil
	}

	user, err := a.checkAndGetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	info := &schema.UserLoginInfo{
		UserID:   user.RecordID,
		UserName: user.UserName,
		RealName: user.RealName,
	}

	userRoleResult, err := a.UserRoleModel.Query(ctx, schema.UserRoleQueryParam{
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	if roleIDs := userRoleResult.Data.ToRoleIDs(); len(roleIDs) > 0 {
		roleResult, err := a.RoleModel.Query(ctx, schema.RoleQueryParam{
			RecordIDs: roleIDs,
			Status:    1,
		})
		if err != nil {
			return nil, err
		}
		info.Roles = roleResult.Data
	}

	return info, nil
}

// QueryUserMenuTree 查询当前用户的权限菜单树
func (a *Login) QueryUserMenuTree(ctx context.Context, userID string) (schema.MenuTrees, error) {
	isRoot := CheckIsRootUser(ctx, userID)
	// 如果是root用户，则查询所有显示的菜单树
	if isRoot {
		result, err := a.MenuModel.Query(ctx, schema.MenuQueryParam{
			Status: 1,
		}, schema.MenuQueryOptions{
			OrderFields: schema.NewOrderFields(schema.NewOrderField("sequence", schema.OrderByDESC)),
		})
		if err != nil {
			return nil, err
		}

		menuActionResult, err := a.MenuActionModel.Query(ctx, schema.MenuActionQueryParam{})
		if err != nil {
			return nil, err
		}
		return result.Data.FillMenuAction(menuActionResult.Data.ToMenuIDMap()).ToTree(), nil
	}

	userRoleResult, err := a.UserRoleModel.Query(ctx, schema.UserRoleQueryParam{
		UserID: userID,
	})
	if err != nil {
		return nil, err
	} else if len(userRoleResult.Data) == 0 {
		return nil, errors.ErrNoPerm
	}

	roleMenuResult, err := a.RoleMenuModel.Query(ctx, schema.RoleMenuQueryParam{
		RoleIDs: userRoleResult.Data.ToRoleIDs(),
	})
	if err != nil {
		return nil, err
	} else if len(roleMenuResult.Data) == 0 {
		return nil, errors.ErrNoPerm
	}

	menuResult, err := a.MenuModel.Query(ctx, schema.MenuQueryParam{
		RecordIDs: roleMenuResult.Data.ToMenuIDs(),
		Status:    1,
	})
	if err != nil {
		return nil, err
	} else if len(menuResult.Data) == 0 {
		return nil, errors.ErrNoPerm
	}

	mData := menuResult.Data.ToMap()
	var qRecordIDs []string
	for _, pid := range menuResult.Data.SplitParentRecordIDs() {
		if _, ok := mData[pid]; !ok {
			qRecordIDs = append(qRecordIDs, pid)
		}
	}

	if len(qRecordIDs) > 0 {
		pmenuResult, err := a.MenuModel.Query(ctx, schema.MenuQueryParam{
			RecordIDs: menuResult.Data.SplitParentRecordIDs(),
		})
		if err != nil {
			return nil, err
		}
		menuResult.Data = append(menuResult.Data, pmenuResult.Data...)
	}

	sort.Sort(menuResult.Data)
	menuActionResult, err := a.MenuActionModel.Query(ctx, schema.MenuActionQueryParam{
		RecordIDs: roleMenuResult.Data.ToActionIDs(),
	})
	if err != nil {
		return nil, err
	}
	return menuResult.Data.FillMenuAction(menuActionResult.Data.ToMenuIDMap()).ToTree(), nil
}

// UpdatePassword 更新当前用户登录密码
func (a *Login) UpdatePassword(ctx context.Context, userID string, params schema.UpdatePasswordParam) error {
	if CheckIsRootUser(ctx, userID) {
		return errors.New400Response("root用户不允许更新密码")
	}

	user, err := a.checkAndGetUser(ctx, userID)
	if err != nil {
		return err
	} else if util.SHA1HashString(params.OldPassword) != user.Password {
		return errors.New400Response("旧密码不正确")
	}

	params.NewPassword = util.SHA1HashString(params.NewPassword)
	return a.UserModel.UpdatePassword(ctx, userID, params.NewPassword)
}
