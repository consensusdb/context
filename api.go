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

package context

import "reflect"

/**
@author Alex Shvid
 */

var Verbose = true

type Closable interface {
	Close() error
}

type Context interface {
	/**
		Closes all beans that implements this interface in the order ot scan.
	 */
	Closable

	/**
		Get list of all registered instances on creation of context with scope 'core'
	 */

	Core() []string

	/**
		Gets obj by type, that is a pointer to the structure or interface.

		Example:
			package app
			type UserService interface {
			}

			b, ok := ctx.Bean(reflect.TypeOf((*app.UserService)(nil)).Elem())
	 */

	Bean(typ reflect.Type) (bean interface{}, ok bool)


	/**
		Panic if bean not found
	 */
	MustBean(typ reflect.Type) interface{}


	/**
		Lookup registered beans in context by name.
		The name is the local package plus name of the interface, for example 'app.UserService'

		Example:
			beans := ctx.Bean("app.UserService")
	 */

	Lookup(iface string) []interface{}

	/**
		Inject fields in to the obj on runtime.
		Does not add a new obj in to the core context, so this method is only for one-time use with scope 'runtime'.

		Example:
			type requestProcessor struct {
				app.UserService  `inject`
			}

			rp := new(requestProcessor)
			ctx.Inject(rp)
			required.NotNil(t, rp.UserService)
	 */

	Inject(interface{}) error

}


