package main

import (
	"crypto/tls"

	"github.com/go-ldap/ldap/v3"
)

func main() {

    // ---------------------------------------------------------------
	// 向LDAP server验证用户信息
	// ---------------------------------------------------------------
	// 用来获取查询权限的 bind 用户。如果 ldap 禁止了匿名查询，那我们就需要先用这个帐户 bind 以下才能开始查询
	// bind 的账号通常要使用完整的 DN 信息。例如 cn=manager,dc=example,dc=org
	// 在 AD 上，则可以用诸如 mananger@example.org 的方式来 bind
	bindusername := "admin"
	bindpassword := "123456"

	l, err := ldap.DialURL("ldap://localhost:389")
	if err != nil {
		panic(err)
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
		"dc=sprd,dc=com",
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

}
