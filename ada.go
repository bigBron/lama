package lama

import (
	"fmt"
	"reflect"
)

// NewAda returns a new instance of Ada.
func NewAda() *Ada {
	return &Ada{
		services: make([]reflect.Value, 0),
		values:   make(map[reflect.Type]reflect.Value),
	}
}

type Ada struct {
	services []reflect.Value
	values   map[reflect.Type]reflect.Value
}

// Register registers one or more services to Ada.
func (s *Ada) Register(services ...interface{}) error {
	for _, srv := range services {
		err := s.Services(srv)
		if err != nil {
			return err
		}
	}
	return nil
}

// Services registers a service to Ada.
func (s *Ada) Services(service interface{}) error {
	value := reflect.ValueOf(service)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("service is not a valid pointer: %v", reflect.TypeOf(service))
	}
	if value.Elem().Type().Kind() != reflect.Struct {
		return fmt.Errorf("service is not a valid struct: %v", reflect.TypeOf(service))
	}
	if !value.IsValid() || !value.CanInterface() {
		return fmt.Errorf("service is not a valid value: %v", reflect.TypeOf(service))
	}

	provideMethod := value.MethodByName("Provide")
	if provideMethod.IsValid() {
		provide := provideMethod.Type()
		numIn := provide.NumIn()
		if numIn > 0 {
			return fmt.Errorf("provide method cannot accept parameters: %v", reflect.TypeOf(service))
		}
		numOut := provide.NumOut()
		if numOut > 0 {
			returnValues := provideMethod.Call([]reflect.Value{})
			for _, returnValue := range returnValues {
				if returnValue.Kind() != reflect.Ptr || returnValue.IsNil() {
					return fmt.Errorf("provide value is not a valid pointer: %v", reflect.TypeOf(service))
				}
				if returnValue.Elem().Type().Kind() != reflect.Struct {
					return fmt.Errorf("provide value is not a valid struct: %v", reflect.TypeOf(service))
				}
				s.values[returnValue.Type()] = returnValue
			}
		}
	}

	s.services = append(s.services, value)
	return nil
}

// Init initializes all registered services.
func (s *Ada) Init() error {
	for _, srv := range s.services {
		initMethod := srv.MethodByName("Init")
		if initMethod.IsValid() {
			numIn := initMethod.Type().NumIn()
			args := make([]reflect.Value, numIn)
			for i := 0; i < numIn; i++ {
				argType := initMethod.Type().In(i)
				argValue, exists := s.values[argType]
				if !exists {
					return fmt.Errorf("missing dependency [%v] for service %s", argType, srv)
				}
				args[i] = argValue
			}
			initMethod.Call(args)
		}
	}
	return nil
}

// Serve calls the Serve method of all registered services.
func (s *Ada) Serve() error {
	for _, srv := range s.services {
		method := srv.MethodByName("Serve")
		if !method.IsValid() {
			continue
		}

		typ := method.Type()
		numIn := typ.NumIn()
		numOut := typ.NumOut()

		if numIn == 0 && numOut == 1 {
			returnTyp := typ.Out(0)
			eType := reflect.TypeOf((*error)(nil)).Elem()

			if returnTyp.AssignableTo(eType) {
				itf := method.Call([]reflect.Value{})[0].Interface()
				if itf != nil {
					return itf.(error)
				}
			}
		}
	}
	return nil
}

// Stop calls the Stop method of all registered services.
func (s *Ada) Stop() error {
	var errList []error
	ecType := reflect.TypeOf((*error)(nil)).Elem()

	for _, srv := range s.services {
		method := srv.MethodByName("Stop")
		if !method.IsValid() {
			continue
		}

		typ := method.Type()
		numIn := typ.NumIn()
		numOut := typ.NumOut()

		if numIn == 0 && numOut == 1 && typ.Out(0).AssignableTo(ecType) {
			itf := method.Call([]reflect.Value{})[0].Interface()
			if itf != nil {
				errList = append(errList, itf.(error))
			}
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("failed to stop some services: %v", errList)
	}

	return nil
}
