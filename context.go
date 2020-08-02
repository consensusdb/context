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
	"sync"
)

/**
@author Alex Shvid
*/


type context struct {

	/**
		All instances scanned on creation of context.
	    No modifications on runtime.
	 */
	core map[reflect.Type]*bean

	/**
		Fast search of beans by faceType and name
	 */

	registry registry

	/**
		Cache bean descriptions for Inject calls in runtime
	 */
	runtimeCache   sync.Map  // key is reflect.Type (classPtr), value is *beanDef
}


func Create(scan... interface{}) (Context, error) {

	beansByName := make(map[string][]*bean)
	beansByType := make(map[reflect.Type]*bean)

	core := make(map[reflect.Type]*bean)
	pointers := make(map[reflect.Type][]*injection)
	interfaces := make(map[reflect.Type][]*injection)

	// scan
	for i, obj := range scan {
		if obj == nil {
			return nil, errors.Errorf("null core are not allowed on position %d", i)
		}
		classPtr := reflect.TypeOf(obj)
		if Verbose {
			fmt.Printf("Instance %v\n", classPtr)
		}
		if classPtr.Kind() != reflect.Ptr {
			return nil, errors.Errorf("non-pointer instance is not allowed on position %d of type '%v'", i, classPtr)
		}
		if already, ok := core[classPtr]; ok {
			return nil, errors.Errorf("repeated instance on position %d of type '%v' visited as '%v'", i, classPtr, already.beanDef.classPtr)
		}
		bean, err := investigate(obj, classPtr)
		if err != nil {
			return nil, err
		}
		for _, inject := range bean.beanDef.fields {
			if Verbose {
				fmt.Printf("	Field %v\n", inject.fieldType)
			}
			switch inject.fieldType.Kind() {
			case reflect.Ptr:
				pointers[inject.fieldType] = append(pointers[inject.fieldType], inject)
			case reflect.Interface:
				interfaces[inject.fieldType] = append(interfaces[inject.fieldType], inject)
			default:
				return nil, errors.Errorf("injecting not a pointer or interface on field type '%v' at position %d in %v", inject.fieldType, i, classPtr)
			}
		}
		core[classPtr] = bean
	}

	// direct match
	var found []reflect.Type
	for requiredType, injects := range pointers {
		if direct, ok := core[requiredType]; ok {

			beansByType[requiredType] = direct
			name := requiredType.String()
			beansByName[name] = append(beansByName[name], direct)

			if Verbose {
				fmt.Printf("Inject '%v' by pointer '%v' in to %+v\n", requiredType, direct.beanDef.classPtr, injects)
			}

			for _, inject := range injects {
				if err := inject.inject(direct); err != nil {
					return nil, err
				}
			}
			found = append(found, requiredType)
		}
	}

	if len(found) != len(pointers) {
		for _, f := range found {
			delete(pointers, f)
		}
		return nil, errorNoCandidates(pointers)
	}

	// interface match
	for ifaceType, injects := range interfaces {

		service, err := searchByInterface(ifaceType, core)
		if err != nil {
			return nil, errors.Errorf("%v, required by those injections: %v", err, injects)
		}

		if Verbose {
			fmt.Printf("Inject '%v' by implementation '%v' in to %+v\n", ifaceType, service.beanDef.classPtr, injects)
		}

		for _, inject := range injects {
			if err := inject.inject(service); err != nil {
				return nil, err
			}
		}

		beansByType[ifaceType] = service
		name := ifaceType.String()
		beansByName[name] = append(beansByName[name], service)
	}

	ctx := &context{
		core:        core,
	}
	ctx.registry.beansByName = beansByName
	ctx.registry.beansByType = beansByType
	return ctx, nil
}

func errorNoCandidates(pointers map[reflect.Type][]*injection) error {
	var out strings.Builder
	out.WriteString("can not find candidates for those types: [")
	first := true
	for requiredType, injects := range pointers {
		if !first {
			out.WriteString(";")
		}
		first = false
		out.WriteString("'")
		out.WriteString(requiredType.String())
		out.WriteRune('\'')
		for i, inject := range injects {
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(" required by ")
			out.WriteString(inject.String())
		}
	}
	out.WriteString("]")
	return errors.New(out.String())
}


