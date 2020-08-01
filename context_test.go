/*
 *
 * Copyright 2020-present Arpabet, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package context_test

import (
	"fmt"
	"github.com/consensusdb/context"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"reflect"
	"strconv"
	"testing"
)

/**
@author Alex Shvid
*/

func TestCreateNil(t *testing.T) {

	ctx, err := context.Create(nil)
	require.NotNil(t, err)
	require.Nil(t, ctx)

}


func TestCreateEmpty(t *testing.T) {

	ctx, err := context.Create()
	require.Nil(t, err)
	require.NotNil(t, ctx)
	require.Equal(t, 0, len(ctx.Core()))

}

var StorageClass = reflect.TypeOf((*Storage)(nil)).Elem()
type Storage interface {
	Load(key string) string
	Store(key, value string)
}

var ConfigServiceClass = reflect.TypeOf((*ConfigService)(nil)).Elem()
type ConfigService interface {
	GetConfig(key string) string
	SetConfig(key, value string)
}

var UserServiceClass = reflect.TypeOf((*UserService)(nil)).Elem()
type UserService interface {
	GetUser(user string) string
	SaveUser(user, details string)
}

type storageImpl struct {
	Logger  *log.Logger `inject`
}

func (t *storageImpl) Load(key string) string {
	t.Logger.Printf("Load %s\n", key)
	return "true"
}
func (t *storageImpl) Store(key, value string) {
	t.Logger.Printf("Store %s, %s\n", key, value)
}

type configServiceImpl struct {
	Storage  `inject`
}

func (t *configServiceImpl) GetConfig(key string) string {
	return t.Load("config:" + key)
}

func (t *configServiceImpl) SetConfig(key, value string) {
	t.Store("config:" + key, value)
}

type userServiceImpl struct {
	Storage        `inject`
	ConfigService  `inject`
}

func (t *userServiceImpl) GetUser(user string) string {
	return t.Load("user:" + user)
}

func (t *userServiceImpl) SaveUser(user, details string) {
	if t.allowWrites() {
		t.Store("user:" + user, details)
	}
}

func (t *userServiceImpl) allowWrites() bool {
	b, err := strconv.ParseBool(t.GetConfig("allowWrites"))
	if err != nil {
		return false
	}
	return b
}

func TestCreate(t *testing.T) {

	context.Verbose = true
	logger := log.New(os.Stderr, "context: ", log.LstdFlags)

	var ctx, err = context.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		/**
		    needed to define usage of UserService in context in order to register bean name with this interface name
		 */
		&struct{ UserService `inject` }{},
	)

	require.Nil(t, err)
	require.NotNil(t, ctx)
	require.Equal(t, 4, len(ctx.Core()))

	beans := ctx.Lookup("context_test.Storage")
	require.Equal(t, 1, len(beans))
	storageInstance := beans[0].(*storageImpl)
	require.NotNil(t, storageInstance)
	require.Equal(t, storageInstance.Logger, logger)
	require.Equal(t, storageInstance, ctx.MustBean(StorageClass))

	beans = ctx.Lookup("context_test.ConfigService")
	require.Equal(t, 1, len(beans))
	configServiceInstance := beans[0].(*configServiceImpl)
	require.NotNil(t, configServiceInstance)
	require.Equal(t, configServiceInstance.Storage, storageInstance)
	require.Equal(t, configServiceInstance, ctx.MustBean(ConfigServiceClass))

	beans = ctx.Lookup("context_test.UserService")
	require.Equal(t, 1, len(beans))
	userServiceInstance := beans[0].(*userServiceImpl)
	require.NotNil(t, userServiceInstance)
	require.Equal(t, userServiceInstance.Storage, storageInstance)
	require.Equal(t, userServiceInstance.ConfigService, configServiceInstance)
	require.Equal(t, userServiceInstance, ctx.MustBean(UserServiceClass))

}

type requestScope struct {
	requestParams string   // scope `runtime`
	UserService  `inject`  // with `inject` tag it guarantees non-null instance
}

func (t *requestScope) routeAddUser(user string) {
	t.UserService.SaveUser(user, t.requestParams)
}

func TestRequest(t *testing.T) {

	context.Verbose = true
	logger := log.New(os.Stderr, "context: ", log.LstdFlags)

	var ctx, err = context.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		&struct{ UserService `inject` }{},  // could be used by runtime injects
	)
	require.Nil(t, err)

	controller := &requestScope {
		requestParams: "username=Alex",
	}

	err = ctx.Inject(controller)
	require.Nil(t, err)

	controller.routeAddUser("alex")

}

func TestMissingPointer(t *testing.T) {

	context.Verbose = true

	_, err := context.Create(
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		&struct{ UserService `inject` }{},  // could be used by runtime injects
	)
	require.NotNil(t, err)
	fmt.Printf("TestMissingPointer: %v\n", err)

}

func TestMissingInterface(t *testing.T) {

	context.Verbose = true
	logger := log.New(os.Stderr, "context: ", log.LstdFlags)

	_, err := context.Create(
		logger,
		&storageImpl{},
		&userServiceImpl{},
	)
	require.NotNil(t, err)
	fmt.Printf("TestMissingInterface: %v\n", err)

}

func TestMissingInterfaceBean(t *testing.T) {

	context.Verbose = true
	logger := log.New(os.Stderr, "context: ", log.LstdFlags)

	var ctx, err = context.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
	)
	require.Nil(t, err)

	beans := ctx.Lookup("context_test.UserService")

	/**
		No one is requested context_test.UserService in scan list, therefore no bean defined under this interface

		To define bean interface use this construction in scan list:
			&struct{ UserService `inject` }{}
	 */
	require.Equal(t, 0, len(beans))

	_, ok := ctx.Bean(UserServiceClass)
	require.False(t, ok)

}