package core

import (
	"fmt"
	"reflect"
	"sync"
)

// Container is a simple dependency injection container
type Container struct {
	services map[string]interface{}
	mu       sync.RWMutex
}

// NewContainer creates a new container
func NewContainer() *Container {
	return &Container{
		services: make(map[string]interface{}),
	}
}

// Bind registers a service in the container
func (c *Container) Bind(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = service
}

// Singleton registers a service as a singleton
func (c *Container) Singleton(name string, factory func() interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already instantiated
	if _, exists := c.services[name]; exists {
		return
	}

	c.services[name] = factory()
}

// Resolve retrieves a service from the container
func (c *Container) Resolve(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	service, exists := c.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// MustResolve retrieves a service from the container and panics if not found
func (c *Container) MustResolve(name string) interface{} {
	service, err := c.Resolve(name)
	if err != nil {
		panic(err)
	}
	return service
}

// Has checks if a service is registered
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.services[name]
	return exists
}

// AutoWire attempts to automatically wire dependencies based on struct tags
func (c *Container) AutoWire(target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetType := targetValue.Elem().Type()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		fieldValue := targetValue.Elem().Field(i)

		// Check for inject tag
		if injectTag := field.Tag.Get("inject"); injectTag != "" {
			service, err := c.Resolve(injectTag)
			if err != nil {
				return fmt.Errorf("failed to resolve dependency %s for field %s: %w", injectTag, field.Name, err)
			}

			// Set the field value
			if fieldValue.CanSet() {
				fieldValue.Set(reflect.ValueOf(service))
			}
		}
	}

	return nil
}

// Clear removes all services from the container
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services = make(map[string]interface{})
}