func (t *context) Core() []reflect.Type {
	var list []reflect.Type
	for typ, _ := range t.core {
		list = append(list, typ)
	}
	return list
}

func (t *context) Bean(typ reflect.Type) (interface{}, bool) {
	if b, ok := t.getBean(typ); ok {
		return b.obj, true
	} else {
		return nil, false
	}
}

func (t *context) MustBean(typ reflect.Type) interface{} {
	if bean, ok := t.Bean(typ); ok {
		return bean
	} else {
		panic(fmt.Sprintf("bean not found %v", typ))
	}
}

func (t *context) Lookup(iface string) []interface{} {
	return t.registry.findByName(iface)
}

func (t *context) Inject(obj interface{}) error {
	if obj == nil {
		return errors.New("null obj is are not allowed")
	}
	classPtr := reflect.TypeOf(obj)
	if classPtr.Kind() != reflect.Ptr {
		return errors.Errorf("non-pointer instances are not allowed, type %v", classPtr)
	}
	if bd, err := t.cache(obj, classPtr); err != nil {
		return err
	} else {
		for _, inject := range bd.fields {
			if impl, ok := t.getBean(inject.fieldType); ok {
				if err := inject.inject(impl); err != nil {
					return err
				}
			} else {
				errors.Errorf("implementation not found for field '%s' with type '%v'",  inject.fieldName, inject.fieldType)
			}
		}
	}
	return nil
}

// multi-threading safe
func (t *context) getBean(ifaceType reflect.Type) (*bean, bool) {
	if b, ok := t.registry.findByType(ifaceType); ok {
		return b, true
	} else if b, ok := t.core[ifaceType]; ok {
		// pointer match with core
		t.registry.addBean(ifaceType, b)
		return b, true
	} else {
		b, err := searchByInterface(ifaceType, t.core)
		if err != nil {
			return nil, false
		}
		t.registry.addBean(ifaceType, b)
		return b, true
	}
}

// multi-threading safe
func (t *context) cache(instance interface{}, classPtr reflect.Type) (*beanDef, error) {
	if bd, ok := t.runtimeCache.Load(classPtr); ok {
		return bd.(*beanDef), nil
	} else {
		b, err := investigate(instance, classPtr)
		if err != nil {
			return nil, err
		}
		t.runtimeCache.Store(classPtr, b.beanDef)
		return b.beanDef, nil
	}
}

func (t *context) Close() error {
	var err []error
	for _, instance := range t.core {
		if c, ok := instance.obj.(Closable); ok {
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

func investigate(obj interface{}, classPtr reflect.Type) (*bean, error) {
	var fields []*injection
	var notImplements []reflect.Type
	valuePtr := reflect.ValueOf(obj)
	value := valuePtr.Elem()
	class := classPtr.Elem()
	for j := 0; j < class.NumField(); j++ {
		field := class.Field(j)
		if field.Anonymous {
			notImplements = append(notImplements, field.Type)
		}
		if field.Tag == "inject" {
			kind := field.Type.Kind()
			if kind != reflect.Ptr && kind != reflect.Interface {
				return nil, errors.Errorf("not a pointer or interface field type '%v' on position %d in %v", field.Type, j, classPtr)
			}
			inject := &injection {
				value:     value,
				class:     class,
				fieldNum:  j,
				fieldName: field.Name,
				fieldType: field.Type,
			}
			fields = append(fields, inject)
		}
	}
	return &bean{
		obj:           obj,
		valuePtr:      valuePtr,
		beanDef:  &beanDef{
			classPtr:      classPtr,
			notImplements: notImplements,
			fields:        fields,
		},
	}, nil
}


func searchByInterface(ifaceType reflect.Type, core map[reflect.Type]*bean) (*bean, error) {
	var candidates []reflect.Type
	for serviceTyp, service := range core {
		if service.beanDef.implements(ifaceType) {
			candidates = append(candidates, serviceTyp)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, errors.Errorf("can not find implementations for '%v' interface", ifaceType)
	case 1:
		serviceType := candidates[0]
		return core[serviceType], nil
	default:
		return nil, errors.Errorf("found two or more beans have the same interface '%v', candidates=%v", ifaceType, candidates)
	}
}