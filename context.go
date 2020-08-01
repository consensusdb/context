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
	"strings"
)

/**
@author Alex Shvid
*/


type context struct {
	instances []interface{}
	beansByName map[string][]interface{}
	beansByType map[reflect.Type]interface{}
}

type injection struct {
	bean      interface{}
	value     reflect.Value
	class     reflect.Type
	fieldNum  int
	fieldName string
	fieldType reflect.Type
}

type impl struct {
	bean      interface{}
	valuePtr     reflect.Value
	classPtr     reflect.Type
	// this is a list of types visited in fields
	notImplements  []reflect.Type
}

func (t *impl) implements(ifaceType reflect.Type) bool {
	for _, ni := range t.notImplements {
		if ni == ifaceType {
			return false
		}
	}
	return t.classPtr.Implements(ifaceType)
}

func (t *injection) inject(ref *impl) {
	field := t.value.Field(t.fieldNum)
	field.Set(ref.valuePtr)
}

func (t *injection) String() string {
	return fmt.Sprintf(" %v$%s ", t.class, t.fieldName)
}

func Create(scan... interface{}) (Context, error) {

	beansByName := make(map[string][]interface{})
	beansByType := make(map[reflect.Type]interface{})

	instances := make(map[reflect.Type]*impl)
	pointers := make(map[reflect.Type][]*injection)
	interfaces := make(map[reflect.Type][]*injection)

	// scan
	for i, instance := range scan {
		if instance == nil {
			return nil, errors.Errorf("null instances are not allowed, position %d", i)
		}
		classPtr := reflect.TypeOf(instance)
		if classPtr.Kind() != reflect.Ptr {
			return nil, errors.Errorf("non-pointer instances are not allowed, position %d, type %v", i, classPtr)
		}
		if already, ok := instances[classPtr]; ok {
			return nil, errors.Errorf("repeated instance in pos %d with type %v, visited %v", i, classPtr, already)
		}
		var notImplements []reflect.Type
		valuePtr := reflect.ValueOf(instance)
		value := valuePtr.Elem()
		class := classPtr.Elem()
		for j := 0; j < class.NumField(); j++ {
			field := class.Field(j)
			if field.Name == field.Type.Name() {
				notImplements = append(notImplements, field.Type)
			}
			if field.Tag == "inject" {
				ic := &injection {
					bean: instance,
					value: value,
					class: class,
					fieldNum: j,
					fieldName: field.Name,
					fieldType: field.Type,
				}
				switch field.Type.Kind() {
				case reflect.Ptr:
					pointers[field.Type] = append(pointers[field.Type], ic)
				case reflect.Interface:
					interfaces[field.Type] = append(interfaces[field.Type], ic)
				default:
					return nil, errors.Errorf("not a pointer or interface field type '%v' on position %d in %v", field.Type, i, value.Type())
				}

			}
		}
		instances[classPtr] = &impl {
			bean: instance,
			valuePtr: valuePtr,
			classPtr: classPtr,
			notImplements: notImplements,
		}
	}

	// direct match
	var found []reflect.Type
	for requiredType, injects := range pointers {
		if direct, ok := instances[requiredType]; ok {

			beansByType[requiredType] = direct.bean
			name := requiredType.String()
			beansByName[name] = append(beansByName[name], direct.bean)

			if Verbose {
				fmt.Printf("Inject %v by pointer instance %v in to %+v\n", requiredType, direct.classPtr, injects)
			}

			for _, inject := range injects {
				inject.inject(direct)
			}
			found = append(found, requiredType)
		}
	}

	if len(found) != len(pointers) {
		for _, f := range found {
			delete(pointers, f)
		}
		var out strings.Builder
		out.WriteString("can not find candidates for those types: ")
		first := true
		for requiredType, injects := range pointers {
			if !first {
				out.WriteString(";")
			}
			first = false
			out.WriteString("type '")
			out.WriteString(requiredType.String())
			out.WriteRune('\'')
			for i, inject := range injects {
				if i > 0 {
					out.WriteString(", ")
				}
				out.WriteString(" - required by ")
				out.WriteString(inject.String())
			}
		}
		return nil, errors.New(out.String())
	}

	// interface match
	for ifaceType, injects := range interfaces {

		var candidates []reflect.Type
		for serviceTyp, service := range instances {
			if service.implements(ifaceType) {
				candidates = append(candidates, serviceTyp)
			}
		}

		switch len(candidates) {
		case 0:
			return nil, errors.Errorf("not found implementation for %v, required by those injections: %v", ifaceType, injects)
		case 1:
			serviceType := candidates[0]
			service := instances[serviceType]
			for _, inject := range injects {
				inject.inject(service)
			}

			if Verbose {
				fmt.Printf("Inject %v by implementation %v in to %+v\n", ifaceType, service.classPtr, injects)
			}
			beansByType[ifaceType] = service.bean
			name := ifaceType.String()
			beansByName[name] = append(beansByName[name], service.bean)

		default:
			return nil, errors.Errorf("found two or more services implemented the same interface %v, services=%v", ifaceType, candidates)
		}
	}

	return &context {
		instances: scan,
		beansByName: beansByName,
		beansByType: beansByType,
	}, nil
}


func (t *context) Beans() []interface{} {
	return t.instances
}

func (t *context) Bean(typ reflect.Type) (interface{}, bool) {
	bean, ok := t.beansByType[typ]
	return bean, ok
}

func (t *context) Lookup(iface string) []interface{} {
	return t.beansByName[iface]
}

func (t *context) Close() error {
	var err []error
	for _, service := range t.instances {
		if c, ok := service.(Closable); ok {
			if e := c.Close(); e != nil {
				err = append(err, e)
			}
		}
	}
	switch len(err) {
	case 0:
		return nil
	case 1:
		return err[0]
	default:
		return errors.Errorf("multiple errors on close, %v", err)
	}
}
