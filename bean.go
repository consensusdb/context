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

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
)

/**
@author Alex Shvid
*/


type injectionDef struct {

	/**
	Class of that struct
	*/
	class     reflect.Type
	/**
	Field number of that struct
	*/
	fieldNum  int
	/**
	Field name where injection is going to be happen
	*/
	fieldName string
	/**
	Type of the field that is going to be injected
	*/
	fieldType reflect.Type

}

type injection struct {
	/**
		Refect value of the struct where injection is going to be happen
	 */
	value     reflect.Value

	/**
		Injection information
	 */
	injectionDef  *injectionDef

}


type beanDef struct {
	/**
		Class of the pointer to the struct or interface
	 */
	classPtr reflect.Type

	/**
		Anonymous fields expose their interfaces though bean itself.
		This is confusing on injection, because this bean is an encapsulator, not an implementation.

		Skip those fields.
	 */
	notImplements  []reflect.Type

	/**
		Fields that are going to be injected
	 */
	fields        []*injectionDef
}

type bean struct {
	/**
		Instance to the bean
	 */
	obj      interface{}
	/**
		Reflect instance to the pointer or interface of the bean
	 */
	valuePtr reflect.Value
	/**
		Bean description
	 */
	beanDef  *beanDef
}


/**
	Check if bean definition can implement interface type
 */
func (t *beanDef) implements(ifaceType reflect.Type) bool {
	for _, ni := range t.notImplements {
		if ni == ifaceType {
			return false
		}
	}
	return t.classPtr.Implements(ifaceType)
}

/**
	Inject value in to the field by using reflection
 */
func (t *injection) inject(impl *bean) error {
	return t.injectionDef.inject(&t.value, impl)
}


func (t *injectionDef) inject(value *reflect.Value, impl *bean) error {
	field := value.Field(t.fieldNum)
	if field.CanSet() {
		field.Set(impl.valuePtr)
		return nil
	} else {
		return errors.Errorf("field '%s' in class '%v' is not public", t.fieldName, t.class)
	}
}

/**
	User friendly information about class and field
 */

func (t *injection) String() string {
	return t.injectionDef.String()
}

func (t *injectionDef) String() string {
	return fmt.Sprintf(" %v->%s ", t.class, t.fieldName)
}

