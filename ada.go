package lama

import (
	"fmt"
	"reflect"
	"sync"
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
		provideTyp := provideMethod.Type()
		numIn := provideTyp.NumIn()
		if numIn > 0 {
			return fmt.Errorf("provide method cannot accept parameters: %v", reflect.TypeOf(service))
		}
		numOut := provideTyp.NumOut()
		if numOut > 0 {
			values := provideMethod.Call([]reflect.Value{})
			for idx, val := range values {
				kind := val.Kind()
				out := provideTyp.Out(idx)
				name := out.String()

				if !val.IsValid() || val.IsNil() {
					return fmt.Errorf("provide value [%s] is not a valid in %v", name, reflect.TypeOf(service))
				}

				if kind == reflect.Ptr {
					if val.Elem().Type().Kind() != reflect.Struct {
						return fmt.Errorf("provide value [%s] is not a valid struct in %v", name, reflect.TypeOf(service))
					}
				} else {
					if kind != reflect.Func {
						return fmt.Errorf("provide value [%s] is not a valid pointer in %v", out.String(), reflect.TypeOf(service))
					}
				}

				s.values[val.Type()] = val
			}
		}
	}

	s.services = append(s.services, value)
	return nil
}

// Init initializes all registered services.
func (s *Ada) Init() error {
	eType := reflect.TypeOf((*error)(nil)).Elem()
	for _, srv := range s.services {
		method := srv.MethodByName("Init")
		if !method.IsValid() {
			continue
		}

		typ := method.Type()
		numIn := typ.NumIn()
		numOut := typ.NumOut()

		args := make([]reflect.Value, numIn)

		for i := 0; i < numIn; i++ {
			argType := method.Type().In(i)
			argValue, exists := s.values[argType]
			if !exists {
				return fmt.Errorf("missing dependency [%v] for service %s", argType, srv)
			}
			args[i] = argValue
		}

		returnValue := method.Call(args)
		if numOut > 0 {
			returnTyp := typ.Out(0)
			if returnTyp.AssignableTo(eType) {
				itf := returnValue[0].Interface()
				if itf != nil {
					return itf.(error)
				}
			}
		}
	}

	return nil
}

// Serve calls the Serve method of all registered services.
func (s *Ada) Serve() <-chan error {
	wg := sync.WaitGroup{}
	out := make(chan error)
	eType := reflect.TypeOf((*error)(nil)).Elem()
	ecType := reflect.TypeOf((chan error)(nil))

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
			if returnTyp.AssignableTo(eType) {
				wg.Add(1)
				itf := method.Call([]reflect.Value{})[0].Interface()
				if itf != nil {
					out <- itf.(error)
				}
				wg.Done()

			} else if returnTyp.AssignableTo(ecType) {
				wg.Add(1)
				itf := method.Call([]reflect.Value{})[0].Interface()
				if itf != nil {
					go func() {
						for err := range itf.(chan error) {
							out <- err
						}
						wg.Done()
					}()
				} else {
					wg.Done()
				}
			}
		}
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
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
