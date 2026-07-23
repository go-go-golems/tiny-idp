package pluginapi

import (
	"fmt"
	"sort"
)

type Registry struct {
	definitions []Definition
	byID        map[string]Definition
}

func NewRegistry(definitions ...Definition) (*Registry, error) {
	registry := &Registry{
		definitions: make([]Definition, 0, len(definitions)),
		byID:        make(map[string]Definition, len(definitions)),
	}
	sectionSlugs := map[string]string{}
	sectionPrefixes := map[string]string{}
	for index, definition := range definitions {
		if definition == nil {
			return nil, fmt.Errorf("plugin definition %d is nil", index)
		}
		descriptor := definition.Descriptor()
		if err := descriptor.Validate(); err != nil {
			return nil, fmt.Errorf("plugin definition %d: %w", index, err)
		}
		if _, duplicate := registry.byID[descriptor.ID]; duplicate {
			return nil, fmt.Errorf("duplicate plugin id %q", descriptor.ID)
		}
		section, err := definition.Section()
		if err != nil {
			return nil, fmt.Errorf("plugin %q section: %w", descriptor.ID, err)
		}
		if section == nil || section.GetSlug() == "" {
			return nil, fmt.Errorf("plugin %q section slug is required", descriptor.ID)
		}
		if owner, duplicate := sectionSlugs[section.GetSlug()]; duplicate {
			return nil, fmt.Errorf("plugin %q section slug %q collides with plugin %q", descriptor.ID, section.GetSlug(), owner)
		}
		if section.GetPrefix() == "" {
			return nil, fmt.Errorf("plugin %q section prefix is required", descriptor.ID)
		}
		if owner, duplicate := sectionPrefixes[section.GetPrefix()]; duplicate {
			return nil, fmt.Errorf("plugin %q section prefix %q collides with plugin %q", descriptor.ID, section.GetPrefix(), owner)
		}
		sectionSlugs[section.GetSlug()] = descriptor.ID
		sectionPrefixes[section.GetPrefix()] = descriptor.ID
		registry.byID[descriptor.ID] = definition
		registry.definitions = append(registry.definitions, definition)
	}
	sort.Slice(registry.definitions, func(i, j int) bool {
		return registry.definitions[i].Descriptor().ID < registry.definitions[j].Descriptor().ID
	})
	return registry, nil
}

func (r *Registry) Definitions() []Definition {
	if r == nil {
		return nil
	}
	return append([]Definition(nil), r.definitions...)
}

func (r *Registry) Definition(id string) (Definition, bool) {
	if r == nil {
		return nil, false
	}
	definition, ok := r.byID[id]
	return definition, ok
}
